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

	"configcenter/src/common"
	"configcenter/src/common/blog"
	"configcenter/src/common/json"
	"configcenter/src/common/metadata"
	"configcenter/src/storage/dal"
	"configcenter/src/storage/dal/redis"
)

type Client struct {
	rds redis.Client
	db  dal.DB
}

// get a business's all info.
func (c *Client) GetBusiness(ctx context.Context, bizID int64) (string, error) {
	rid := ctx.Value(common.BKHTTPCCRequestID)

	key := bizKey.detailKey(bizID)
	biz, err := c.rds.Get(ctx, key).Result()
	if err == nil {
		return biz, nil
	}
	// error occurs, get from db directly.
	blog.Errorf("get business %d info from cache failed, will get from db, err: %v, rid: %v", bizID, err, rid)

	biz, err = c.getBusinessFromMongo(bizID)
	if err != nil {
		blog.Errorf("get biz detail from db failed, err: %v, rid: %v", err, rid)
		return "", err
	}

	err = c.rds.Set(ctx, bizKey.detailKey(bizID), biz, bizKey.detailExpireDuration).Err()
	if err != nil {
		blog.Errorf("update biz cache failed, err: %v, rid: %s", err, rid)
		// do not return
	}

	// get from db
	return biz, nil
}

func (c *Client) ListBusiness(ctx context.Context, opt *metadata.ListWithIDOption) ([]string, error) {
	rid := ctx.Value(common.ContextRequestIDField)

	if len(opt.IDs) == 0 {
		return make([]string, 0), nil
	}

	keys := make([]string, len(opt.IDs))
	for idx, bizID := range opt.IDs {
		keys[idx] = bizKey.detailKey(bizID)
	}

	bizList, err := c.rds.MGet(context.Background(), keys...).Result()
	if err != nil {
		blog.Errorf("get business %d info from cache failed, get from db directly, err: %v, rid: %v", opt.IDs, err, rid)
		return c.listBusinessWithRefreshCache(ctx, opt.IDs, opt.Fields)
	}

	all := make([]string, 0)
	toAdd := make([]int64, 0)
	for idx, biz := range bizList {
		if biz == nil {
			// can not find in cache
			toAdd = append(toAdd, opt.IDs[idx])
			continue
		}

		detail, ok := biz.(string)
		if !ok {
			blog.Errorf("got invalid biz cache %v, rid: %v", biz, rid)
			return nil, fmt.Errorf("got invalid biz cache %v", biz)
		}

		if len(opt.Fields) != 0 {
			all = append(all, *json.CutJsonDataWithFields(&detail, opt.Fields))
		} else {
			all = append(all, detail)
		}

	}

	if len(toAdd) != 0 {
		details, err := c.listBusinessWithRefreshCache(ctx, toAdd, opt.Fields)
		if err != nil {
			blog.Errorf("get business list from db failed, err: %v, rid: %v", err, rid)
			return nil, err
		}

		all = append(all, details...)
	}

	return all, nil
}

func (c *Client) ListModules(ctx context.Context, opt *metadata.ListWithIDOption) ([]string, error) {
	rid := ctx.Value(common.ContextRequestIDField)

	if len(opt.IDs) == 0 {
		return make([]string, 0), nil
	}

	keys := make([]string, len(opt.IDs))
	list, err := c.rds.MGet(context.Background(), keys...).Result()
	if err != nil {
		blog.Errorf("list module %d info from cache failed, get from db directly, err: %v, rid: %v", opt.IDs, err, rid)
		return c.listModuleWithRefreshCache(ctx, opt.IDs, opt.Fields)
	}

	all := make([]string, 0)
	toAdd := make([]int64, 0)
	for idx, module := range list {
		if module == nil {
			// can not find in cache
			toAdd = append(toAdd, opt.IDs[idx])
			continue
		}

		detail, ok := module.(string)
		if !ok {
			blog.Errorf("got invalid module cache %v, rid: %v", module, rid)
			return nil, fmt.Errorf("got invalid module cache %v", module)
		}

		if len(opt.Fields) != 0 {
			all = append(all, *json.CutJsonDataWithFields(&detail, opt.Fields))
		} else {
			all = append(all, detail)
		}
	}

	if len(toAdd) != 0 {
		details, err := c.listModuleWithRefreshCache(ctx, toAdd, opt.Fields)
		if err != nil {
			blog.Errorf("get module list from db failed, err: %v, rid: %v", err, rid)
			return nil, err
		}

		all = append(all, details...)
	}

	return all, nil
}

func (c *Client) ListSets(ctx context.Context, opt *metadata.ListWithIDOption) ([]string, error) {
	rid := ctx.Value(common.ContextRequestIDField)

	if len(opt.IDs) == 0 {
		return make([]string, 0), nil
	}

	keys := make([]string, len(opt.IDs))
	for idx, id := range opt.IDs {
		keys[idx] = setKey.detailKey(id)
	}

	list, err := c.rds.MGet(ctx, keys...).Result()
	if err != nil {
		blog.Errorf("list set %d info from cache failed, get from db directly, err: %v, rid: %v", opt.IDs, err, rid)
		return c.listSetWithRefreshCache(ctx, opt.IDs, opt.Fields)
	}

	all := make([]string, 0)
	toAdd := make([]int64, 0)
	for idx, set := range list {
		if set == nil {
			// can not find in cache
			toAdd = append(toAdd, opt.IDs[idx])
			continue
		}

		detail, ok := set.(string)
		if !ok {
			blog.Errorf("got invalid set cache %v, rid: %v", set, rid)
			return nil, fmt.Errorf("got invalid set cache %v", set)
		}

		if len(opt.Fields) != 0 {
			all = append(all, *json.CutJsonDataWithFields(&detail, opt.Fields))
		} else {
			all = append(all, detail)
		}
	}

	if len(toAdd) != 0 {
		details, err := c.listSetWithRefreshCache(ctx, toAdd, opt.Fields)
		if err != nil {
			blog.Errorf("get set list from db failed, err: %v, rid: %v", err, rid)
			return nil, err
		}

		all = append(all, details...)
	}

	return all, nil
}

func (c *Client) ListModuleDetails(ctx context.Context, moduleIDs []int64) ([]string, error) {
	if len(moduleIDs) == 0 {
		return make([]string, 0), nil
	}

	rid := ctx.Value(common.ContextRequestIDField)

	keys := make([]string, len(moduleIDs))
	for idx, id := range moduleIDs {
		keys[idx] = moduleKey.detailKey(id)
	}

	modules, err := c.rds.MGet(context.Background(), keys...).Result()
	if err == nil {
		list := make([]string, 0)
		for idx, m := range modules {
			if m == nil {
				detail, isNotFound, err := c.getModuleDetailCheckNotFoundWithRefreshCache(ctx, moduleIDs[idx])
				// 跳过不存在的模块，因为作为批量查询的API，调用方希望查询到存在的资源，并自动过滤掉不存在的资源
				if isNotFound {
					blog.Errorf("module %d not exist, err: %v, rid: %v", moduleIDs[idx], err, rid)
					continue
				}

				if err != nil {
					blog.Errorf("get module %d detail from db failed, err: %v, rid: %v", moduleIDs[idx], err, rid)
					return nil, err
				}

				list = append(list, detail)
				continue
			}
			list = append(list, m.(string))
		}
		return list, nil
	}
	blog.Errorf("get modules details from redis failed, err: %v, rid: %v", err, rid)

	// can not get from redis, get from db directly.
	return c.listModuleWithRefreshCache(ctx, moduleIDs, nil)
}

func (c *Client) GetModuleDetail(ctx context.Context, moduleID int64) (string, error) {
	rid := ctx.Value(common.ContextRequestIDField)

	mod, err := c.rds.Get(context.Background(), moduleKey.detailKey(moduleID)).Result()
	if err == nil {
		return mod, nil
	}

	blog.Errorf("get module: %d failed from redis, err: %v, rid: %v", moduleID, err, rid)
	// get from db directly.\
	detail, _, err := c.getModuleDetailCheckNotFoundWithRefreshCache(ctx, moduleID)
	return detail, err
}

func (c *Client) GetSet(ctx context.Context, setID int64) (string, error) {
	rid := ctx.Value(common.ContextRequestIDField)

	set, err := c.rds.Get(context.Background(), setKey.detailKey(setID)).Result()
	if err == nil {
		return set, nil
	}

	blog.Errorf("get set: %d failed from redis failed, err: %v, rid: %v", setID, err, rid)

	detail, _, err := c.getSetDetailCheckNotFoundWithRefreshCache(ctx, setID)
	return detail, err
}

func (c *Client) ListSetDetails(ctx context.Context, setIDs []int64) ([]string, error) {
	rid := ctx.Value(common.ContextRequestIDField)

	if len(setIDs) == 0 {
		return make([]string, 0), nil
	}

	keys := make([]string, len(setIDs))
	for idx, set := range setIDs {
		keys[idx] = setKey.detailKey(set)
	}

	sets, err := c.rds.MGet(context.Background(), keys...).Result()
	if err == nil && len(sets) != 0 {
		all := make([]string, 0)
		for idx, s := range sets {
			if s == nil {
				detail, isNotFound, err := c.getSetDetailCheckNotFoundWithRefreshCache(ctx, setIDs[idx])
				// 跳过不存在的集群，因为作为批量查询的API，调用方希望查询到存在的资源，并自动过滤掉不存在的资源
				if isNotFound {
					blog.Errorf("set %d not exist, err: %v, rid: %v", setIDs[idx], err, rid)
					continue
				}

				if err != nil {
					blog.Errorf("get set %d from mongodb failed, err: %v, rid: %v", setIDs[idx], err, rid)
					return nil, err
				}
				all = append(all, detail)
				continue
			}
			all = append(all, s.(string))
		}

		return all, nil
	}

	blog.Errorf("get sets: %v failed from redis failed, err: %v, rid: %v", setIDs, err, rid)

	// get from db directly.
	return c.listSetWithRefreshCache(ctx, setIDs, nil)
}

func (c *Client) GetCustomLevelDetail(ctx context.Context, objID, supplierAccount string, instID int64) (
	string, error) {

	rid := ctx.Value(common.ContextRequestIDField)
	key := newCustomKey(objID)
	custom, err := c.rds.Get(context.Background(), key.detailKey(instID)).Result()
	if err == nil {
		return custom, nil
	}

	blog.Errorf("get biz custom level, obj:%s, inst: %d failed from redis, err: %v, rid: %v", objID, instID, err, rid)

	detail, _, err := c.getCustomDetailCheckNotFoundWithRefreshCache(ctx, key, objID, supplierAccount, instID)
	return detail, err
}

func (c *Client) ListCustomLevelDetail(ctx context.Context, objID, supplierAccount string, instIDs []int64) (
	[]string, error) {

	if len(instIDs) == 0 {
		return make([]string, 0), nil
	}

	rid := ctx.Value(common.ContextRequestIDField)

	customKey := newCustomKey(objID)
	keys := make([]string, len(instIDs))
	for idx, instID := range instIDs {
		keys[idx] = customKey.detailKey(instID)
	}

	customs, err := c.rds.MGet(context.Background(), keys...).Result()
	if err == nil && len(customs) != 0 {
		all := make([]string, 0)
		for idx, cu := range customs {
			if cu == nil {
				detail, isNotFound, err := c.getCustomDetailCheckNotFoundWithRefreshCache(ctx, customKey, objID,
					supplierAccount, instIDs[idx])
				// 跳过不存在的自定义节点，因为作为批量查询的API，调用方希望查询到存在的资源，并自动过滤掉不存在的资源
				if isNotFound {
					blog.Errorf("custom layer %s/%d not exist, err: %v, rid: %v", objID, instIDs[idx], err, rid)
					continue
				}

				if err != nil {
					blog.Errorf("get %s/%d detail from mongodb failed, err: %v, rid: %v", objID, instIDs[idx], err, rid)
					return nil, err
				}
				all = append(all, detail)
				continue
			}

			all = append(all, cu.(string))
		}
		return all, nil
	}

	blog.Errorf("get biz custom level, obj:%s, inst: %v failed from redis, err: %v, rid: %v", objID, instIDs, err, rid)
	// get from db directly.
	return c.listCustomLevelDetailWithRefreshCache(ctx, customKey, objID, supplierAccount, instIDs)
}

func (c *Client) GetTopology() ([]string, error) {

	rank, err := c.rds.Get(context.Background(), topologyKey).Result()
	if err != nil {
		blog.Errorf("get mainline topology from cache failed, get from db directly. err: %v", err)
		return c.refreshAndGetTopologyRank()
	}

	topo := strings.Split(rank, ",")
	if len(topo) < 4 {
		// invalid topology
		return c.refreshAndGetTopologyRank()
	}

	return topo, nil
}
