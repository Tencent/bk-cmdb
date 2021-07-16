/*
 * Tencent is pleased to support the open source community by making 蓝鲸 available.
 * Copyright (C) 2017-2018 THL A29 Limited, a Tencent company. All rights reserved.
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
	"fmt"
	"strconv"
	"strings"

	"configcenter/src/common"
	"configcenter/src/common/blog"
	"configcenter/src/common/errors"
	"configcenter/src/common/http/rest"
	"configcenter/src/common/mapstr"
	meta "configcenter/src/common/metadata"
	parse "configcenter/src/common/paraparse"
	"configcenter/src/common/util"
)

type InstNameAsst struct {
	ID         string                 `json:"id"`
	ObjID      string                 `json:"bk_obj_id"`
	ObjIcon    string                 `json:"bk_obj_icon"`
	ObjectID   int64                  `json:"bk_inst_id"`
	ObjectName string                 `json:"bk_obj_name"`
	Name       string                 `json:"bk_inst_name"`
	InstInfo   map[string]interface{} `json:"inst_info,omitempty"`
}

func (lgc *Logics) getInstAsst(kit *rest.Kit, objID string, IDs []string, query *meta.QueryInput) ([]InstNameAsst, int, errors.CCError) {
	return lgc.getRawInstAsst(kit, objID, IDs, query, false)

}

func (lgc *Logics) getInstAsstDetail(kit *rest.Kit, objID string, IDs []string, query *meta.QueryInput) ([]InstNameAsst, int, errors.CCError) {
	return lgc.getRawInstAsst(kit, objID, IDs, query, true)
}

func (lgc *Logics) getRawInstAsst(kit *rest.Kit, objID string, IDs []string, query *meta.QueryInput, isDetail bool) ([]InstNameAsst, int, errors.CCError) {
	var instName, instID string
	tmpIDs := []int{}
	for _, ID := range IDs {
		if "" == strings.TrimSpace(ID) {
			continue
		}
		tmpID, err := strconv.Atoi(ID)
		if nil != err {
			blog.Errorf("getRawInstAsst get objID(%s) inst id not integer, inst id:(%+v), rid:%s", objID, IDs, kit.Rid)
			return nil, 0, kit.CCError.Errorf(common.CCErrCommInstFieldConvertFail, objID, "association id", "int", err.Error())
		}
		tmpIDs = append(tmpIDs, tmpID)
	}
	if 0 == len(tmpIDs) {
		return make([]InstNameAsst, 0), 0, nil
	}
	condition := mapstr.New()
	if nil != query.Condition {
		newCondtion, err := mapstr.NewFromInterface(query.Condition)
		if err != nil {
			blog.Errorf("getRawInstAsst get objID(%s) inst id not integer, inst id:(%+v), rid:%s", objID, IDs, kit.Rid)
			return nil, 0, kit.CCError.Errorf(common.CCErrCommInstFieldConvertFail, objID, "query condition", "map[string]interface{}", err.Error())
		}
		condition = newCondtion
	}
	input := &meta.QueryCondition{
		Fields: strings.Split(query.Fields, ","),
		Page:   meta.BasePage{Start: query.Start, Limit: query.Limit, Sort: query.Sort},
	}
	rawObjID := objID
	switch objID {
	case common.BKInnerObjIDHost:
		instName = common.BKHostInnerIPField
		instID = common.BKHostIDField
		if 0 != len(tmpIDs) {
			condition[common.BKHostIDField] = map[string]interface{}{"$in": tmpIDs}
		}
	case common.BKInnerObjIDApp:
		instName = common.BKAppNameField
		instID = common.BKAppIDField
		if 0 != len(tmpIDs) {
			condition[common.BKAppIDField] = map[string]interface{}{"$in": tmpIDs}
		}
	case common.BKInnerObjIDSet:
		instID = common.BKSetIDField
		instName = common.BKSetNameField
		query.Sort = common.BKSetIDField
		if 0 != len(tmpIDs) {
			condition[common.BKSetIDField] = map[string]interface{}{"$in": tmpIDs}
		}
	case common.BKInnerObjIDModule:
		instID = common.BKModuleIDField
		instName = common.BKModuleNameField
		query.Sort = common.BKModuleIDField
		if 0 != len(tmpIDs) {
			condition[common.BKModuleIDField] = map[string]interface{}{"$in": tmpIDs}
		}
	case common.BKInnerObjIDPlat:
		instID = common.BKCloudIDField
		instName = common.BKCloudNameField
		query.Sort = common.BKCloudIDField
		if 0 != len(tmpIDs) {
			condition[common.BKCloudIDField] = map[string]interface{}{"$in": tmpIDs}
		}
	default:
		instName = common.BKInstNameField
		instID = common.BKInstIDField
		query.Sort = common.BKInstIDField
		condition[common.BKObjIDField] = objID
		if 0 != len(tmpIDs) {
			condition[common.BKInstIDField] = map[string]interface{}{"$in": tmpIDs}
		}
		rawObjID = objID
	}
	input.Condition = condition
	rtn, err := lgc.CoreAPI.CoreService().Instance().ReadInstance(kit.Ctx, kit.Header, rawObjID, input)
	if err != nil {
		blog.Errorf("getRawInstAsst SearchObjects http do error, err:%s,objID:%s,input:%+v,rid:%s", err.Error(), objID, input, kit.Rid)
		return nil, 0, kit.CCError.Error(common.CCErrCommHTTPDoRequestFailed)
	}
	if !rtn.Result {
		blog.Errorf("getRawInstAsst SearchObjects http reponse error, err code:%d, err msg:%s,objID:%s,input:%+v,rid:%s", rtn.Code, rtn.ErrMsg, objID, input, kit.Rid)
		return nil, 0, kit.CCError.New(rtn.Code, rtn.ErrMsg)
	}

	delarry := func(s []string, i int) []string {
		s[len(s)-1], s[i] = s[i], s[len(s)-1]
		return s[:len(s)-1]
	}

	allInst := make([]InstNameAsst, 0)
	for _, info := range rtn.Data.Info {
		if val, exist := info[instName]; exist {
			inst := InstNameAsst{}
			if name, can := val.(string); can {
				inst.Name = name
				inst.ObjID = objID
				if isDetail {
					inst.InstInfo = info
				}
			}

			if dataVal, exist := info[instID]; exist {

				itemInstID, err := util.GetInt64ByInterface(dataVal)
				if nil != err {
					blog.Errorf("not found assocte object ID %s from %v, rid: %s", instID, info, kit.Rid)
					return nil, 0, fmt.Errorf("not found assocte object ID %s from %v", instID, info)
				}
				if 0 != len(IDs) {
					for idx, key := range IDs {
						if key == strconv.FormatInt(itemInstID, 10) {
							inst.ID = IDs[idx]
							inst.ObjectID, _ = util.GetInt64ByInterface(IDs[idx])
							IDs = delarry(IDs, idx)
							allInst = append(allInst, inst)
							goto next
						}
					}
				} else {
					inst.ID = strconv.FormatInt(itemInstID, 10)
					inst.ObjectID = itemInstID
					allInst = append(allInst, inst)
				}

			next:
			}
		}
	}

	// get the InstName name
	for _, ID := range IDs {
		allInst = append(allInst, InstNameAsst{ID: ID})
	}

	return allInst, rtn.Data.Count, nil
}

func (lgc *Logics) SearchInstance(kit *rest.Kit, objID string, input *meta.QueryCondition) ([]mapstr.MapStr,
	errors.CCErrorCoder) {

	instanceRes, err := lgc.CoreAPI.CoreService().Instance().ReadInstance(kit.Ctx, kit.Header, objID, input)
	if err != nil {
		blog.ErrorJSON("search %s instance failed, err: %s, input: %s, rid: %s", objID, err, input, kit.Rid)
		return nil, kit.CCError.CCError(common.CCErrCommHTTPDoRequestFailed)
	}

	if err := instanceRes.CCError(); err != nil {
		blog.ErrorJSON("search %s instance failed, err: %s, query: %s, rid: %s", objID, err, input, kit.Rid)
		return nil, err
	}

	return instanceRes.Data.Info, nil
}

// GetInstIDNameInfo get instance ids and id to name map by condition
func (lgc *Logics) GetInstIDNameInfo(kit *rest.Kit, objID string, cond mapstr.MapStr) (map[int64]string, []int64, error) {
	idField := meta.GetInstIDFieldByObjID(objID)
	nameField := meta.GetInstNameFieldName(objID)

	query := &meta.QueryCondition{
		Fields:    []string{idField, nameField},
		Condition: cond,
		Page: meta.BasePage{
			Limit: common.BKNoLimit,
		},
	}

	instances, err := lgc.SearchInstance(kit, objID, query)
	if err != nil {
		return nil, nil, err
	}

	instanceMap := make(map[int64]string)
	instanceIDs := make([]int64, 0)
	for _, instance := range instances {
		instanceID, err := instance.Int64(idField)
		if err != nil {
			blog.ErrorJSON("instance %s id is invalid, error: %s, rid: %s", instance, err, kit.Rid)
			return nil, nil, kit.CCError.CCErrorf(common.CCErrCommParamsInvalid, idField)
		}
		instanceIDs = append(instanceIDs, instanceID)

		instanceName, err := instance.String(nameField)
		if err != nil {
			blog.ErrorJSON("instance %s name is invalid, error: %s, rid: %s", instance, err, kit.Rid)
			return nil, nil, kit.CCError.CCErrorf(common.CCErrCommParamsInvalid, nameField)
		}
		instanceMap[instanceID] = instanceName
	}

	return instanceMap, instanceIDs, nil
}

// GetInstIDs get instance ids by condition items
func (lgc *Logics) GetInstIDs(kit *rest.Kit, objID string, cond []meta.ConditionItem) ([]int64, errors.CCErrorCoder) {
	if len(cond) == 0 {
		return make([]int64, 0), nil
	}

	condition := make(map[string]interface{})
	if err := parse.ParseCommonParams(cond, condition); err != nil {
		blog.ErrorJSON("parse condition item failed, err: %s, cond: %s, rid: %s", err, cond, kit.Rid)
		return nil, kit.CCError.CCErrorf(common.CCErrCommParamsInvalid, objID+"_cond")
	}

	idField := meta.GetInstIDFieldByObjID(objID)

	query := &meta.QueryCondition{
		Fields:    []string{idField},
		Condition: condition,
		Page: meta.BasePage{
			Limit: common.BKNoLimit,
		},
	}

	instances, err := lgc.SearchInstance(kit, objID, query)
	if err != nil {
		return nil, err
	}

	instanceIDs := make([]int64, 0)
	for _, instance := range instances {
		instanceID, err := instance.Int64(idField)
		if err != nil {
			blog.ErrorJSON("instance %s id is invalid, error: %s, rid: %s", instance, err, kit.Rid)
			return nil, kit.CCError.CCErrorf(common.CCErrCommParamsInvalid, idField)
		}

		if instanceID == 0 {
			continue
		}

		instanceIDs = append(instanceIDs, instanceID)
	}
	return instanceIDs, nil
}
