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

package flow

import (
	"context"
	"fmt"
	"strings"
	"time"

	"configcenter/src/apimachinery/discovery"
	"configcenter/src/common"
	"configcenter/src/common/blog"
	"configcenter/src/common/json"
	"configcenter/src/common/metadata"
	types2 "configcenter/src/common/types"
	"configcenter/src/common/watch"
	"configcenter/src/source_controller/cacheservice/event"
	"configcenter/src/storage/dal"
	"configcenter/src/storage/dal/mongo/local"
	"configcenter/src/storage/driver/redis"
	"configcenter/src/storage/stream"
	"configcenter/src/storage/stream/types"
	"configcenter/src/thirdparty/monitor"
	"configcenter/src/thirdparty/monitor/meta"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/x/bsonx"
)

type flowOptions struct {
	key      event.Key
	watch    stream.LoopInterface
	isMaster discovery.ServiceManageInterface
	watchDB  *local.Mongo
	ccDB     dal.DB
}

func newFlow(ctx context.Context, opts flowOptions) error {
	flow := Flow{
		flowOptions: opts,
		metrics:     event.InitialMetrics(opts.key.Collection(), "watch"),
	}

	return flow.RunFlow(ctx)
}

type Flow struct {
	flowOptions
	metrics      *event.EventMetrics
	tokenHandler *flowTokenHandler
}

const batchSize = 200

func (f *Flow) RunFlow(ctx context.Context) error {
	blog.Infof("start run flow for key: %s.", f.key.Namespace())

	es := make(map[string]interface{})
	watchOpts := &types.WatchOptions{
		Options: types.Options{
			EventStruct:     &es,
			Collection:      f.key.Collection(),
			StartAfterToken: nil,
		},
	}
	if f.key.Collection() == common.BKTableNameBaseHost {
		watchOpts.EventStruct = &metadata.HostMapStr{}
	}

	if f.key.Collection() == common.BKTableNameBaseInst {
		watchOpts.Collection = ""
		watchOpts.CollectionFilter = map[string]interface{}{
			common.BKDBLIKE: event.ObjInstTablePrefixRegex,
		}
	}

	f.tokenHandler = NewFlowTokenHandler(f.key, f.watchDB, f.metrics)

	startAtTime, err := f.tokenHandler.getStartWatchTime(ctx)
	if err != nil {
		blog.Errorf("get start watch time for %s failed, err: %v", f.key.Collection(), err)
		return err
	}
	watchOpts.StartAtTime = startAtTime
	watchOpts.WatchFatalErrorCallback = f.tokenHandler.resetWatchToken

	opts := &types.LoopBatchOptions{
		LoopOptions: types.LoopOptions{
			Name:         f.key.Namespace(),
			WatchOpt:     watchOpts,
			TokenHandler: f.tokenHandler,
			RetryOptions: &types.RetryOptions{
				MaxRetryCount: 10,
				RetryDuration: 1 * time.Second,
			},
		},
		EventHandler: &types.BatchHandler{
			DoBatch: f.doBatch,
		},
		BatchSize: batchSize,
	}

	if err := f.watch.WithBatch(opts); err != nil {
		blog.Errorf("run flow, but watch batch failed, err: %v", err)
		return err
	}

	return nil
}

func (f *Flow) doBatch(es []*types.Event) (retry bool) {
	eventLen := len(es)
	if eventLen == 0 {
		return false
	}

	rid := es[0].ID()
	hasError := true

	// collect event related metrics
	start := time.Now()
	defer func() {
		if retry {
			f.metrics.CollectRetryError()
		}
		if hasError {
			return
		}
		f.metrics.CollectCycleDuration(time.Since(start))
	}()

	oidDetailMap, retry, err := f.getDeleteEventDetails(es)
	if err != nil {
		blog.Errorf("get deleted event details failed, err: %v, rid: %s", err, rid)
		return retry
	}

	ids, err := f.watchDB.NextSequences(context.Background(), f.key.ChainCollection(), eventLen)
	if err != nil {
		blog.Errorf("get %s event ids failed, err: %v, rid: %s", f.key.ChainCollection(), err, rid)
		return true
	}

	chainNodes := make([]*watch.ChainNode, 0)
	oids := make([]string, eventLen)
	// process events into db chain nodes to store in db and details to store in redis
	pipe := redis.Client().Pipeline()
	lastTokenData := make(map[string]interface{})
	cursorMap := make(map[string]struct{})
	hitConflict := false
	for index, e := range es {
		// collect event's basic metrics
		f.metrics.CollectBasic(e)
		lastTokenData[common.BKTokenField] = e.Token.Data
		lastTokenData[common.BKStartAtTimeField] = e.ClusterTime

		switch e.OperationType {
		case types.Insert, types.Update, types.Replace:
			// validate the event is valid or not.
			// the invalid event will be dropped.
			if err := f.key.Validate(e.DocBytes); err != nil {
				blog.Errorf("run flow, received %s event, but got invalid event, doc: %s, oid: %s, err: %v, rid: %s",
					f.key.Collection(), e.DocBytes, e.Oid, err, rid)
				continue
			}
		case types.Delete:

			doc, exist := oidDetailMap[e.Oid]
			if !exist {
				blog.Errorf("run flow, received %s event, but got delete doc[oid: %s] detail failed, err: %v, rid: %s",
					f.key.Collection(), e.Oid, err, rid)
				continue
			}
			// update delete event detail doc bytes.
			e.DocBytes = doc

			// validate the event is valid or not.
			// the invalid event will be dropped.
			if err := f.key.Validate(doc); err != nil {
				blog.Errorf("run flow, received %s event, but got invalid event, doc: %s, oid: %s, err: %v, rid: %s",
					f.key.Collection(), e.DocBytes, e.Oid, err, rid)
				continue
			}
		case types.Invalidate:
			blog.Errorf("loop flow, received invalid event operation type, doc: %s, rid: %s", e.DocBytes, rid)
			continue
		default:
			blog.Errorf("loop flow, received unsupported event operation type: %s, doc: %s, rid: %s",
				e.OperationType, e.DocBytes, rid)
			continue
		}

		oids[index] = e.ID()
		id := ids[index]
		name := f.key.Name(e.DocBytes)
		instID := f.key.InstanceID(e.DocBytes)
		currentCursor, err := watch.GetEventCursor(f.key.Collection(), e, instID)
		if err != nil {
			blog.Errorf("get %s event cursor failed, name: %s, err: %v, oid: %s, rid: %s", f.key.Collection(), name,
				err, e.ID(), rid)

			monitor.Collect(&meta.Alarm{
				RequestID: rid,
				Type:      meta.FlowFatalError,
				Detail: fmt.Sprintf("run event flow, but get invalid %s cursor, inst id: %d, name: %s",
					f.key.Collection(), instID, name),
				Module:    types2.CC_MODULE_CACHESERVICE,
				Dimension: map[string]string{"hit_invalid_cursor": "yes"},
			})

			return false
		}

		// validate if the cursor is already exists, this is happens when the concurrent operation is very high.
		// which will generate the same operation event with same cluster time, and generate with the same cursor
		// in the end. if this happens, the last event will be used finally, and the former events with the same
		// cursor will be dropped, and it's acceptable.
		if _, exists := cursorMap[currentCursor]; exists {
			hitConflict = true
		}
		cursorMap[currentCursor] = struct{}{}

		chainNode := &watch.ChainNode{
			ID:          id,
			ClusterTime: e.ClusterTime,
			Oid:         e.Oid,
			EventType:   watch.ConvertOperateType(e.OperationType),
			Token:       e.Token.Data,
			Cursor:      currentCursor,
		}

		if instID > 0 {
			chainNode.InstanceID = instID
		}
		chainNodes = append(chainNodes, chainNode)

		detail := types.EventDetail{
			Detail:        types.JsonString(e.DocBytes),
			UpdatedFields: e.ChangeDesc.UpdatedFields,
			RemovedFields: e.ChangeDesc.RemovedFields,
		}
		detailBytes, err := json.Marshal(detail)
		if err != nil {
			blog.Errorf("run flow, %s, marshal detail failed, name: %s, detail: %+v, err: %v, oid: %s, rid: %s",
				f.key.Collection(), name, detail, err, e.ID(), rid)
			return false
		}

		// if hit cursor conflict, the former cursor node's detail will be overwrite by the later one, so it
		// is not needed to remove the overlapped cursor node's detail again.
		pipe.Set(f.key.DetailKey(currentCursor), string(detailBytes), time.Duration(f.key.TTLSeconds())*time.Second)
	}

	// if all events are invalid, set last token to the last events' token, do not need to retry for the invalid ones
	if len(chainNodes) == 0 {
		if err := f.tokenHandler.setLastWatchToken(context.Background(), lastTokenData); err != nil {
			f.metrics.CollectMongoError()
			return false
		}
		return false
	}

	// store details at first, in case those watching cmdb events read chain when details are not inserted yet
	if _, err := pipe.Exec(); err != nil {
		f.metrics.CollectRedisError()
		blog.Errorf("run flow, but insert details for %s failed, oids: %+v, err: %v, rid: %s,", f.key.Collection(),
			oids, err, rid)
		return true
	}

	if hitConflict {
		// remove the earlier chain nodes with the same cursor with a later one
		pickedChainNodes := make([]*watch.ChainNode, 0)
		conflictNodes := make([]*watch.ChainNode, 0)
		reminder := make(map[string]struct{})
		for i := len(chainNodes) - 1; i >= 0; i-- {
			chainNode := chainNodes[i]
			if _, exists := reminder[chainNode.Cursor]; exists {
				conflictNodes = append(conflictNodes, chainNode)
				// skip this event, because it has been replaced the the one later.
				continue
			}

			reminder[chainNode.Cursor] = struct{}{}
			pickedChainNodes = append(pickedChainNodes, chainNode)
		}

		// reverse the picked chain nodes to their origin order
		for i, j := 0, len(pickedChainNodes)-1; i < j; i, j = i+1, j-1 {
			pickedChainNodes[i], pickedChainNodes[j] = pickedChainNodes[j], pickedChainNodes[i]
		}

		blog.WarnJSON("got conflict cursor with chain nodes: %s, replaced with nodes: %s, rid: %s",
			conflictNodes, pickedChainNodes, rid)

		// update the chain nodes with picked chain nodes, so that we can handle them later.
		chainNodes = pickedChainNodes
	}

	retry, err = f.doInsertEvents(chainNodes, lastTokenData, rid)
	if err != nil {
		return retry
	}

	blog.Infof("insert watch event for %s success, oids: %v, rid: %s", f.key.Collection(), oids, rid)
	hasError = false
	return false
}

func (f *Flow) doInsertEvents(chainNodes []*watch.ChainNode, lastTokenData map[string]interface{}, rid string) (
	bool, error) {

	count := len(chainNodes)

	if count == 0 {
		return false, nil
	}

	watchDBClient := f.watchDB.GetDBClient()

	session, err := watchDBClient.StartSession()
	if err != nil {
		blog.Errorf("run flow, but start session failed, coll: %s, err: %v, rid: %s", f.key.Collection(), err, rid)
		return true, err
	}
	defer session.EndSession(context.Background())

	// retry insert the event node with remove the first event node,
	// which means the first one's cursor is conflicted with the former's batch operation inserted nodes.
	retryWithReduce := false

	txnErr := mongo.WithSession(context.Background(), session, func(sc mongo.SessionContext) error {
		if err = session.StartTransaction(); err != nil {
			blog.Errorf("run flow, but start transaction failed, coll: %s, err: %v, rid: %s", f.key.Collection(),
				err, rid)
			return err
		}

		if err := f.watchDB.Table(f.key.ChainCollection()).Insert(sc, chainNodes); err != nil {
			blog.ErrorJSON("run flow, but insert chain nodes for %s failed, nodes: %s, err: %v, rid: %s",
				f.key.Collection(), chainNodes, err, rid)
			f.metrics.CollectMongoError()
			_ = session.AbortTransaction(context.Background())

			if event.IsConflictError(err) {
				// set retry with reduce flag and retry later
				retryWithReduce = true
			}
			return err
		}

		lastNode := chainNodes[len(chainNodes)-1]
		lastTokenData[common.BKFieldID] = lastNode.ID
		lastTokenData[common.BKCursorField] = lastNode.Cursor
		lastTokenData[common.BKStartAtTimeField] = lastNode.ClusterTime
		if err := f.tokenHandler.setLastWatchToken(sc, lastTokenData); err != nil {
			f.metrics.CollectMongoError()
			_ = session.AbortTransaction(context.Background())
			return err
		}

		// Use context.Background() to ensure that the commit can complete successfully even if the context passed to
		// mongo.WithSession is changed to have a timeout.
		if err = session.CommitTransaction(context.Background()); err != nil {
			blog.Errorf("run flow, but commit mongo transaction failed, err: %v", err)
			f.metrics.CollectMongoError()
			return err
		}
		return nil
	})

	if txnErr != nil {
		blog.Errorf("do insert flow events failed, err: %v, rid: %s", txnErr, rid)

		rid = rid + ":" + chainNodes[0].Oid
		if retryWithReduce {
			monitor.Collect(&meta.Alarm{
				RequestID: rid,
				Type:      meta.EventFatalError,
				Detail:    fmt.Sprintf("run event flow, but got conflict %s cursor with chain nodes", f.key.Collection()),
				Module:    types2.CC_MODULE_CACHESERVICE,
				Dimension: map[string]string{"retry_conflict_nodes": "yes"},
			})

			if len(chainNodes) <= 1 {
				return false, nil
			}

			for index, reducedChainNode := range chainNodes {
				if isConflictChainNode(reducedChainNode, txnErr) {
					chainNodes = append(chainNodes[:index], chainNodes[index+1:]...)

					// need do with retry with reduce
					blog.ErrorJSON("run flow, insert %s event with reduce node %s, remain nodes: %s, rid: %s",
						f.key.Collection(), reducedChainNode, chainNodes, rid)

					return f.doInsertEvents(chainNodes, lastTokenData, rid)
				}
			}

			// when no cursor conflict node is found, discard the first node and try to insert the others
			blog.ErrorJSON("run flow, insert %s event with reduce node %s, remain nodes: %s, rid: %s",
				f.key.Collection(), chainNodes[0], chainNodes[1:], rid)

			return f.doInsertEvents(chainNodes[1:], lastTokenData, rid)
		}

		// if an error occurred, roll back and re-watch again
		return true, err
	}

	return false, nil
}

func isConflictChainNode(chainNode *watch.ChainNode, err error) bool {
	return strings.Contains(err.Error(), chainNode.Cursor) && strings.Contains(err.Error(), "index_cursor")
}

// getDeleteEventDetails get delete events' oid and related detail map from cmdb
func (f *Flow) getDeleteEventDetails(es []*types.Event) (map[string][]byte, bool, error) {
	oidDetailMap := make(map[string][]byte)

	deletedEventOidMap := make(map[string][]string, 0)
	for _, e := range es {
		if e.OperationType == types.Delete {
			deletedEventOidMap[e.Collection] = append(deletedEventOidMap[e.Collection], e.Oid)
		}
	}

	if len(deletedEventOidMap) == 0 {
		return oidDetailMap, false, nil
	}

	for collection, deletedEventOids := range deletedEventOidMap {
		filter := map[string]interface{}{
			"oid":  map[string]interface{}{common.BKDBIN: deletedEventOids},
			"coll": collection,
		}

		if collection == common.BKTableNameBaseHost {
			docs := make([]event.HostArchive, 0)
			err := f.ccDB.Table(common.BKTableNameDelArchive).Find(filter).All(context.Background(), &docs)
			if err != nil {
				f.metrics.CollectMongoError()
				blog.Errorf("get archive deleted doc for collection %s from mongodb failed, oids: %+v, err: %v",
					collection, deletedEventOids, err)
				return nil, true, err
			}

			for _, doc := range docs {
				byt, err := json.Marshal(doc.Detail)
				if err != nil {
					blog.Errorf("received delete %s event, but marshal detail to bytes failed, oid: %s, err: %v",
						collection, doc.Oid, err)
					return nil, false, err
				}
				oidDetailMap[doc.Oid] = byt
			}
		} else {
			docs := make([]bsonx.Doc, 0)
			err := f.ccDB.Table(common.BKTableNameDelArchive).Find(filter).All(context.Background(), &docs)
			if err != nil {
				f.metrics.CollectMongoError()
				blog.Errorf("get archive deleted doc for collection %s from mongodb failed, oids: %+v, err: %v",
					collection, deletedEventOids, err)
				return nil, true, err
			}

			for _, doc := range docs {
				byt, err := bson.MarshalExtJSON(doc.Lookup("detail"), false, false)
				if err != nil {
					blog.Errorf("received delete %s event, but marshal detail to bytes failed, oid: %s, err: %v",
						collection, doc.Lookup("oid").String(), err)
					return nil, false, err
				}
				oidDetailMap[doc.Lookup("oid").String()] = byt
			}
		}
	}

	return oidDetailMap, false, nil
}

var _ = types.TokenHandler(&flowTokenHandler{})

type flowTokenHandler struct {
	key     event.Key
	watchDB dal.DB
	metrics *event.EventMetrics
}

func NewFlowTokenHandler(key event.Key, watchDB dal.DB, metrics *event.EventMetrics) *flowTokenHandler {
	return &flowTokenHandler{
		key:     key,
		watchDB: watchDB,
		metrics: metrics,
	}
}

/* SetLastWatchToken do not set last watch token in the do batch action(set it after events are successfully inserted)
   when there are several masters watching db event, we use db transaction to avoid inserting duplicate data by setting
   the last token after the insertion of db chain nodes in one transaction, since we have a unique index on the cursor
   field, the later one will encounters an error when inserting nodes and roll back without setting the token and watch
   another round from the last token of the last inserted node, thus ensures the sequence of db chain nodes.
*/
func (f *flowTokenHandler) SetLastWatchToken(ctx context.Context, token string) error {
	return nil
}

// setLastWatchToken set last watch token(used after events are successfully inserted)
func (f *flowTokenHandler) setLastWatchToken(ctx context.Context, data map[string]interface{}) error {
	filter := map[string]interface{}{
		"_id": f.key.Collection(),
	}
	if err := f.watchDB.Table(common.BKTableNameWatchToken).Update(ctx, filter, data); err != nil {
		blog.Errorf("set last watch token failed, err: %v, data: %+v", err, data)
		return err
	}
	return nil
}

// GetStartWatchToken get start watch token from watch token db first, if an error occurred, get from chain db
func (f *flowTokenHandler) GetStartWatchToken(ctx context.Context) (token string, err error) {
	filter := map[string]interface{}{
		"_id": f.key.Collection(),
	}

	data := new(watch.LastChainNodeData)
	if err := f.watchDB.Table(common.BKTableNameWatchToken).Find(filter).Fields(common.BKTokenField).One(ctx, data); err != nil {
		if !f.watchDB.IsNotFoundError(err) {
			f.metrics.CollectMongoError()
			blog.ErrorJSON("run flow, but get start watch token failed, err: %v, filter: %+v", err, filter)
		}

		tailNode := new(watch.ChainNode)
		if err := f.watchDB.Table(f.key.ChainCollection()).Find(map[string]interface{}{}).Fields(common.BKTokenField).
			Sort(common.BKFieldID+":-1").One(context.Background(), tailNode); err != nil {

			if !f.watchDB.IsNotFoundError(err) {
				f.metrics.CollectMongoError()
				blog.Errorf("get last watch token from mongo failed, err: %v", err)
				return "", err
			}
			// the tail node is not exist.
			return "", nil
		}
		return tailNode.Token, nil
	}

	return data.Token, nil
}

// resetWatchToken set watch token to empty and set the start watch time to the given one for next watch
func (f *flowTokenHandler) resetWatchToken(startAtTime types.TimeStamp) error {
	data := map[string]interface{}{
		common.BKTokenField:       "",
		common.BKStartAtTimeField: startAtTime,
	}

	filter := map[string]interface{}{
		"_id": f.key.Collection(),
	}

	if err := f.watchDB.Table(common.BKTableNameWatchToken).Update(context.Background(), filter, data); err != nil {
		blog.ErrorJSON("clear watch token failed, err: %s, collection: %s, data: %s", err, f.key.Collection(), data)
		return err
	}
	return nil
}

func (f *flowTokenHandler) getStartWatchTime(ctx context.Context) (*types.TimeStamp, error) {
	filter := map[string]interface{}{
		"_id": f.key.Collection(),
	}

	data := new(watch.LastChainNodeData)
	err := f.watchDB.Table(common.BKTableNameWatchToken).Find(filter).Fields(common.BKStartAtTimeField).One(ctx, data)
	if err != nil {
		if !f.watchDB.IsNotFoundError(err) {
			f.metrics.CollectMongoError()
			blog.ErrorJSON("run flow, but get start watch time failed, err: %v, filter: %+v", err, filter)
			return nil, err
		}
		return new(types.TimeStamp), nil
	}
	return &data.StartAtTime, nil
}
