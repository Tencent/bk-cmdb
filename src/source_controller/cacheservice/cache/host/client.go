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

package host

import (
	"context"
	"errors"
	"strings"

	"configcenter/src/common"
	"configcenter/src/common/blog"
	"configcenter/src/common/json"
	"configcenter/src/common/mapstr"
	"configcenter/src/common/metadata"
	"configcenter/src/common/util"
	"configcenter/src/source_controller/cacheservice/cache/tools"
	"configcenter/src/storage/driver/redis"
)

type Client struct {
	lock tools.RefreshingLock
}

// GetHostWithID get host with host id.
// fields allows you can specify which fields you need only.
func (c *Client) GetHostWithID(ctx context.Context, opt *metadata.SearchHostWithIDOption) (string, error) {
	rid := ctx.Value(common.ContextRequestIDField)
	needRefresh := false
	data, err := redis.Client().Get(context.Background(), hostKey.HostDetailKey(opt.HostID)).Result()
	if err != nil {
		if !redis.IsNilErr(err) {
			// return directly to avoid cache penetration
			blog.Errorf("get host: %d from redis failed, err: %v, rid: %s", opt.HostID, err, rid)
			return "", err
		}
		// do not exist in cache, need to refresh from db.
		needRefresh = true
	}

	if !needRefresh {
		// already get the data, use cache
		if len(opt.Fields) == 0 {
			return data, nil
		}
		return *json.CutJsonDataWithFields(&data, opt.Fields), nil
	}

	// data has already expired, need to refresh from db
	ips, cloudID, detail, err := getHostDetailsFromMongoWithHostID(opt.HostID)
	if err != nil {
		blog.Errorf("get host with id: %d, and cache expired, but get from mongo failed, err: %v, rid: %s", opt.HostID, err, rid)
		return "", err
	}

	// try refresh cache
	c.tryRefreshHostDetail(opt.HostID, ips, cloudID, detail)

	if len(opt.Fields) == 0 {
		return string(detail), nil
	} else {
		h := string(detail)
		return *json.CutJsonDataWithFields(&h, opt.Fields), nil
	}
}

// ListHostWithHostIDs list hosts info from redis with host id list.
// if a host is not exist in cache and still can not find in mongodb,
// then it will not be return. so the returned array may not equal to
// the request host ids length and the sequence is also may not same.
func (c *Client) ListHostWithHostIDs(ctx context.Context, opt *metadata.ListWithIDOption) ([]string, error) {
	rid := ctx.Value(common.ContextRequestIDField)
	if len(opt.IDs) > 500 {
		return nil, errors.New("host id length is over limit")
	}

	if len(opt.IDs) == 0 {
		return nil, errors.New("host id array is empty")
	}

	keys := make([]string, len(opt.IDs))
	for i, id := range opt.IDs {
		keys[i] = hostKey.HostDetailKey(id)
	}

	hosts, err := redis.Client().MGet(context.Background(), keys...).Result()
	if err != nil {
		blog.Errorf("list host with ids, but get from redis failed, err: %v, rid: %s", err, rid)
		return nil, err
	}

	needRefreshIdx := make([]int, 0)
	list := make([]string, 0)
	for idx, h := range hosts {
		if h == nil {
			needRefreshIdx = append(needRefreshIdx, idx)
			continue
		}
		detail, ok := h.(string)
		if !ok {
			blog.Errorf("list host with ids, but got invalid host type, not string, host: %v, rid: %s", h, rid)
			return nil, errors.New("invalid host detail type, not string")
		}

		if len(opt.Fields) != 0 {
			list = append(list, *json.CutJsonDataWithFields(&detail, opt.Fields))
		} else {
			list = append(list, detail)
		}
	}

	if len(needRefreshIdx) != 0 {
		// can not found in the cache, need refresh the cache
		ids := make([]int64, len(needRefreshIdx))
		for i, idx := range needRefreshIdx {
			ids[i] = opt.IDs[idx]
		}

		toAdd, err := listHostDetailsFromMongoWithHostID(ids)
		if err != nil {
			blog.Errorf("list host with ids, but get from db failed, host: %v, rid: %s", ids, rid)
			return nil, err
		}
		for _, host := range toAdd {
			c.tryRefreshHostDetail(host.id, host.ip, host.cloudID, []byte(host.detail))

			if len(opt.Fields) != 0 {
				list = append(list, *json.CutJsonDataWithFields(&host.detail, opt.Fields))
			} else {
				list = append(list, host.detail)
			}
		}
	}
	return list, nil
}

// GetHostWithInnerIP is to get host with the ip and cloud id it belongs.
// the ip must be a unique one, can not be a ip string with multiple ip separated with comma.
func (c *Client) GetHostWithInnerIP(ctx context.Context, opt *metadata.SearchHostWithInnerIPOption) (string, error) {
	rid := ctx.Value(common.ContextRequestIDField)
	if len(opt.InnerIP) == 0 || len(strings.Split(opt.InnerIP, ",")) > 1 {
		return "", errors.New("invalid ip address with multiple ip")
	}

	detail, err := c.getHostDetailWithIP(opt.InnerIP, opt.CloudID)
	if err != nil {
		blog.Errorf("get host with inner ip: %s failed, err：%v, rid: %s", opt.InnerIP, err, rid)
		return "", err
	}

	if len(opt.Fields) == 0 {
		return *detail, nil
	} else {
		return *json.CutJsonDataWithFields(detail, opt.Fields), nil
	}
}

// ListHostIDsWithPage get host id list sorted with host id with forward sort.
// this id list has a ttl life cycle, and triggered with update with user's request.
func (c *Client) ListHostsWithPage(ctx context.Context, opt *metadata.ListHostWithPage) (int64, []string, error) {
	rid := util.ExtractRequestIDFromContext(ctx)

	if len(opt.HostIDs) != 0 {
		// find with host id directly.
		total, err := c.countHost(ctx, mapstr.MapStr{common.BKHostIDField: mapstr.MapStr{common.BKDBIN: opt.HostIDs}})
		if err != nil {
			blog.Errorf("list host with page, but count failed, err: %v, rid: %v", err, rid)
			return 0, nil, err
		}

		options := metadata.ListWithIDOption{
			IDs:    opt.HostIDs,
			Fields: opt.Fields,
		}
		list, err := c.ListHostWithHostIDs(ctx, &options)

		return int64(total), list, err
	}

	// validate the page limit
	if opt.Page.Limit > common.BKMaxPageSize {
		return 0, nil, errors.New("page size is over limit")
	}

	cnt, idList, details, err := c.getPagedHostDetailList(opt.Page)
	if err != nil {
		if err != keyNotExistError {
			return 0, nil, err
		}

		// force refresh the host id list
		c.forceRefreshHostIDList(ctx)

		// zset key is not exist, then we get it from mongodb.
		return c.getHostsWithPage(ctx, opt)
	}

	// try to refresh host id list in cache.
	c.tryRefreshHostIDList(rid)

	// check details and find those which needs to be refreshed
	all := make([]string, 0)
	toRefreshIds := make([]int64, 0)
	for idx, h := range details {
		if len(h) == 0 {
			toRefreshIds = append(toRefreshIds, idList[idx])
			continue
		}

		if len(opt.Fields) != 0 {
			// only return with user needed fields.
			all = append(all, *json.CutJsonDataWithFields(&h, opt.Fields))
		} else {
			all = append(all, h)
		}
	}

	if len(toRefreshIds) != 0 {

		refresh, err := listHostDetailsFromMongoWithHostID(toRefreshIds)
		if err != nil {
			blog.Errorf("list host with ids, but get from db failed, host: %v, rid: %s", toRefreshIds, rid)
			return 0, nil, err
		}

		for _, host := range refresh {
			c.tryRefreshHostDetail(host.id, host.ip, host.cloudID, []byte(host.detail))

			if len(opt.Fields) != 0 {
				// only return with user needed fields.
				all = append(all, *json.CutJsonDataWithFields(&host.detail, opt.Fields))
			} else {
				all = append(all, host.detail)
			}
		}
	}

	return cnt, all, nil
}
