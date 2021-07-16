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
	"context"

	"configcenter/src/common"
	"configcenter/src/common/blog"
	"configcenter/src/common/mapstr"
	"configcenter/src/storage/dal"
	"configcenter/src/storage/dal/redis"
	"configcenter/src/storage/stream"
	"configcenter/src/storage/stream/types"

	"github.com/tidwall/gjson"
)

type module struct {
	key   keyGenerator
	event stream.LoopInterface
	rds   redis.Client
	db    dal.DB
}

func (m *module) Run() error {

	// initialize module token handler key.
	handler := newTokenHandler(m.key)
	startTime, err := handler.getStartTimestamp(context.Background())
	if err != nil {
		blog.Errorf("get module cache event start at time failed, err: %v", err)
		return err
	}

	loopOpts := &types.LoopOneOptions{
		LoopOptions: types.LoopOptions{
			Name: "module_cache",
			WatchOpt: &types.WatchOptions{
				Options: types.Options{
					EventStruct: new(map[string]interface{}),
					Collection:  common.BKTableNameBaseModule,
					// start token will be automatically set when it's running,
					// so we do not set here.
					StartAfterToken:         nil,
					StartAtTime:             startTime,
					WatchFatalErrorCallback: handler.resetWatchTokenWithTimestamp,
				},
			},
			TokenHandler: handler,
			RetryOptions: &types.RetryOptions{
				MaxRetryCount: 4,
				RetryDuration: retryDuration,
			},
		},
		EventHandler: &types.OneHandler{
			DoAdd:    m.onUpsert,
			DoUpdate: m.onUpsert,
			DoDelete: m.onDelete,
		},
	}

	return m.event.WithOne(loopOpts)
}

// onUpsert set or update module cache.
func (m *module) onUpsert(e *types.Event) bool {
	if blog.V(4) {
		blog.Infof("received module cache event, op: %s, doc: %s, rid: %s", e.OperationType, e.DocBytes, e.ID())
	}

	moduleID := gjson.GetBytes(e.DocBytes, common.BKModuleIDField).Int()
	if moduleID <= 0 {
		blog.Errorf("received invalid module event, skip, op: %s, doc: %s, rid: %s",
			e.OperationType, e.DocBytes, e.ID())
		return false
	}

	// update the cache.
	err := m.rds.Set(context.Background(), m.key.detailKey(moduleID), e.DocBytes, m.key.detailExpireDuration).Err()
	if err != nil {
		blog.Errorf("update module cache failed, op: %s, doc: %s, err: %v, rid: %s",
			e.OperationType, e.DocBytes, err, e.ID())
		return true
	}

	return false
}

// onDelete delete module cache.
func (m *module) onDelete(e *types.Event) bool {

	filter := mapstr.MapStr{
		"coll": common.BKTableNameBaseModule,
		"oid":  e.Oid,
	}

	module := new(moduleArchive)
	err := m.db.Table(common.BKTableNameDelArchive).Find(filter).Fields("detail").One(context.Background(), module)
	if err != nil {
		blog.Errorf("get module del archive detail failed, err: %v, rid: %s", err, e.ID())
		if m.db.IsNotFoundError(err) {
			return false
		}
		return true
	}

	blog.Infof("received delete module %d/%s event, rid: %s", module.Detail.ModuleID, module.Detail.ModuleName, e.ID())

	// delete the cache.
	if err := m.rds.Del(context.Background(), m.key.detailKey(module.Detail.ModuleID)).Err(); err != nil {
		blog.Errorf("delete module cache failed, err: %v, rid: %s", err, e.ID())
		return true
	}

	return false
}
