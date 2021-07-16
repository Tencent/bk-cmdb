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

package operation

import (
	"configcenter/src/apimachinery"
	"configcenter/src/common"
	"configcenter/src/common/blog"
	"configcenter/src/common/http/rest"
	"configcenter/src/common/metadata"
)

type AuditOperationInterface interface {
	SearchAuditList(kit *rest.Kit, query metadata.QueryCondition) (int64, []metadata.AuditLog, error)
	SearchAuditDetail(kit *rest.Kit, query metadata.QueryCondition) ([]metadata.AuditLog, error)
}

// NewAuditOperation create a new inst operation instance
func NewAuditOperation(client apimachinery.ClientSetInterface) AuditOperationInterface {
	return &audit{
		clientSet: client,
	}
}

type audit struct {
	clientSet apimachinery.ClientSetInterface
}

func (a *audit) SearchAuditList(kit *rest.Kit, query metadata.QueryCondition) (int64, []metadata.AuditLog, error) {
	rsp, err := a.clientSet.CoreService().Audit().SearchAuditLog(kit.Ctx, kit.Header, query)
	if nil != err {
		blog.Errorf("search audit log list failed, error: %s, query: %s, rid: %s", err.Error(), query, kit.Rid)
		return 0, nil, err
	}

	return rsp.Data.Count, rsp.Data.Info, nil
}

func (a *audit) SearchAuditDetail(kit *rest.Kit, query metadata.QueryCondition) ([]metadata.AuditLog, error) {
	rsp, err := a.clientSet.CoreService().Audit().SearchAuditLog(kit.Ctx, kit.Header, query)
	if nil != err {
		blog.Errorf("search audit log detail list failed, error: %s, query: %s, rid: %s", err.Error(), query, kit.Rid)
		return nil, err
	}

	if len(rsp.Data.Info) == 0 {
		blog.Errorf("get no audit log detail, rid: %s", kit.Rid)
		return nil, kit.CCError.CCErrorf(common.CCErrCommParamsIsInvalid, common.BKFieldID)
	}

	return rsp.Data.Info, nil
}
