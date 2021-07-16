/*
 * Tencent is pleased to support the open source community by making 蓝鲸 available.
 * Copyright (C) 2017-2019 THL A29 Limited, a Tencent company. All rights reserved.
 * Licensed under the MIT License (the "License"); you may not use this file except
 * in compliance with the License. You may obtain a copy of the License at
 * http://opensource.org/licenses/MIT
 * Unless required by applicable law or agreed to in writing, software distributed under
 * the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
 * either express or implied. See the License for the specific language governing permissions and
 * limitations under the License.
 */

package logics

import (
	"context"
	"fmt"
	"net/http"
	"reflect"
	"strconv"
	"strings"

	"configcenter/src/ac/iam"
	"configcenter/src/common"
	"configcenter/src/common/blog"
	"configcenter/src/common/metadata"
	"configcenter/src/common/util"
	"configcenter/src/scene_server/auth_server/sdk/operator"
	sdktypes "configcenter/src/scene_server/auth_server/sdk/types"
	"configcenter/src/scene_server/auth_server/types"
)

const (
	numericType = "numeric"
	booleanType = "boolean"
	stringType  = "string"
)

// parse filter expression to corresponding resource type's mongo query condition,
// nil means having no query condition for the resource type, and using this filter can't get any resource of this type
func (lgc *Logics) parseFilterToMongo(ctx context.Context, header http.Header, filter *operator.Policy, resourceType iam.TypeID) (map[string]interface{}, error) {
	if filter == nil || filter.Operator == "" {
		return nil, nil
	}

	op := filter.Operator

	if op == operator.Any {
		// op any means having all permissions of this resource
		return make(map[string]interface{}), nil
	}

	// parse filter which is composed of multiple sub filters
	if op == operator.And || op == operator.Or {
		content, ok := filter.Element.(*operator.Content)
		if !ok {
			return nil, fmt.Errorf("invalid policy with unknown element type: %s", reflect.TypeOf(filter.Element).String())
		}
		if content == nil || len(content.Content) == 0 {
			return nil, fmt.Errorf("filter op %s content can't be empty", op)
		}
		mongoFilters := make([]map[string]interface{}, 0)
		for _, content := range content.Content {
			mongoFilter, err := lgc.parseFilterToMongo(ctx, header, content, resourceType)
			if err != nil {
				return nil, err
			}
			// ignore other resource filter
			if mongoFilter != nil {
				mongoFilters = append(mongoFilters, mongoFilter)
			}
		}
		if len(mongoFilters) == 0 {
			return nil, nil
		}
		return map[string]interface{}{
			operatorMap[op]: mongoFilters,
		}, nil
	}

	// parse single attribute filter field to [ resourceType, attribute ]
	fieldValue, ok := filter.Element.(*operator.FieldValue)
	if !ok {
		return nil, fmt.Errorf("invalid policy with unknown element type: %s", reflect.TypeOf(filter.Element).String())
	}
	field := fieldValue.Field
	// if field is another resource's attribute, then the filter isn't for this resource, ignore it
	if field.Resource != string(resourceType) {
		return nil, nil
	}
	attribute := field.Attribute
	value := fieldValue.Value
	if attribute == types.IDField {
		attribute = GetResourceIDField(resourceType)
	}
	if attribute == "display_name" {
		attribute = GetResourceNameField(resourceType)
	}
	if attribute == sdktypes.IamPathKey {
		return lgc.parseIamPathToMongo(ctx, header, resourceType, op, value)
	}

	switch op {
	case operator.Equal, operator.NEqual:
		if getValueType(value) == "" {
			return nil, fmt.Errorf("filter op %s value %#v isn't string, numeric or boolean type", op, value)
		}
		return map[string]interface{}{
			attribute: map[string]interface{}{
				operatorMap[op]: value,
			},
		}, nil
	case operator.In, operator.Nin:
		valueArr, ok := value.([]interface{})
		if !ok || len(valueArr) == 0 {
			return nil, fmt.Errorf("filter op %s value %#v isn't array type or is empty", op, value)
		}
		valueType := getValueType(valueArr[0])
		if valueType == "" {
			return nil, fmt.Errorf("filter op %s value %#v isn't string, numeric or boolean array type", op, value)
		}
		for _, val := range valueArr {
			if getValueType(val) != valueType {
				return nil, fmt.Errorf("filter op %s value %#v contains values with different types", op, valueArr)
			}
		}
		return map[string]interface{}{
			attribute: map[string]interface{}{
				operatorMap[op]: valueArr,
			},
		}, nil
	case operator.LessThan, operator.LessThanEqual, operator.GreaterThan, operator.GreaterThanEqual:
		if !util.IsNumeric(value) {
			return nil, fmt.Errorf("filter op %s value %#v isn't numeric type", op, value)
		}
		return map[string]interface{}{
			attribute: map[string]interface{}{
				operatorMap[op]: value,
			},
		}, nil
	case operator.Contains, operator.StartWith, operator.EndWith:
		valueStr, ok := value.(string)
		if !ok {
			return nil, fmt.Errorf("filter op %s value %#v isn't string type", op, value)
		}
		return map[string]interface{}{
			attribute: map[string]interface{}{
				common.BKDBLIKE: fmt.Sprintf(operatorRegexFmtMap[op], valueStr),
			},
		}, nil
	case operator.NContains, operator.NStartWith, operator.NEndWith:
		valueStr, ok := value.(string)
		if !ok {
			return nil, fmt.Errorf("filter op %s value %#v isn't string type", op, value)
		}
		return map[string]interface{}{
			attribute: map[string]interface{}{
				common.BKDBNot: map[string]interface{}{common.BKDBLIKE: fmt.Sprintf(operatorRegexFmtMap[op], valueStr)},
			},
		}, nil
	default:
		return nil, fmt.Errorf("filter op %s not supported", op)
	}
}

// parse iam path filter expression to corresponding resource type's mongo query condition
func (lgc *Logics) parseIamPathToMongo(ctx context.Context, header http.Header, resourceType iam.TypeID, op operator.OperType, value interface{}) (map[string]interface{}, error) {
	// generate path condition
	cond := make(map[string]interface{}, 0)
	var err error
	switch op {
	case operator.Equal, operator.Contains, operator.StartWith, operator.EndWith:
		valueStr, ok := value.(string)
		if !ok {
			return nil, fmt.Errorf("filter op %s value %#v isn't string type", op, value)
		}
		cond, err = parseIamPathToMongo(valueStr, common.BKDBEQ)
		if err != nil {
			return nil, err
		}
	case operator.In:
		pathArr, ok := value.([]interface{})
		if !ok || len(pathArr) == 0 {
			return nil, fmt.Errorf("filter op %s value %#v isn't array type or is empty", op, value)
		}
		condArr := make([]map[string]interface{}, len(pathArr))
		for index, path := range pathArr {
			pathStr, ok := path.(string)
			if !ok {
				return nil, fmt.Errorf("filter op %s value %#v isn't string type", op, value)
			}
			subCond, err := parseIamPathToMongo(pathStr, common.BKDBEQ)
			if err != nil {
				return nil, err
			}
			condArr[index] = subCond
		}
		cond[common.BKDBOR] = condArr
	case operator.Nin:
		pathArr, ok := value.([]interface{})
		if !ok || len(pathArr) == 0 {
			return nil, fmt.Errorf("filter op %s value %#v isn't array type or is empty", op, value)
		}
		condArr := make([]map[string]interface{}, len(pathArr))
		for index, path := range pathArr {
			pathStr, ok := path.(string)
			if !ok {
				return nil, fmt.Errorf("filter op %s value %#v isn't string type", op, value)
			}
			subCond, err := parseIamPathToMongo(pathStr, common.BKDBNE)
			if err != nil {
				return nil, err
			}
			condArr[index] = subCond
		}
		cond[common.BKDBAND] = condArr
	case operator.NEqual, operator.NContains, operator.NStartWith, operator.NEndWith:
		valueStr, ok := value.(string)
		if !ok {
			return nil, fmt.Errorf("filter op %s value %#v isn't string type", op, value)
		}
		cond, err = parseIamPathToMongo(valueStr, common.BKDBNE)
		if err != nil {
			return nil, err
		}
	case operator.Any:
		// op any means having all permissions of this resource
		return make(map[string]interface{}), nil
	default:
		return nil, fmt.Errorf("filter op %s not supported", op)
	}

	// resources except for host has their parent id stored in their instance table(currently all resources only have one layer TODO support multiple layers if needed)
	if resourceType != iam.Host {
		return cond, nil
	}

	// get host ids by path condition from host module config table
	param := metadata.PullResourceParam{
		Collection: common.BKTableNameModuleHostConfig,
		Condition:  cond,
		Fields:     []string{common.BKHostIDField},
		Limit:      common.BKNoLimit,
	}
	res, err := lgc.CoreAPI.CoreService().Auth().SearchAuthResource(ctx, header, param)
	if err != nil {
		blog.ErrorJSON("search auth resource failed, error: %s, param: %s", err.Error(), param)
		return nil, err
	}
	if !res.Result {
		blog.ErrorJSON("search auth resource failed, error code: %s, error message: %s, param: %s", res.Code, res.ErrMsg, param)
		return nil, res.Error()
	}
	if len(res.Data.Info) == 0 {
		return nil, nil
	}
	hostIDs := make([]int64, len(res.Data.Info))
	for index, data := range res.Data.Info {
		hostID, err := util.GetInt64ByInterface(data[common.BKHostIDField])
		if err != nil {
			return nil, err
		}
		hostIDs[index] = hostID
	}
	return map[string]interface{}{
		common.BKHostIDField: map[string]interface{}{
			common.BKDBIN: hostIDs,
		},
	}, nil
}

// parse string format iam path to mongo condition
func parseIamPathToMongo(iamPath string, op string) (map[string]interface{}, error) {
	pathItemArr := strings.Split(strings.Trim(iamPath, "/"), "/")
	cond := make(map[string]interface{}, 0)

	for _, pathItem := range pathItemArr {
		typeAndID := strings.Split(pathItem, ",")
		if len(typeAndID) != 2 {
			return nil, fmt.Errorf("pathItem %s invalid", pathItem)
		}
		idStr := typeAndID[1]
		if idStr == "*" {
			continue
		}
		resourceType := iam.TypeID(typeAndID[0])
		idField := GetResourceIDField(resourceType)
		if isResourceIDStringType(resourceType) {
			cond[idField] = map[string]interface{}{
				op: idStr,
			}
			continue
		}
		id, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("id %s parse int failed, error: %s", idStr, err.Error())
		}
		cond[idField] = map[string]interface{}{
			op: id,
		}
	}
	return cond, nil
}

var (
	operatorMap = map[operator.OperType]string{
		operator.And:              common.BKDBAND,
		operator.Or:               common.BKDBOR,
		operator.Equal:            common.BKDBEQ,
		operator.NEqual:           common.BKDBNE,
		operator.In:               common.BKDBIN,
		operator.Nin:              common.BKDBNIN,
		operator.LessThan:         common.BKDBLT,
		operator.LessThanEqual:    common.BKDBLTE,
		operator.GreaterThan:      common.BKDBGT,
		operator.GreaterThanEqual: common.BKDBGTE,
	}

	operatorRegexFmtMap = map[operator.OperType]string{
		operator.Contains:   "%s",
		operator.NContains:  "%s",
		operator.StartWith:  "^%s",
		operator.NStartWith: "^%s",
		operator.EndWith:    "%s$",
		operator.NEndWith:   "%s$",
	}
)

func getValueType(value interface{}) string {
	if util.IsNumeric(value) {
		return numericType
	}
	switch value.(type) {
	case string:
		return stringType
	case bool:
		return booleanType
	}
	return ""
}

// get resource id's actual field
func GetResourceIDField(resourceType iam.TypeID) string {
	switch resourceType {
	case iam.Host:
		return common.BKHostIDField
	case iam.SysModelGroup:
		return common.BKFieldID
	case iam.SysModel:
		return common.BKFieldID
	case iam.SysInstanceModel:
		return common.BKFieldID
	case iam.SysModelEvent:
		return common.BKFieldID
	case iam.MainlineModelEvent:
		return common.BKFieldID
	case iam.SysInstance:
		return common.BKInstIDField
	case iam.SysAssociationType:
		return common.BKFieldID
	case iam.SysResourcePoolDirectory, iam.SysHostRscPoolDirectory:
		return common.BKModuleIDField
	case iam.SysCloudArea:
		return common.BKCloudIDField
	case iam.SysCloudAccount:
		return common.BKCloudAccountID
	case iam.SysCloudResourceTask:
		return common.BKCloudTaskID
	case iam.Business, iam.BusinessForHostTrans:
		return common.BKAppIDField
	case iam.BizCustomQuery, iam.BizProcessServiceTemplate, iam.BizProcessServiceCategory, iam.BizProcessServiceInstance, iam.BizSetTemplate:
		return common.BKFieldID
	//case iam.Set:
	//	return common.BKSetIDField
	//case iam.Module:
	//	return common.BKModuleIDField
	default:
		return ""
	}
}

// get resource display name's actual field
func GetResourceNameField(resourceType iam.TypeID) string {
	switch resourceType {
	case iam.Host:
		return common.BKHostInnerIPField
	case iam.SysModelGroup:
		return common.BKClassificationNameField
	case iam.SysModel, iam.SysInstanceModel, iam.SysModelEvent, iam.MainlineModelEvent:
		return common.BKObjNameField
	case iam.SysInstance:
		return common.BKInstNameField
	case iam.SysAssociationType:
		return common.AssociationKindNameField
	case iam.SysResourcePoolDirectory, iam.SysHostRscPoolDirectory:
		return common.BKModuleNameField
	case iam.SysCloudArea:
		return common.BKCloudNameField
	case iam.SysCloudAccount:
		return common.BKCloudAccountName
	case iam.SysCloudResourceTask:
		return common.BKCloudSyncTaskName
	case iam.Business, iam.BusinessForHostTrans:
		return common.BKAppNameField
	case iam.BizCustomQuery, iam.BizProcessServiceTemplate, iam.BizProcessServiceCategory, iam.BizProcessServiceInstance, iam.BizSetTemplate:
		return common.BKFieldName
	//case iam.Set:
	//	return common.BKSetNameField
	//case iam.Module:
	//	return common.BKModuleNameField
	default:
		return ""
	}
}
