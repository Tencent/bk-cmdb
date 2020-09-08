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

package auditlog

import (
	"fmt"
	
	"configcenter/src/apimachinery/coreservice"
	"configcenter/src/common"
	"configcenter/src/common/blog"
	"configcenter/src/common/http/rest"
	"configcenter/src/common/mapstr"
	"configcenter/src/common/metadata"
)

type resourceDirAuditLog struct {
	audit
}

// GenerateAuditLog generate audit log of resource directory, if data is nil, will auto get data by instModuleID.
func (h *resourceDirAuditLog) GenerateAuditLog(kit *rest.Kit, action metadata.ActionType, instModuleID, bizID int64, OperateFrom metadata.OperateFromType,
	data mapstr.MapStr, updateFields map[string]interface{}) (*metadata.AuditLog, error) {
	if data == nil {
		//
		query := &metadata.QueryCondition{Condition: mapstr.MapStr{common.BKModuleIDField: instModuleID}}
		rsp, err := h.clientSet.Instance().ReadInstance(kit.Ctx, kit.Header, common.BKInnerObjIDModule, query)
		if err != nil {
			blog.Errorf("generate audit log of resource directory failed, failed to read resource directory, err: %v, rid: %s",
				err.Error(), kit.Rid)
			return nil, err
		}
		if rsp.Result != true {
			blog.Errorf("generate audit log of resource directory failed, failed to read resource directory, rsp code is %v, err: %s, rid: %s",
				rsp.Code, rsp.ErrMsg, kit.Rid)
			return nil, err
		}
		if len(rsp.Data.Info) <= 0 {
			blog.Errorf("generate audit log of resource directory failed, not find resource directory, rid: %s",
				kit.Rid)
			return nil, fmt.Errorf("generate audit log of resource directory failed, not find resource directory")
		}

		data = rsp.Data.Info[0]
	}

	// get resource directory name.
	moduleName, err := data.String(common.BKModuleNameField)
	if err != nil {
		return nil, err
	}

	var basicDetail *metadata.BasicContent
	switch action {
	case metadata.AuditCreate:
		basicDetail = &metadata.BasicContent{
			CurData: data,
		}
	case metadata.AuditDelete:
		basicDetail = &metadata.BasicContent{
			PreData: data,
		}
	case metadata.AuditUpdate:
		basicDetail = &metadata.BasicContent{
			PreData:      data,
			UpdateFields: updateFields,
		}
	}

	var auditLog = &metadata.AuditLog{
		AuditType:    metadata.ModelInstanceType,
		ResourceType: metadata.ResourceDirRes,
		Action:       action,
		BusinessID:   bizID,
		ResourceID:   instModuleID,
		ResourceName: moduleName,
		OperateFrom:  OperateFrom,
		OperationDetail: &metadata.InstanceOpDetail{
			BasicOpDetail: metadata.BasicOpDetail{
				Details: basicDetail,
			},
			ModelID: common.BKInnerObjIDModule,
		},
	}

	return auditLog, nil
}

func NewReSourceDirAuditLog(clientSet coreservice.CoreServiceClientInterface) *resourceDirAuditLog {
	return &resourceDirAuditLog{
		audit: audit{
			clientSet: clientSet,
		},
	}
}
