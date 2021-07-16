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
	"fmt"
	"strings"
	"time"

	"configcenter/src/common"
	"configcenter/src/common/blog"
	"configcenter/src/common/mapstr"
	"configcenter/src/common/util"
	"configcenter/src/storage/dal"
	"configcenter/src/storage/dal/redis"
	"configcenter/src/storage/driver/mongodb"
	drvredis "configcenter/src/storage/driver/redis"
	"configcenter/src/storage/stream"
	"configcenter/src/storage/stream/types"

	"github.com/tidwall/gjson"
)

type customLevel struct {
	rds      redis.Client
	db       dal.DB
	observer *watchObserver
	event    stream.LoopInterface
}

func (m *customLevel) Run() error {
	// initialize observer at first.
	m.observer = &watchObserver{
		observer: make(map[string]chan struct{}),
	}

	rid := util.GenerateRID()
	if err := m.runCustomLevelInstance(rid); err != nil {
		return fmt.Errorf("run mainline instance watch failed, err: %v, rid: %s", err, rid)
	}

	go m.runObserver()

	return nil
}

func (m *customLevel) runObserver() {
	blog.Infof("start run custom level object watch observer.")
	// wait for a moment and then start loop.
	time.Sleep(5 * time.Minute)
	for {
		rid := util.GenerateRID()
		blog.Infof("start run biz custom level cache observer, rid: %s", rid)
		if err := m.runCustomLevelInstance(rid); err != nil {
			blog.Errorf("run mainline instance watch failed, err: %v, rid: %s", err)
			time.Sleep(time.Minute)
			continue
		}

		blog.Infof("finished run biz custom level cache observer, rid: %s", rid)
		time.Sleep(5 * time.Minute)
	}
}

// to watch each custom level object instance's change for cache update.
func (m *customLevel) runCustomLevelInstance(rid string) error {

	relations, err := getMainlineTopology()
	if err != nil {
		blog.Errorf("get mainline topology from mongodb failed, err: %v, rid: %s", err, rid)
		return err
	}
	// rank start from biz to host
	rank := rankMainlineTopology(relations)

	// refresh topology cache.
	refreshTopologyRank(rank)

	// reconcile the custom watch
	if err := m.reconcileCustomWatch(rid, rank); err != nil {
		return err
	}

	return nil
}

// reconcileCustomWatch to check if the custom watch is already exist or not.
// if not exist, then do loop watch, otherwise, if the custom level object is
// deleted then  stop the watch.
func (m *customLevel) reconcileCustomWatch(rid string, rank []string) error {
	rankMap := make(map[string]bool)
	for _, objID := range rank {
		if objID == "biz" || objID == "module" || objID == "set" || objID == "host" {
			// skip system embed object
			continue
		}
		rankMap[objID] = true
	}

	// check if custom watch need to be stopped at first.
	for _, obj := range m.observer.getAllObjects() {
		if _, exist := rankMap[obj]; !exist {
			// a redundant watch exist, stop watch it.
			blog.Warnf("reconcile custom watch, find a redundant one with object %s, stop it now. rid: %s", obj, rid)

			// delete resume token and start time at first, because it should not be reused.
			key := newCustomKey(obj)
			pipe := m.rds.Pipeline()
			pipe.Del(key.resumeAtTimeKey())
			pipe.Del(key.resumeTokenKey())
			_, err := pipe.Exec()
			if err != nil {
				blog.Errorf("delete resume token and start time key failed, err: %v, rid: %s", err, key)
				// try next round.
				return err
			}

			// stop watch now.
			stopNotifier := m.observer.delete(obj)
			if stopNotifier != nil {
				// cancel the watch immediately.
				close(stopNotifier)
			}

		}
	}

	if len(rankMap) == 0 {
		// no business custom level exist, do nothing.
		return nil
	}

	// check if new watch is need secondly.
	for objID := range rankMap {
		if m.observer.exist(objID) {
			// already exist, check next
			continue
		}

		// object watch not exist, need to add a new watch immediately.
		stopNotifier := make(chan struct{})
		if err := m.runCustomWatch(rid, objID, stopNotifier); err != nil {
			// close the notifier channel
			close(stopNotifier)

			blog.Errorf("reconcile custom watch, run new watch with object %s, but failed, err: %v, rid: %s",
				objID, err, rid)
			return err
		}

		blog.Infof("run new custom level object: %s instance watch to cache success. rid: %s", objID, rid)

		// loop watch success, it's time to add this object watch to observer for now.
		m.observer.add(objID, stopNotifier)
	}

	return nil
}

func (m *customLevel) runCustomWatch(rid, objID string, stopNotifier chan struct{}) error {
	key := newCustomKey(objID)

	handler := newTokenHandler(*key)
	startTime, err := handler.getStartTimestamp(context.Background())
	if err != nil {
		blog.Errorf("get biz custom object %s cache event start at time failed, err: %v, rid :%s", objID, err, rid)
		return err
	}

	loopOpts := &types.LoopOneOptions{
		LoopOptions: types.LoopOptions{
			Name: fmt.Sprintf("biz_custom_obj_%s_cache", objID),
			WatchOpt: &types.WatchOptions{
				Options: types.Options{
					EventStruct: new(map[string]interface{}),
					Collection:  common.GetInstTableName(objID, common.BKDefaultOwnerID),
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
			StopNotifier: stopNotifier,
		},
		EventHandler: &types.OneHandler{
			DoAdd: func(event *types.Event) (retry bool) {
				return m.onUpsert(key, event)
			},
			DoUpdate: func(event *types.Event) (retry bool) {
				return m.onUpsert(key, event)
			},
			DoDelete: func(event *types.Event) (retry bool) {
				return m.onDelete(key, event)
			},
		},
	}

	blog.Infof("start run new custom level object: %s instance to cache with watch. rid: %s", objID, rid)
	return m.event.WithOne(loopOpts)
}

// onUpsert is to upsert the custom object instance cache.
func (m *customLevel) onUpsert(key *keyGenerator, e *types.Event) bool {
	if blog.V(4) {
		blog.Infof("received biz custom cache event, op: %s, doc: %s, rid: %s", e.OperationType, e.DocBytes, e.ID())
	}

	instID := gjson.GetBytes(e.DocBytes, common.BKInstIDField).Int()
	if instID <= 0 {
		blog.Errorf("received invalid biz custom object instance event, skip, op: %s, doc: %s, rid: %s",
			e.OperationType, e.DocBytes, e.ID())
		return false
	}

	// update the cache.
	err := m.rds.Set(context.Background(), key.detailKey(instID), e.DocBytes, key.detailExpireDuration).Err()
	if err != nil {
		blog.Errorf("update module cache failed, op: %s, doc: %s, err: %v, rid: %s",
			e.OperationType, e.DocBytes, err, e.ID())
		return true
	}

	return false
}

func (m *customLevel) onDelete(key *keyGenerator, e *types.Event) bool {
	filter := mapstr.MapStr{
		"coll": e.Collection,
		"oid":  e.Oid,
	}

	module := new(customArchive)
	err := m.db.Table(common.BKTableNameDelArchive).Find(filter).Fields("detail").One(context.Background(), module)
	if err != nil {
		blog.Errorf("get biz custom level archive detail failed, err: %v, rid: %s", err, e.ID())
		if m.db.IsNotFoundError(err) {
			return false
		}
		return true
	}

	blog.Infof("received delete custom instance %d/%s event, rid: %s", module.Detail.InstanceID,
		module.Detail.InstanceName, e.ID())

	// delete the cache.
	if err := m.rds.Del(context.Background(), key.detailKey(module.Detail.InstanceID)).Err(); err != nil {
		blog.Errorf("delete custom instance cache failed, err: %v, rid: %s", err, e.ID())
		return true
	}

	return false
}

func getMainlineTopology() ([]mainlineAssociation, error) {
	relations := make([]mainlineAssociation, 0)
	filter := mapstr.MapStr{
		common.AssociationKindIDField: common.AssociationKindMainline,
	}
	err := mongodb.Client().Table(common.BKTableNameObjAsst).Find(filter).All(context.Background(), &relations)
	if err != nil {
		blog.Errorf("get mainline topology association failed, err: %v", err)
		return nil, err
	}
	return relations, nil
}

// rankTopology is to rank the biz topology to a array, start from biz to host
func rankMainlineTopology(relations []mainlineAssociation) []string {
	rank := make([]string, 0)
	next := "biz"
	rank = append(rank, next)
	for _, relation := range relations {
		if relation.AssociateTo == next {
			rank = append(rank, relation.ObjectID)
			next = relation.ObjectID
			continue
		} else {
			for _, rel := range relations {
				if rel.AssociateTo == next {
					rank = append(rank, rel.ObjectID)
					next = rel.ObjectID
					break
				}
			}
		}
	}
	return rank
}

func refreshTopologyRank(rank []string) {
	// then set the rank to cache
	value := strings.Join(rank, ",")
	err := drvredis.Client().Set(context.Background(), topologyKey, value, detailTTLDuration).Err()
	if err != nil {
		blog.Errorf("refresh mainline topology rank, but update to cache failed, err: %v", err)
		// do not return
	}
}
