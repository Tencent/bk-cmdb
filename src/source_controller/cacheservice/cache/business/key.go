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

package business

import (
	"fmt"
	"time"

	"configcenter/src/common"
)

const (
	bizNamespace      = common.BKCacheKeyV3Prefix + "biz"
	detailTTLDuration = 30 * time.Minute
)

var bizKey = keyGenerator{
	namespace:            bizNamespace,
	name:                 bizKeyName,
	detailExpireDuration: detailTTLDuration,
}

var moduleKey = keyGenerator{
	namespace:            bizNamespace,
	name:                 moduleKeyName,
	detailExpireDuration: detailTTLDuration,
}

var setKey = keyGenerator{
	namespace:            bizNamespace,
	name:                 setKeyName,
	detailExpireDuration: detailTTLDuration,
}

type keyName string

const (
	bizKeyName    keyName = common.BKInnerObjIDApp
	setKeyName    keyName = common.BKInnerObjIDSet
	moduleKeyName keyName = common.BKInnerObjIDModule
)

func newCustomKey(objID string) *keyGenerator {
	return &keyGenerator{
		namespace:            bizNamespace,
		name:                 keyName(objID),
		detailExpireDuration: detailTTLDuration,
	}
}

type keyGenerator struct {
	namespace            string
	name                 keyName
	detailExpireDuration time.Duration
}

// resumeTokenKey is used to store the event resume token data
func (k keyGenerator) resumeTokenKey() string {
	return fmt.Sprintf("%s:%s:resume_token", k.namespace, k.name)
}

// resumeAtTimeKey is used to store the time where to resume the event stream.
func (k keyGenerator) resumeAtTimeKey() string {
	return fmt.Sprintf("%s:%s:resume_at_time", k.namespace, k.name)
}

func (k keyGenerator) detailKey(instID int64) string {
	return fmt.Sprintf("%s:%s_detail:%d", k.namespace, k.name, instID)
}
