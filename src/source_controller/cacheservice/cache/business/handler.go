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
	"encoding/json"
	"time"

	"configcenter/src/storage/dal/redis"
	drv "configcenter/src/storage/driver/redis"
	"configcenter/src/storage/stream/types"
)

func newTokenHandler(key keyGenerator) *tokenHandler {
	return &tokenHandler{
		key: key,
		rds: drv.Client(),
	}
}

type tokenHandler struct {
	key keyGenerator
	rds redis.Client
}

// SetLastWatchToken set watch token and resume time at the same time.
func (t *tokenHandler) SetLastWatchToken(_ context.Context, token string) error {
	stamp := &types.TimeStamp{
		Sec:  uint32(time.Now().Unix()),
		Nano: 0,
	}
	atTime, err := json.Marshal(stamp)
	if err != nil {
		return err
	}

	pipe := t.rds.Pipeline()
	pipe.Set(t.key.resumeTokenKey(), token, 0)
	pipe.Set(t.key.resumeAtTimeKey(), string(atTime), 0)
	_, err = pipe.Exec()
	if err != nil {
		return err
	}

	return nil
}

// GetStartWatchToken get the last watched token, it can be empty.
func (t *tokenHandler) GetStartWatchToken(ctx context.Context) (token string, err error) {
	token, err = t.rds.Get(ctx, t.key.resumeTokenKey()).Result()
	if err != nil {
		if redis.IsNilErr(err) {
			return "", nil
		}
		return "", err
	}
	return token, err
}

// getStartTimestamp get the last event's timestamp.
func (t *tokenHandler) getStartTimestamp(ctx context.Context) (*types.TimeStamp, error) {
	js, err := t.rds.Get(ctx, t.key.resumeAtTimeKey()).Result()
	if err != nil {
		if redis.IsNilErr(err) {
			// start from now.
			return &types.TimeStamp{Sec: uint32(time.Now().Unix())}, nil
		}
		return nil, err
	}

	stamp := new(types.TimeStamp)
	if len(js) == 0 {
		// it will be empty when it is never set.
		return stamp, nil
	}

	if err := json.Unmarshal([]byte(js), stamp); err != nil {
		return nil, err
	}

	return stamp, nil
}

// resetWatchToken reset the watch token, and update startAtTime time, so that we can
// re-watch from the timestamp we set now.
func (t *tokenHandler) resetWatchTokenWithTimestamp(startAtTime types.TimeStamp) error {
	atTime, err := json.Marshal(startAtTime)
	if err != nil {
		return err
	}

	pipe := t.rds.Pipeline()
	pipe.Set(t.key.resumeTokenKey(), "", 0)
	pipe.Set(t.key.resumeAtTimeKey(), string(atTime), 0)
	_, err = pipe.Exec()
	if err != nil {
		return err
	}

	return nil
}
