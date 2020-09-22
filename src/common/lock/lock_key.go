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

package lock

import (
	"fmt"
)

// not duplicate allow
const (
	// CreateModelFormat create model user format
	CreateModelFormat = "coreservice:create:model:%s"

	// CreateModuleAttrFormat create model  attribute format
	CreateModuleAttrFormat = "coreservice:create:model:%s:attr:%s"

	// CheckSetTemplateSyncFormat  检测集群模板同步的状态
	CheckSetTemplateSyncFormat = "topo:settemplate:sync:status:check:%d"

	// UniqueValidTemplateFormat 用于创建/更新模型的唯一校验
	UniqueValidTemplateFormat = "cc:v3:unique_valid_lock:objID:%s"

	// TransactionRecordLockKeyTemplateFormat 用于保存当前事务中抢到的 redis 锁列表
	TransactionRecordLockKeyTemplateFormat = "cc:v3:transaction:%s"
)

// StrFormat  build  lock key format
type StrFormat string

// GetLockKey build lock key
func GetLockKey(format StrFormat, params ...interface{}) StrFormat {
	key := fmt.Sprintf(string(format), params...)
	return StrFormat(key)
}

// GetLockKeyByRid get redis lock key by rid
func GetTransactionKeyByRid(rid string) StrFormat {
	return GetLockKey(TransactionRecordLockKeyTemplateFormat, fmt.Sprintf("%s", rid))
}
