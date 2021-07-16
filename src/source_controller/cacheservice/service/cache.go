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

package service

import (
	"strconv"
	"time"

	"configcenter/src/common"
	"configcenter/src/common/blog"
	"configcenter/src/common/http/rest"
	"configcenter/src/common/metadata"
	"configcenter/src/common/util"
	"configcenter/src/common/watch"
	"configcenter/src/source_controller/cacheservice/cache/topo_tree"
	"configcenter/src/source_controller/cacheservice/event"
)

func (s *cacheService) SearchHostWithInnerIPInCache(ctx *rest.Contexts) {
	opt := new(metadata.SearchHostWithInnerIPOption)
	if err := ctx.DecodeInto(&opt); nil != err {
		ctx.RespAutoError(err)
		return
	}
	host, err := s.cacheSet.Host.GetHostWithInnerIP(ctx.Kit.Ctx, opt)
	if err != nil {
		ctx.RespErrorCodeOnly(common.CCErrCommDBSelectFailed, "search host with inner ip in cache, but get host failed, err: %v", err)
		return
	}
	ctx.RespString(&host)
}

func (s *cacheService) SearchHostWithHostIDInCache(ctx *rest.Contexts) {
	opt := new(metadata.SearchHostWithIDOption)
	if err := ctx.DecodeInto(&opt); nil != err {
		ctx.RespAutoError(err)
		return
	}

	host, err := s.cacheSet.Host.GetHostWithID(ctx.Kit.Ctx, opt)
	if err != nil {
		ctx.RespErrorCodeOnly(common.CCErrCommDBSelectFailed, "search host with id in cache, but get host failed, err: %v", err)
		return
	}
	ctx.RespString(&host)
}

// ListHostWithHostIDInCache list hosts info from redis with host id list.
// if a host is not exist in cache and still can not find in mongodb,
// then it will not be return. so the returned array may not equal to
// the request host ids length and the sequence is also may not same.
func (s *cacheService) ListHostWithHostIDInCache(ctx *rest.Contexts) {
	opt := new(metadata.ListWithIDOption)
	if err := ctx.DecodeInto(&opt); nil != err {
		ctx.RespAutoError(err)
		return
	}

	host, err := s.cacheSet.Host.ListHostWithHostIDs(ctx.Kit.Ctx, opt)
	if err != nil {
		ctx.RespErrorCodeOnly(common.CCErrCommDBSelectFailed, "list host with id in cache, but get host failed, err: %v", err)
		return
	}
	ctx.RespStringArray(host)
}

func (s *cacheService) ListHostWithPageInCache(ctx *rest.Contexts) {
	opt := new(metadata.ListHostWithPage)
	if err := ctx.DecodeInto(&opt); nil != err {
		ctx.RespAutoError(err)
		return
	}

	cnt, host, err := s.cacheSet.Host.ListHostsWithPage(ctx.Kit.Ctx, opt)
	if err != nil {
		ctx.RespErrorCodeOnly(common.CCErrCommDBSelectFailed, "list host with id in cache, but get host failed, err: %v", err)
		return
	}
	ctx.RespCountInfoString(cnt, host)
}

// ListBusiness list business with id from cache, if not exist in cache, then get from mongodb directly.
func (s *cacheService) ListBusinessInCache(ctx *rest.Contexts) {
	opt := new(metadata.ListWithIDOption)
	if err := ctx.DecodeInto(&opt); nil != err {
		ctx.RespAutoError(err)
		return
	}

	details, err := s.cacheSet.Business.ListBusiness(ctx.Kit.Ctx, opt)
	if err != nil {
		ctx.RespErrorCodeOnly(common.CCErrCommDBSelectFailed, "list business with id in cache failed, err: %v", err)
		return
	}
	ctx.RespStringArray(details)
}

// ListModules list modules with id from cache, if not exist in cache, then get from mongodb directly.
func (s *cacheService) ListModulesInCache(ctx *rest.Contexts) {
	opt := new(metadata.ListWithIDOption)
	if err := ctx.DecodeInto(&opt); nil != err {
		ctx.RespAutoError(err)
		return
	}

	details, err := s.cacheSet.Business.ListModules(ctx.Kit.Ctx, opt)
	if err != nil {
		ctx.RespErrorCodeOnly(common.CCErrCommDBSelectFailed, "list modules with id in cache failed, err: %v", err)
		return
	}
	ctx.RespStringArray(details)
}

// ListSets list sets with id from cache, if not exist in cache, then get from mongodb directly.
func (s *cacheService) ListSetsInCache(ctx *rest.Contexts) {
	opt := new(metadata.ListWithIDOption)
	if err := ctx.DecodeInto(&opt); nil != err {
		ctx.RespAutoError(err)
		return
	}

	details, err := s.cacheSet.Business.ListSets(ctx.Kit.Ctx, opt)
	if err != nil {
		ctx.RespErrorCodeOnly(common.CCErrCommDBSelectFailed, "list sets with id in cache failed, err: %v", err)
		return
	}
	ctx.RespStringArray(details)
}

func (s *cacheService) SearchBusinessInCache(ctx *rest.Contexts) {
	bizID, err := strconv.ParseInt(ctx.Request.PathParameter(common.BKAppIDField), 10, 64)
	if err != nil {
		ctx.RespErrorCodeOnly(common.CCErrCommParamsIsInvalid, "invalid biz id")
		return
	}
	biz, err := s.cacheSet.Business.GetBusiness(ctx.Kit.Ctx, bizID)
	if err != nil {
		ctx.RespErrorCodeOnly(common.CCErrCommDBSelectFailed, "search biz with id in cache, but get biz failed, err: %v", err)
		return
	}
	ctx.RespString(&biz)
}

func (s *cacheService) SearchSetInCache(ctx *rest.Contexts) {
	setID, err := strconv.ParseInt(ctx.Request.PathParameter(common.BKSetIDField), 10, 64)
	if err != nil {
		ctx.RespErrorCodeOnly(common.CCErrCommParamsIsInvalid, "invalid set id")
		return
	}

	set, err := s.cacheSet.Business.GetSet(ctx.Kit.Ctx, setID)
	if err != nil {
		ctx.RespErrorCodeOnly(common.CCErrCommDBSelectFailed, "search set with id in cache failed, err: %v", err)
		return
	}
	ctx.RespString(&set)
}

func (s *cacheService) SearchModuleInCache(ctx *rest.Contexts) {
	moduleID, err := strconv.ParseInt(ctx.Request.PathParameter(common.BKModuleIDField), 10, 64)
	if err != nil {
		ctx.RespErrorCodeOnly(common.CCErrCommParamsIsInvalid, "invalid module id")
		return
	}

	module, err := s.cacheSet.Business.GetModuleDetail(ctx.Kit.Ctx, moduleID)
	if err != nil {
		ctx.RespErrorCodeOnly(common.CCErrCommDBSelectFailed, "search module with id in cache failed, err: %v", err)
		return
	}
	ctx.RespString(&module)
}

func (s *cacheService) SearchCustomLayerInCache(ctx *rest.Contexts) {
	objID := ctx.Request.PathParameter(common.BKObjIDField)

	instID, err := strconv.ParseInt(ctx.Request.PathParameter(common.BKInstIDField), 10, 64)
	if err != nil {
		ctx.RespErrorCodeOnly(common.CCErrCommParamsIsInvalid, "invalid instance id")
		return
	}

	inst, err := s.cacheSet.Business.GetCustomLevelDetail(ctx.Kit.Ctx, objID, ctx.Kit.SupplierAccount, instID)
	if err != nil {
		ctx.RespErrorCodeOnly(common.CCErrCommDBSelectFailed, "search custom layer with id in cache failed, err: %v", err)
		return
	}
	ctx.RespString(&inst)
}

// SearchBizTopologyNodePath is to search biz instance topology node's parent path. eg:
// from itself up to the biz instance, but not contains the node itself.
func (s *cacheService) SearchBizTopologyNodePath(ctx *rest.Contexts) {
	bizID, err := strconv.ParseInt(ctx.Request.PathParameter(common.BKAppIDField), 10, 64)
	if err != nil {
		ctx.RespErrorCodeOnly(common.CCErrCommParamsIsInvalid, "invalid biz id")
		return
	}

	if bizID <= 0 {
		ctx.RespErrorCodeOnly(common.CCErrCommParamsIsInvalid, "invalid biz id")
		return
	}

	opt := new(topo_tree.SearchNodePathOption)
	if err := ctx.DecodeInto(&opt); nil != err {
		ctx.RespAutoError(err)
		return
	}

	opt.Business = bizID

	paths, err := s.cacheSet.Tree.SearchNodePath(ctx.Kit.Ctx, opt, ctx.Kit.SupplierAccount)
	if err != nil {
		ctx.RespAutoError(err)
		return
	}

	ctx.RespEntity(paths)
}

// SearchBusinessBriefTopology search a business's brief topology from biz to module
// with only required fields.
func (s *cacheService) SearchBusinessBriefTopology(ctx *rest.Contexts) {
	biz := ctx.Request.PathParameter("biz")
	bizID, err := strconv.ParseInt(biz, 10, 64)
	if err != nil {
		ctx.RespErrorCodeOnly(common.CCErrCommParamsInvalid, "search biz topology, got invalid biz id, err: %v", err)
		return
	}

	topo, err := s.cacheSet.Topology.GetBizTopology(ctx.Kit, bizID)
	if err != nil {
		ctx.RespErrorCodeOnly(common.CCErrCommDBSelectFailed, "search biz topology, select db failed, err: %v", err)
		return
	}

	ctx.RespString(topo)
}

func (s *cacheService) WatchEvent(ctx *rest.Contexts) {
	var err error
	// sleep for a while if an error occurred to avoid others using wrong input to request too frequently
	defer func() {
		if err != nil {
			time.Sleep(500 * time.Millisecond)
		}
	}()

	options := new(watch.WatchEventOptions)
	if err = ctx.DecodeInto(&options); err != nil {
		blog.Errorf("watch event, but decode request body failed, err: %v, rid: %s", err, ctx.Kit.Rid)
		ctx.RespAutoError(err)
		return
	}

	if err = options.Validate(); err != nil {
		blog.Errorf("watch event, but got invalid request options, err: %v, rid: %s", err, ctx.Kit.Rid)
		ctx.RespAutoError(err)
		return
	}

	key, err := event.GetResourceKeyWithCursorType(options.Resource)
	if err != nil {
		blog.Errorf("watch event, but get resource key with cursor type failed, err: %v, rid: %s", err, ctx.Kit.Rid)
		ctx.RespAutoError(err)
		return
	}

	// read all data from db in case secondary node's latency causes data inconsistency
	util.SetDBReadPreference(ctx.Kit.Ctx, common.PrimaryMode)

	// watch with cursor
	if len(options.Cursor) != 0 {
		events, err := s.cacheSet.Event.WatchWithCursor(ctx.Kit, key, options)
		if err != nil {
			blog.Errorf("watch event with cursor failed, cursor: %s, err: %v, rid: %s", options.Cursor, err, ctx.Kit.Rid)
			ctx.RespAutoError(err)
			return
		}

		// if not events is hit, then we return user's cursor, so that they can watch with this cursor again.
		ctx.RespEntity(s.generateWatchEventResp(options.Cursor, options.Resource, events))
		return
	}

	// watch with start from
	if options.StartFrom != 0 {
		events, err := s.cacheSet.Event.WatchWithStartFrom(ctx.Kit, key, options)
		if err != nil {
			blog.Errorf("watch event with start from: %s failed, err: %v, rid: %s",
				time.Unix(options.StartFrom, 0).Format(time.RFC3339), err, ctx.Kit.Rid)
			ctx.RespAutoError(err)
			return
		}

		ctx.RespEntity(s.generateWatchEventResp("", options.Resource, events))
		return
	}

	// watch from now
	events, err := s.cacheSet.Event.WatchFromNow(ctx.Kit, key, options)
	if err != nil {
		blog.Errorf("watch event from now failed, err: %v, rid: %s", err, ctx.Kit.Rid)
		ctx.RespAutoError(err)
		return
	}

	ctx.RespEntity(s.generateWatchEventResp("", options.Resource, []*watch.WatchEventDetail{events}))
}

func (s *cacheService) generateWatchEventResp(startCursor string, rsc watch.CursorType,
	events []*watch.WatchEventDetail) *watch.WatchResp {

	result := new(watch.WatchResp)
	if len(events) == 0 {
		result.Watched = false
		if len(startCursor) == 0 {
			result.Events = []*watch.WatchEventDetail{
				{
					Cursor:   watch.NoEventCursor,
					Resource: rsc,
				},
			}
		} else {
			// if user's watch with a start cursor, but we do not find event after this cursor,
			// then we return this start cursor directly, so that they can watch with this cursor for next round.
			result.Events = []*watch.WatchEventDetail{
				{
					Cursor:   startCursor,
					Resource: rsc,
				},
			}
		}
	} else {
		if events[0].Cursor == watch.NoEventCursor {
			result.Watched = false

			if len(startCursor) == 0 {
				// user watch with start form time, or watch from now, then return with NoEventCursor cursor.
				result.Events = []*watch.WatchEventDetail{
					{
						Cursor:   watch.NoEventCursor,
						Resource: rsc,
					},
				}
			} else {
				// if user's watch with a start cursor, but hit a NoEventCursor cursor,
				// then we return this start cursor directly, so that they can watch with this cursor for next round.
				result.Events = []*watch.WatchEventDetail{
					{
						Cursor:   startCursor,
						Resource: rsc,
					},
				}
			}
		} else if events[0].Detail == nil {
			// compatible for event happens but not hit(with different event type), last cursor is returned with no detail
			result.Watched = false
			result.Events = []*watch.WatchEventDetail{events[0]}
		} else {
			result.Watched = true
			result.Events = events
		}
	}

	return result
}
