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

package watch

import (
	"errors"
	"fmt"
	"time"

	"configcenter/src/common"
	"configcenter/src/common/blog"
	"configcenter/src/common/http/rest"
	"configcenter/src/common/json"
	"configcenter/src/common/metadata"
	types2 "configcenter/src/common/types"
	"configcenter/src/common/util"
	"configcenter/src/common/watch"
	"configcenter/src/source_controller/cacheservice/event"
	"configcenter/src/source_controller/coreservice/core/host/identifier"
	"configcenter/src/storage/stream/types"
	"configcenter/src/thirdparty/monitor"
	"configcenter/src/thirdparty/monitor/meta"
)

/* eventserver watcher defines, just created base on old service/watch.go */

const (
	// timeoutWatchLoopSeconds 20s timeout
	timeoutWatchLoopSeconds = 20

	// loopInternal watch loop internal duration
	loopInternal = 250 * time.Millisecond

	// the number of events to read in one step TODO: increase this later
	eventStep = 200
)

// WatchWithStartFrom watches target resource base on timestamp.
func (c *Client) WatchWithStartFrom(kit *rest.Kit, key event.Key, opts *watch.WatchEventOptions) (
	[]*watch.WatchEventDetail, error) {

	rid := kit.Rid

	// validate start from time with key's ttl
	diff := time.Now().Unix() - opts.StartFrom
	if diff < 0 || diff > key.TTLSeconds() {
		return nil, kit.CCError.CCErrorf(common.CCErrCommParamsInvalid, "bk_start_from")
	}

	// validate if start from value is in the range or not
	tailNode, exists, err := c.getLatestEvent(kit, key)
	if err != nil {
		blog.Errorf("get head and tail targeted node detail failed, err: %v, rid: %s", err, rid)
		return nil, err
	}

	// no events
	if !exists {
		return []*watch.WatchEventDetail{{
			Cursor:    watch.NoEventCursor,
			Resource:  opts.Resource,
			EventType: "",
			Detail:    nil,
		}}, nil
	}

	// start from is ahead of the latest's event time, watch from now.
	if int64(tailNode.ClusterTime.Sec) <= opts.StartFrom {
		if !c.isNodeHitEventType(tailNode, opts.EventTypes) || !c.isNodeHitSubResource(tailNode, opts.Filter.SubResource) {
			// not matched, set to no event cursor with empty detail
			return []*watch.WatchEventDetail{{
				Cursor:    watch.NoEventCursor,
				Resource:  opts.Resource,
				EventType: "",
				Detail:    nil,
			}}, nil
		}

		detail, exists, err := c.getEventDetail(kit, tailNode, opts.Fields, key)
		if err != nil {
			blog.Errorf("get latest event detail failed, err: %v, rid: %s", err, rid)
			return nil, err
		}

		if !exists {
			return nil, kit.CCError.CCError(common.CCErrEventDetailNotExist)
		}

		event := &watch.WatchEventDetail{
			Cursor:    tailNode.Cursor,
			Resource:  opts.Resource,
			EventType: tailNode.EventType,
		}

		if detail == nil {
			// convert to a no event cursor
			event.Detail = nil
		} else {
			if len(*detail) == 0 {
				// convert to a no event cursor
				event.Detail = nil
			} else {
				event.Detail = watch.JsonString(*detail)
			}

		}

		// matched the event type.
		return []*watch.WatchEventDetail{event}, nil
	}

	// find the first node with a larger cluster time than the start from parameter
	filter := map[string]interface{}{
		common.BKClusterTimeField: map[string]interface{}{
			common.BKDBGT: metadata.Time{Time: time.Unix(opts.StartFrom, 0).Local()},
		},
	}

	node := new(watch.ChainNode)
	err = c.watchDB.Table(key.ChainCollection()).Find(filter).Sort(common.BKFieldID).One(kit.Ctx, node)
	if err != nil {
		blog.ErrorJSON("get chain node from mongo failed, err: %s, filter: %s, rid: %s", err, filter, kit.Rid)
		if !c.watchDB.IsNotFoundError(err) {
			return nil, kit.CCError.CCError(common.CCErrCommDBSelectFailed)
		}

		return []*watch.WatchEventDetail{{
			Cursor:    watch.NoEventCursor,
			Resource:  opts.Resource,
			EventType: "",
			Detail:    nil,
		}}, nil
	}

	searchOpt := &searchFollowingChainNodesOption{
		id:          node.ID,
		limit:       eventStep,
		types:       opts.EventTypes,
		key:         key,
		subResource: opts.Filter.SubResource,
	}
	nodes, err := c.searchFollowingEventChainNodesByID(kit, searchOpt)
	if err != nil {
		blog.ErrorJSON("get event failed, err: %s, rid: %s, filter: %s", err, rid, filter)
		return nil, err
	}

	// since the first node is after the start time, we need to include it in the nodes after the start time
	if c.isNodeHitEventType(node, opts.EventTypes) && c.isNodeHitSubResource(node, opts.Filter.SubResource) {
		nodes = append([]*watch.ChainNode{node}, nodes...)
	}

	// nodes has already scan to the end
	if len(nodes) == 0 {
		resp := &watch.WatchEventDetail{
			Cursor:    tailNode.Cursor,
			Resource:  opts.Resource,
			EventType: "",
			Detail:    nil,
		}
		return []*watch.WatchEventDetail{resp}, nil
	}

	// matched event has been found, get them all.
	return c.getEventDetailsWithNodes(kit, opts, nodes, key)
}

// getEventDetailsWithNodes get event details with nodes, first get from redis, then get failed ones from mongo
func (c *Client) getEventDetailsWithNodes(kit *rest.Kit, opts *watch.WatchEventOptions, hitNodes []*watch.ChainNode, key event.Key) (
	[]*watch.WatchEventDetail, error) {

	if len(hitNodes) == 0 {
		return make([]*watch.WatchEventDetail, 0), nil
	}

	if opts.Resource == watch.HostIdentifier {
		// get from db directly.
		return c.getHostIdentityEventDetailWithNodes(kit, hitNodes)
	}

	cursors := make([]string, len(hitNodes))
	for index, node := range hitNodes {
		cursors[index] = node.Cursor
	}

	details, errCursors, errCursorIndexMap, err := c.searchEventDetailsFromRedis(kit, cursors, key)
	if err != nil {
		return nil, err
	}

	if len(errCursors) == 0 {
		resp := make([]*watch.WatchEventDetail, len(details))
		for idx, detail := range details {
			jsonStr := types.GetEventDetail(&detail)
			detail = *json.CutJsonDataWithFields(jsonStr, opts.Fields)
			resp[idx] = &watch.WatchEventDetail{
				Cursor:    hitNodes[idx].Cursor,
				Resource:  opts.Resource,
				EventType: hitNodes[idx].EventType,
				Detail:    watch.JsonString(detail),
			}
		}
		return resp, nil
	}

	// get event chain nodes from db for cursors that failed when reading redis
	errCursorsExistMap := make(map[string]struct{})
	for _, errCursor := range errCursors {
		errCursorsExistMap[errCursor] = struct{}{}
	}

	errNodes := make([]*watch.ChainNode, 0)
	for _, node := range hitNodes {
		if _, exists := errCursorsExistMap[node.Cursor]; exists {
			errNodes = append(errNodes, node)
		}
	}

	indexDetailMap, err := c.searchEventDetailsFromMongo(kit, errNodes, opts.Fields, errCursorIndexMap, key)
	if err != nil {
		blog.Errorf("get details from mongo failed, err: %v, cursors: %+v, rid: %s", err, errCursors, kit.Rid)
		return nil, err
	}

	resp := make([]*watch.WatchEventDetail, len(details))
	for idx, detail := range details {
		errDetail, exists := indexDetailMap[idx]
		if exists {
			detail = errDetail
		} else {
			jsonStr := types.GetEventDetail(&detail)
			detail = *json.CutJsonDataWithFields(jsonStr, opts.Fields)
		}

		resp[idx] = &watch.WatchEventDetail{
			Cursor:    hitNodes[idx].Cursor,
			Resource:  opts.Resource,
			EventType: hitNodes[idx].EventType,
			Detail:    watch.JsonString(detail),
		}
	}
	return resp, nil
}

// get host identity from db directly
func (c *Client) getHostIdentityEventDetailWithNodes(kit *rest.Kit, hitNodes []*watch.ChainNode) (
	[]*watch.WatchEventDetail, error) {

	if len(hitNodes) == 0 {
		return nil, errors.New("no hit host identity event nodes")
	}

	hostIDs := make([]int64, 0)
	for idx := range hitNodes {
		if hitNodes[idx].InstanceID <= 0 {
			monitor.Collect(&meta.Alarm{
				RequestID: kit.Rid,
				Type:      meta.EventFatalError,
				Detail: fmt.Sprintf("host identity, instance id: %d is invalid, cursor: %s",
					hitNodes[idx].InstanceID, hitNodes[idx].Cursor),
				Module:    types2.CC_MODULE_CACHESERVICE,
				Dimension: map[string]string{"host_identifier": "yes"},
			})

			blog.ErrorJSON("get host identity with chain nodes, but got invalid host id, skip, detail: %s, rid: %s",
				hitNodes[idx], kit.Rid)
			continue
		}

		hostIDs = append(hostIDs, hitNodes[idx].InstanceID)
	}

	hostIDs = util.IntArrayUnique(hostIDs)
	// read from secondary, but this may get host identity may not same with master.
	// kit.Ctx, kit.Header = util.SetReadPreference(kit.Ctx, kit.Header, common.SecondaryPreferredMode)
	list, err := identifier.NewIdentifier().Identifier(kit, hostIDs)
	if err != nil {
		blog.Errorf("get host identity from db failed, host id: %v, err: %v, rid: %s", hostIDs, err, kit.Rid)
		return nil, err
	}

	identityMap := make(map[int64]*metadata.HostIdentifier)
	for idx := range list {
		identityMap[list[idx].HostID] = &list[idx]
	}

	details := make([]*watch.WatchEventDetail, 0)
	for _, one := range hitNodes {

		if one.InstanceID <= 0 {
			// skip
			continue
		}

		identity, exists := identityMap[one.InstanceID]
		if !exists {
			// host already be deleted, skip this event.
			continue
		}

		js, err := json.Marshal(identity)
		if err != nil {
			blog.Errorf("marshal host identity failed, skip, detail: %+v, err :%v, rid: %s", *identity, err, kit.Rid)
			continue
		}

		details = append(details, &watch.WatchEventDetail{
			Cursor:    one.Cursor,
			Resource:  watch.HostIdentifier,
			EventType: one.EventType,
			Detail:    watch.JsonString(js),
		})
	}

	if len(details) == 0 {
		// return the last node's cursor without details, so that use can watch from
		// this nodes continually.
		lastOne := hitNodes[len(hitNodes)-1]
		one := &watch.WatchEventDetail{
			Cursor:    lastOne.Cursor,
			Resource:  watch.HostIdentifier,
			EventType: lastOne.EventType,
			Detail:    nil,
		}
		return []*watch.WatchEventDetail{one}, nil
	}

	return details, nil
}

// WatchFromNow watches target resource events from now.
func (c *Client) WatchFromNow(kit *rest.Kit, key event.Key, opts *watch.WatchEventOptions) (
	*watch.WatchEventDetail, error) {

	rid := kit.Rid

	node, exists, err := c.getLatestEvent(kit, key)
	if err != nil {
		blog.Errorf("watch from now, but get latest event failed, err: %v, rid: %s", err, rid)
		return nil, err
	}

	if !exists {
		// event chain list is empty, which means no event and not be initialized.
		return &watch.WatchEventDetail{
			Cursor:    watch.NoEventCursor,
			Resource:  opts.Resource,
			EventType: "",
			Detail:    nil,
		}, nil
	}

	if !c.isNodeHitEventType(node, opts.EventTypes) || !c.isNodeHitSubResource(node, opts.Filter.SubResource) {
		// not matched, set to no event cursor with empty detail
		return &watch.WatchEventDetail{
			Cursor:    watch.NoEventCursor,
			Resource:  opts.Resource,
			EventType: "",
			Detail:    nil,
		}, nil
	}

	detail, exists, err := c.getEventDetail(kit, node, opts.Fields, key)
	if err != nil {
		blog.Errorf("watch from now, but get latest event detail failed, err: %v, rid: %s", err, rid)
		return nil, err
	}

	if !exists {
		return nil, kit.CCError.CCError(common.CCErrEventDetailNotExist)
	}

	e := &watch.WatchEventDetail{
		Cursor:    node.Cursor,
		Resource:  opts.Resource,
		EventType: node.EventType,
	}

	if detail == nil {
		// convert to a no event cursor
		e.Detail = nil
	} else {
		if len(*detail) == 0 {
			// convert to a no event cursor
			e.Detail = nil
		} else {
			e.Detail = watch.JsonString(*detail)
		}

	}

	// matched the event type.
	return e, nil
}

// watchWithCursor get events with the start cursor which is offered by user.
// it will hold the request for timeout seconds if no matched event is hit.
// if event has been hit in a round, then events will be returned immediately.
// if no events hit, then will loop the event every 200ms until timeout and return
// with a special cursor named "NoEventCursor", then we will help the user watch
// event from the head cursor.
func (c *Client) WatchWithCursor(kit *rest.Kit, key event.Key, opts *watch.WatchEventOptions) (
	[]*watch.WatchEventDetail, error) {

	rid := kit.Rid
	start := time.Now().Unix()

	searchOpt := &searchFollowingChainNodesOption{
		startCursor: opts.Cursor,
		limit:       eventStep,
		types:       opts.EventTypes,
		key:         key,
		subResource: opts.Filter.SubResource,
	}
	exists, nodes, nodeID, err := c.searchFollowingEventChainNodes(kit, searchOpt)
	if err != nil {
		blog.Errorf("search nodes after cursor %s failed, err: %v, rid: %s", opts.Cursor, err, kit.Rid)
		return nil, err
	}

	if !exists && opts.Cursor != watch.NoEventCursor {
		return nil, kit.CCError.CCError(common.CCErrEventChainNodeNotExist)
	}

	for {
		if len(nodes) != 0 {
			return c.getEventDetailsWithNodes(kit, opts, nodes, key)
		}

		// we got not even one event, sleep a little, and then try to continue the loop watch
		time.Sleep(loopInternal)
		blog.V(5).Infof("watch key: %s with resource: %s, got nothing, try next round. rid: %s",
			key.Namespace(), opts.Resource, rid)

		if time.Now().Unix()-start > timeoutWatchLoopSeconds {
			lastNode, exists, err := c.getLatestEvent(kit, key)
			if err != nil {
				blog.Errorf("watch from now, but get latest event failed, err: %v, rid: %s", err, rid)
				return nil, err
			}

			if !exists {
				// has already looped for timeout seconds, and we still got no event.
				// return with NoEventCursor and empty detail
				opts.Cursor = watch.NoEventCursor
				return []*watch.WatchEventDetail{{
					Cursor:    watch.NoEventCursor,
					Resource:  opts.Resource,
					EventType: "",
					Detail:    nil,
				}}, nil
			} else {
				resp := &watch.WatchEventDetail{
					Cursor:   lastNode.Cursor,
					Resource: opts.Resource,
					Detail:   nil,
				}

				// at least the tail node should can be scan, so something goes wrong.
				return []*watch.WatchEventDetail{resp}, nil
			}
		}

		searchOpt.id = nodeID
		nodes, err = c.searchFollowingEventChainNodesByID(kit, searchOpt)
		if err != nil {
			blog.Errorf("watch event from cursor: %s failed, err: %v, rid: %s", opts.Cursor, err, rid)
			return nil, err
		}
	}
}

func (c *Client) isNodeHitEventType(node *watch.ChainNode, types []watch.EventType) bool {
	if len(types) == 0 {
		return true
	}

	for _, eventType := range types {
		if node.EventType == eventType {
			return true
		}
	}
	return false
}

// isNodeHitSubResource check if node hit the sub resource, not specifying sub resource means matching all and is hit
// only used for common and mainline instances that contains sub resource
func (c *Client) isNodeHitSubResource(node *watch.ChainNode, subResource string) bool {
	if len(subResource) == 0 {
		return true
	}

	if node.SubResource == subResource {
		return true
	}
	return false
}
