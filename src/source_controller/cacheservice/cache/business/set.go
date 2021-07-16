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

type set struct {
	key   keyGenerator
	event stream.LoopInterface
	rds   redis.Client
	db    dal.DB
}

func (s *set) Run() error {

	// initialize set token handler key.
	handler := newTokenHandler(s.key)
	startTime, err := handler.getStartTimestamp(context.Background())
	if err != nil {
		blog.Errorf("get set cache event start at time failed, err: %v", err)
		return err
	}

	loopOpts := &types.LoopOneOptions{
		LoopOptions: types.LoopOptions{
			Name: "set_cache",
			WatchOpt: &types.WatchOptions{
				Options: types.Options{
					EventStruct: new(map[string]interface{}),
					Collection:  common.BKTableNameBaseSet,
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
			DoAdd:    s.onUpsert,
			DoUpdate: s.onUpsert,
			DoDelete: s.onDelete,
		},
	}

	return s.event.WithOne(loopOpts)
}

// onUpsert set or update set cache.
func (s *set) onUpsert(e *types.Event) bool {
	if blog.V(4) {
		blog.Infof("received set cache event, op: %s, doc: %s, rid: %s", e.OperationType, e.DocBytes, e.ID())
	}

	setID := gjson.GetBytes(e.DocBytes, common.BKSetIDField).Int()
	if setID <= 0 {
		blog.Errorf("received invalid set event, skip, op: %s, doc: %s, rid: %s",
			e.OperationType, e.DocBytes, e.ID())
		return false
	}

	// update the cache.
	err := s.rds.Set(context.Background(), s.key.detailKey(setID), string(e.DocBytes), s.key.detailExpireDuration).Err()
	if err != nil {
		blog.Errorf("update set cache failed, op: %s, doc: %s, err: %v, rid: %s",
			e.OperationType, string(e.DocBytes), err, e.ID())
		return true
	}

	return false
}

// onDelete delete set cache.
func (s *set) onDelete(e *types.Event) bool {

	filter := mapstr.MapStr{
		"coll": common.BKTableNameBaseSet,
		"oid":  e.Oid,
	}

	set := new(setArchive)
	err := s.db.Table(common.BKTableNameDelArchive).Find(filter).Fields("detail").One(context.Background(), set)
	if err != nil {
		blog.Errorf("get set del archive detail failed, err: %v, rid: %s", err, e.ID())
		if s.db.IsNotFoundError(err) {
			return false
		}
		return true
	}

	blog.Infof("received delete set %d/%s event, rid: %s", set.Detail.SetID, set.Detail.SetName, e.ID())

	// delete the cache.
	if err := s.rds.Del(context.Background(), s.key.detailKey(set.Detail.SetID)).Err(); err != nil {
		blog.Errorf("delete set cache failed, err: %v, rid: %s", err, e.ID())
		return true
	}

	return false
}
