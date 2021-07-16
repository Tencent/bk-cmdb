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

package metadata

import "configcenter/src/common/watch"

type SearchHostWithInnerIPOption struct {
	InnerIP string `json:"bk_host_innerip"`
	CloudID int64  `json:"bk_cloud_id"`
	// only return these fields in hosts.
	Fields []string `json:"fields"`
}

type SearchHostWithIDOption struct {
	HostID int64 `json:"bk_host_id"`
	// only return these fields in hosts.
	Fields []string `json:"fields"`
}

type ListWithIDOption struct {
	// length range is [1,500]
	IDs []int64 `json:"ids"`
	// only return these fields in hosts.
	Fields []string `json:"fields"`
}

type DeleteArchive struct {
	Oid    string      `json:"oid" bson:"oid"`
	Coll   string      `json:"coll" bson:"coll"`
	Detail interface{} `json:"detail" bson:"detail"`
}

// list hosts with page in cache, which page info is in redis cache.
// store in a zset.
type ListHostWithPage struct {
	// length range is [1,1000]
	HostIDs []int64 `json:"bk_host_ids"`
	// only return these fields in hosts.
	Fields []string `json:"fields"`
	// sort field is not used.
	// max page limit is 1000
	Page BasePage `json:"page"`
}

type WatchEventResp struct {
	BaseResp `json:",inline"`
	Data     *watch.WatchResp `json:"data"`
}
