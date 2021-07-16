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
	ccError "configcenter/src/common/errors"
	"configcenter/src/common/json"
	"configcenter/src/common/mapstr"
	"configcenter/src/common/util"
)

func (c *Client) getBusinessFromMongo(bizID int64) (string, error) {
	biz := make(map[string]interface{})
	filter := mapstr.MapStr{
		common.BKAppIDField: bizID,
	}
	err := c.db.Table(common.BKTableNameBaseApp).Find(filter).One(context.Background(), &biz)
	if err != nil {
		blog.Errorf("get business %d info from db, but failed, err: %v", bizID, err)
		return "", ccError.New(common.CCErrCommDBSelectFailed, err.Error())
	}

	js, err := json.Marshal(biz)
	if err != nil {
		return "", err
	}

	return string(js), nil
}

func (c *Client) listBusinessWithRefreshCache(ctx context.Context, ids []int64, fields []string) ([]string, error) {
	rid := ctx.Value(common.ContextRequestIDField)

	list := make([]map[string]interface{}, 0)
	filter := mapstr.MapStr{
		common.BKAppIDField: mapstr.MapStr{
			common.BKDBIN: ids,
		},
	}

	err := c.db.Table(common.BKTableNameBaseApp).Find(filter).All(context.Background(), &list)
	if err != nil {
		blog.Errorf("list business info from db failed, err: %v, rid: %v", err, rid)
		return nil, ccError.New(common.CCErrCommDBSelectFailed, err.Error())
	}

	pipe := c.rds.Pipeline()
	all := make([]string, len(list))
	for idx, biz := range list {

		id, err := util.GetInt64ByInterface(biz[common.BKAppIDField])
		if err != nil {
			return nil, err
		}

		js, err := json.Marshal(biz)
		if err != nil {
			return nil, err
		}

		pipe.Set(bizKey.detailKey(id), js, detailTTLDuration)

		all[idx] = string(js)
		if len(fields) != 0 {
			all[idx] = *json.CutJsonDataWithFields(&all[idx], fields)
		}
	}

	_, err = pipe.Exec()
	if err != nil {
		blog.Errorf("update biz cache failed, err: %v, rid: %v", err, rid)
		// do not return
	}

	return all, nil
}

func (c *Client) listModuleWithRefreshCache(ctx context.Context, ids []int64, fields []string) ([]string, error) {
	rid := ctx.Value(common.ContextRequestIDField)

	list := make([]map[string]interface{}, 0)
	filter := mapstr.MapStr{
		common.BKModuleIDField: mapstr.MapStr{
			common.BKDBIN: ids,
		},
	}

	err := c.db.Table(common.BKTableNameBaseModule).Find(filter).All(context.Background(), &list)
	if err != nil {
		blog.Errorf("list module info from db failed, err: %v, rid: %v", err, rid)
		return nil, ccError.New(common.CCErrCommDBSelectFailed, err.Error())
	}

	pipe := c.rds.Pipeline()
	all := make([]string, len(list))
	for idx, mod := range list {
		id, err := util.GetInt64ByInterface(mod[common.BKModuleIDField])
		if err != nil {
			return nil, err
		}

		js, err := json.Marshal(mod)
		if err != nil {
			return nil, err
		}

		pipe.Set(moduleKey.detailKey(id), js, detailTTLDuration)

		all[idx] = string(js)
		if len(fields) != 0 {
			all[idx] = *json.CutJsonDataWithFields(&all[idx], fields)
		}
	}

	_, err = pipe.Exec()
	if err != nil {
		blog.Errorf("update module cache failed, err: %v, rid: %v", err, rid)
		// do not return
	}

	return all, nil
}

func (c *Client) listSetWithRefreshCache(ctx context.Context, ids []int64, fields []string) ([]string, error) {
	rid := ctx.Value(common.ContextRequestIDField)

	list := make([]map[string]interface{}, 0)
	filter := mapstr.MapStr{
		common.BKSetIDField: mapstr.MapStr{
			common.BKDBIN: ids,
		},
	}

	err := c.db.Table(common.BKTableNameBaseSet).Find(filter).All(context.Background(), &list)
	if err != nil {
		blog.Errorf("list set info from db failed, err: %v, rid: %v", err, rid)
		return nil, ccError.New(common.CCErrCommDBSelectFailed, err.Error())
	}

	pipe := c.rds.Pipeline()
	all := make([]string, len(list))
	for idx, set := range list {
		id, err := util.GetInt64ByInterface(set[common.BKSetIDField])
		if err != nil {
			return nil, err
		}

		js, err := json.Marshal(set)
		if err != nil {
			return nil, err
		}

		pipe.Set(setKey.detailKey(id), js, detailTTLDuration)

		all[idx] = string(js)
		if len(fields) != 0 {
			all[idx] = *json.CutJsonDataWithFields(&all[idx], fields)
		}
	}

	_, err = pipe.Exec()
	if err != nil {
		blog.Errorf("update set cache failed, err: %v, rid: %v", err, rid)
		// do not return
	}

	return all, nil
}

func (c *Client) getModuleDetailCheckNotFoundWithRefreshCache(ctx context.Context, id int64) (string, bool, error) {
	rid := ctx.Value(common.ContextRequestIDField)

	mod := make(map[string]interface{})
	filter := mapstr.MapStr{
		common.BKModuleIDField: id,
	}

	if err := c.db.Table(common.BKTableNameBaseModule).Find(filter).One(context.Background(), &mod); err != nil {
		blog.Errorf("get module %d detail from mongo failed, err: %v, rid: %v", id, err, rid)

		// if module is not found, returns not found flag
		if c.db.IsNotFoundError(err) {
			return "", true, err
		}
		return "", false, err
	}

	js, err := json.Marshal(mod)
	if err != nil {
		return "", false, err
	}

	// refresh cache
	err = c.rds.Set(ctx, moduleKey.detailKey(id), js, detailTTLDuration).Err()
	if err != nil {
		blog.Errorf("update module: %d cache failed, err: %v, rid: %v", id, err, rid)
		// do not return
	}

	return string(js), false, nil
}

func (c *Client) getSetDetailCheckNotFoundWithRefreshCache(ctx context.Context, id int64) (string, bool, error) {
	rid := ctx.Value(common.ContextRequestIDField)

	set := make(map[string]interface{})
	filter := mapstr.MapStr{
		common.BKSetIDField: id,
	}

	if err := c.db.Table(common.BKTableNameBaseSet).Find(filter).One(context.Background(), &set); err != nil {
		blog.Errorf("get set %d detail from mongo failed, err: %v, rid: %v", id, err, rid)

		// if set is not found, returns not found flag
		if c.db.IsNotFoundError(err) {
			return "", true, err
		}
		return "", false, err
	}

	js, err := json.Marshal(set)
	if err != nil {
		return "", false, err
	}

	// refresh cache
	err = c.rds.Set(ctx, setKey.detailKey(id), js, detailTTLDuration).Err()
	if err != nil {
		blog.Errorf("update set: %d cache failed, err: %v, rid: %v", id, err, rid)
		// do not return
	}

	return string(js), false, nil
}

func (c *Client) getCustomDetailCheckNotFoundWithRefreshCache(ctx context.Context, key *keyGenerator, objID,
	supplierAccount string, instID int64) (string, bool, error) {

	rid := ctx.Value(common.ContextRequestIDField)

	filter := mapstr.MapStr{
		common.BKObjIDField:  objID,
		common.BKInstIDField: instID,
	}
	instance := make(map[string]interface{})
	instTableName := common.GetObjectInstTableName(objID, supplierAccount)

	err := c.db.Table(instTableName).Find(filter).One(context.Background(), &instance)
	// if module is not found, returns not found flag
	if c.db.IsNotFoundError(err) {
		return "", true, err
	}

	if err != nil {
		blog.Errorf("get custom level object: %s, inst: %d from db failed, err: %v, rid: %v", objID, instID, err, rid)
		return "", false, err
	}

	js, err := json.Marshal(instance)
	if err != nil {
		return "", false, err
	}

	// refresh cache
	err = c.rds.Set(ctx, key.detailKey(instID), js, detailTTLDuration).Err()
	if err != nil {
		blog.Errorf("update object: %s, inst: %d cache failed, err: %v, rid: %v", objID, instID, err, rid)
		// do not return
	}

	return string(js), false, nil
}

func (c *Client) listCustomLevelDetailWithRefreshCache(ctx context.Context, key *keyGenerator, objID,
	supplierAccount string, instIDs []int64) ([]string, error) {

	rid := ctx.Value(common.ContextRequestIDField)

	filter := mapstr.MapStr{
		common.BKObjIDField: objID,
		common.BKInstIDField: mapstr.MapStr{
			common.BKDBIN: instIDs,
		},
	}

	instanceTableName := common.GetObjectInstTableName(objID, supplierAccount)
	instance := make([]map[string]interface{}, 0)
	err := c.db.Table(instanceTableName).Find(filter).One(context.Background(), &instance)
	if err != nil {
		blog.Errorf("get custom level object: %s, inst: %v from db failed, err: %v, rid: %v", objID, instIDs, err, rid)
		return nil, err
	}

	pipe := c.rds.Pipeline()
	all := make([]string, len(instance))
	for idx := range instance {
		js, err := json.Marshal(instance[idx])
		if err != nil {
			return nil, err
		}
		all[idx] = string(js)

		id, err := util.GetInt64ByInterface(instance[idx][common.BKInstIDField])
		if err != nil {
			return nil, err
		}

		pipe.Set(key.detailKey(id), js, detailTTLDuration)
	}

	_, err = pipe.Exec()
	if err != nil {
		blog.Errorf("update custom object instance cache failed, err: %v, rid: %v", err, rid)
		// do not return
	}

	return all, nil
}

func (c *Client) refreshAndGetTopologyRank() ([]string, error) {
	// read information from mongodb
	relations, err := getMainlineTopology()
	if err != nil {
		blog.Errorf("refresh mainline topology rank, but get it from mongodb failed, err: %v", err)
		return nil, err
	}
	// rank start from biz to host
	rank := rankMainlineTopology(relations)
	refreshTopologyRank(rank)

	return rank, nil
}
