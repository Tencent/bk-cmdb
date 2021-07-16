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

package event

import (
	"fmt"

	"configcenter/src/common"

	"github.com/tidwall/gjson"
)

const watchCacheNamespace = common.BKCacheKeyV3Prefix + "watch:"

var hostFields = []string{common.BKHostIDField, common.BKHostInnerIPField, common.BKCloudIDField}

var HostKey = Key{
	namespace:  watchCacheNamespace + "host",
	collection: common.BKTableNameBaseHost,
	ttlSeconds: 6 * 60 * 60,
	validator: func(doc []byte) error {
		fields := gjson.GetManyBytes(doc, hostFields...)
		for idx := range hostFields {
			if !fields[idx].Exists() {
				return fmt.Errorf("field %s not exist", hostFields[idx])
			}
		}
		return nil
	},
	instName: func(doc []byte) string {
		fields := gjson.GetManyBytes(doc, hostFields...)
		return fields[1].String() + ":" + fields[2].String()
	},
	instID: func(doc []byte) int64 {
		return gjson.GetBytes(doc, common.BKHostIDField).Int()
	},
}

var ModuleHostRelationKey = Key{
	namespace:  watchCacheNamespace + "host_relation",
	collection: common.BKTableNameModuleHostConfig,
	ttlSeconds: 6 * 60 * 60,
	instName: func(doc []byte) string {
		fields := gjson.GetManyBytes(doc, "bk_module_id", "bk_host_id")
		return fmt.Sprintf("module id: %s, host id: %s", fields[0].String(), fields[1].String())
	},
}

var bizFields = []string{common.BKAppIDField, common.BKAppNameField}
var BizKey = Key{
	namespace:  watchCacheNamespace + common.BKInnerObjIDApp,
	collection: common.BKTableNameBaseApp,
	ttlSeconds: 6 * 60 * 60,
	validator: func(doc []byte) error {
		fields := gjson.GetManyBytes(doc, bizFields...)
		for idx := range bizFields {
			if !fields[idx].Exists() {
				return fmt.Errorf("field %s not exist", bizFields[idx])
			}
		}
		return nil
	},
	instName: func(doc []byte) string {
		fields := gjson.GetManyBytes(doc, bizFields...)
		return fields[1].String()
	},
	instID: func(doc []byte) int64 {
		return gjson.GetBytes(doc, common.BKAppIDField).Int()
	},
}

var setFields = []string{common.BKSetIDField, common.BKSetNameField}
var SetKey = Key{
	namespace:  watchCacheNamespace + common.BKInnerObjIDSet,
	collection: common.BKTableNameBaseSet,
	ttlSeconds: 6 * 60 * 60,
	validator: func(doc []byte) error {
		fields := gjson.GetManyBytes(doc, setFields...)
		for idx := range setFields {
			if !fields[idx].Exists() {
				return fmt.Errorf("field %s not exist", setFields[idx])
			}
		}
		return nil
	},
	instName: func(doc []byte) string {
		fields := gjson.GetManyBytes(doc, setFields...)
		return fields[1].String()
	},
	instID: func(doc []byte) int64 {
		return gjson.GetBytes(doc, common.BKSetIDField).Int()
	},
}

var moduleFields = []string{common.BKModuleIDField, common.BKModuleNameField}
var ModuleKey = Key{
	namespace:  watchCacheNamespace + common.BKInnerObjIDModule,
	collection: common.BKTableNameBaseModule,
	ttlSeconds: 6 * 60 * 60,
	validator: func(doc []byte) error {
		fields := gjson.GetManyBytes(doc, moduleFields...)
		for idx := range moduleFields {
			if !fields[idx].Exists() {
				return fmt.Errorf("field %s not exist", moduleFields[idx])
			}
		}
		return nil
	},
	instName: func(doc []byte) string {
		fields := gjson.GetManyBytes(doc, moduleFields...)
		return fields[1].String()
	},
	instID: func(doc []byte) int64 {
		return gjson.GetBytes(doc, common.BKModuleIDField).Int()
	},
}

var ObjectBaseKey = Key{
	namespace:  watchCacheNamespace + common.BKInnerObjIDObject,
	collection: common.BKTableNameBaseInst,
	ttlSeconds: 6 * 60 * 60,
	validator: func(doc []byte) error {
		field := gjson.GetBytes(doc, common.BKInstIDField)
		if !field.Exists() {
			return fmt.Errorf("field %s not exist", common.BKInstIDField)
		}

		if field.Int() <= 0 {
			return fmt.Errorf("invalid bk_inst_id: %s, should be integer type and >= 0", field.Raw)
		}

		return nil
	},
	instName: func(doc []byte) string {
		return gjson.GetBytes(doc, common.BKInstNameField).String()
	},
	instID: func(doc []byte) int64 {
		return gjson.GetBytes(doc, common.BKInstIDField).Int()
	},
}

var MainlineInstanceKey = Key{
	namespace:  watchCacheNamespace + "mainline_instance",
	collection: common.BKTableNameMainlineInstance,
	ttlSeconds: 6 * 60 * 60,
	validator: func(doc []byte) error {
		field := gjson.GetBytes(doc, common.BKInstIDField)
		if !field.Exists() {
			return fmt.Errorf("field %s not exist", common.BKInstIDField)
		}

		if field.Int() <= 0 {
			return fmt.Errorf("invalid bk_inst_id: %s, should be integer type and >= 0", field.Raw)
		}

		return nil
	},
	instName: func(doc []byte) string {
		return gjson.GetBytes(doc, common.BKInstNameField).String()
	},
	instID: func(doc []byte) int64 {
		return gjson.GetBytes(doc, common.BKInstIDField).Int()
	},
}

var processFields = []string{common.BKProcessIDField, common.BKProcessNameField}
var ProcessKey = Key{
	namespace:  watchCacheNamespace + common.BKInnerObjIDProc,
	collection: common.BKTableNameBaseProcess,
	ttlSeconds: 6 * 60 * 60,
	validator: func(doc []byte) error {
		fields := gjson.GetManyBytes(doc, processFields...)
		for idx := range processFields {
			if !fields[idx].Exists() {
				return fmt.Errorf("field %s not exist", processFields[idx])
			}
		}
		return nil
	},
	instName: func(doc []byte) string {
		fields := gjson.GetManyBytes(doc, processFields...)
		return fields[1].String()
	},
	instID: func(doc []byte) int64 {
		return gjson.GetBytes(doc, common.BKProcessIDField).Int()
	},
}

var processInstanceRelationFields = []string{common.BKProcessIDField, common.BKServiceInstanceIDField, common.BKHostIDField}
var ProcessInstanceRelationKey = Key{
	namespace:  watchCacheNamespace + "process_instance_relation",
	collection: common.BKTableNameProcessInstanceRelation,
	ttlSeconds: 6 * 60 * 60,
	validator: func(doc []byte) error {
		fields := gjson.GetManyBytes(doc, processInstanceRelationFields...)
		for idx := range processInstanceRelationFields {
			if !fields[idx].Exists() {
				return fmt.Errorf("field %s not exist", processInstanceRelationFields[idx])
			}
		}
		return nil
	},
	instName: func(doc []byte) string {
		fields := gjson.GetManyBytes(doc, processInstanceRelationFields...)
		return fields[0].String()
	},
}

// this is a virtual collection name which represent for
// the mix of host, host relation, process events.
const hostIdentityWatchCollName = "cc_HostIdentityMixed"

var HostIdentityKey = Key{
	namespace:  watchCacheNamespace + "host_identity",
	collection: hostIdentityWatchCollName,
	// unused ttl seconds, details is generated directly from db.
	ttlSeconds: 6 * 60 * 60,
	validator: func(doc []byte) error {
		value := gjson.GetBytes(doc, common.BKHostIDField)
		if !value.Exists() {
			return fmt.Errorf("field %s not exist", common.BKHostIDField)
		}

		return nil
	},
	instID: func(doc []byte) int64 {
		return gjson.GetBytes(doc, common.BKHostIDField).Int()
	},
}

type Key struct {
	namespace string
	// the watching db collection name
	collection string
	// the valid event's life time.
	// if the event is exist longer than this, it will be deleted.
	// if use's watch start from value is older than time.Now().Unix() - startFrom value,
	// that means use's is watching event that has already deleted, it's not allowed.
	ttlSeconds int64

	// validator validate whether the event data is valid or not.
	// if not, then this event should not be handle, should be dropped.
	validator func(doc []byte) error

	// instance name returns a name which can describe the event's instances
	instName func(doc []byte) string

	// instID returns the event's corresponding instance id,
	instID func(doc []byte) int64
}

// Note: do not change the format, it will affect the way in event server to
// get the details with lua scripts.
func (k Key) DetailKey(cursor string) string {
	return k.namespace + ":detail:" + cursor
}

func (k Key) Namespace() string {
	return k.namespace
}

func (k Key) TTLSeconds() int64 {
	return k.ttlSeconds
}

func (k Key) Validate(doc []byte) error {
	if k.validator != nil {
		return k.validator(doc)
	}

	return nil
}

func (k Key) Name(doc []byte) string {
	if k.instName != nil {
		return k.instName(doc)
	}
	return ""
}

func (k Key) InstanceID(doc []byte) int64 {
	if k.instID != nil {
		return k.instID(doc)
	}
	return 0
}

func (k Key) Collection() string {
	return k.collection
}

// ChainCollection returns the event chain db collection name
func (k Key) ChainCollection() string {
	return k.collection + "WatchChain"
}

// ShardingCollection returns the sharding collection name. ** Can only be used for common and mainline instance **
func (k Key) ShardingCollection(objID, supplierAccount string) string {
	if k.Collection() != common.BKTableNameBaseInst && k.Collection() != common.BKTableNameMainlineInstance {
		return ""
	}

	return common.GetObjectInstTableName(objID, supplierAccount)
}
