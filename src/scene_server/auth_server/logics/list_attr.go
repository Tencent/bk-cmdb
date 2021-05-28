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
	"configcenter/src/ac/iam"
	"configcenter/src/common"
	"configcenter/src/common/blog"
	"configcenter/src/common/http/rest"
	"configcenter/src/common/metadata"
	"configcenter/src/scene_server/auth_server/types"
)

// list enumeration attributes of instance type resource
func (lgc *Logics) ListAttr(kit *rest.Kit, resourceType iam.TypeID) ([]types.AttrResource, error) {
	attrs := make([]types.AttrResource, 0)
	objID := getInstanceResourceObjID(resourceType)
	if objID == "" && !iam.IsIAMSysInstance(resourceType) {
		return attrs, nil
	}

	param := metadata.QueryCondition{
		Condition: map[string]interface{}{
			common.BKPropertyTypeField: common.FieldTypeEnum,
		},
		Fields: []string{common.BKPropertyIDField, common.BKPropertyNameField},
		Page:   metadata.BasePage{Limit: common.BKNoLimit},
	}

	if iam.IsIAMSysInstance(resourceType) {
		objID = iam.GetObjIDFromIamSysInstance(resourceType)
	}
	res, err := lgc.CoreAPI.CoreService().Model().ReadModelAttr(kit.Ctx, kit.Header, objID, &param)
	if err != nil {
		blog.ErrorJSON("read model attribute failed, error: %s, param: %s, rid: %s", err.Error(), param, kit.Rid)
		return nil, err
	}
	if !res.Result {
		blog.ErrorJSON("read model attribute failed, error code: %s, error message: %s, param: %s, rid: %s", res.Code, res.ErrMsg, param, kit.Rid)
		return nil, res.CCError()
	}
	if len(res.Data.Info) == 0 {
		return attrs, nil
	}

	for _, attr := range res.Data.Info {
		displayName := attr.PropertyName
		attrs = append(attrs, types.AttrResource{
			ID:          attr.PropertyID,
			DisplayName: displayName,
		})
	}
	return attrs, nil
}
