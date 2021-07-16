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
	"sync"
	"time"
)

const topologyKey = bizNamespace + ":custom:topology"
const retryDuration = 500 * time.Millisecond

type cacheCollection struct {
	business *business
	set      *set
	module   *module
	custom   *customLevel
}

type refreshInstance struct {
	// the key to store data.
	mainKey        string
	lockKey        string
	expireKey      string
	expireDuration time.Duration
	// detail is to get the data to be refresh.
	getDetail func(instID int64) (string, error)
}

type bizArchive struct {
	Detail bizBaseInfo `json:"detail" bson:"detail"`
}

type bizBaseInfo struct {
	BusinessID   int64  `json:"bk_biz_id" bson:"bk_biz_id"`
	BusinessName string `json:"bk_biz_name" bson:"bk_biz_name"`
}

type moduleArchive struct {
	Detail moduleBaseInfo `json:"detail" bson:"detail"`
}

type moduleBaseInfo struct {
	ModuleID   int64  `json:"bk_module_id" bson:"bk_module_id"`
	ModuleName string `json:"bk_module_name" bson:"bk_module_name"`
}

type setArchive struct {
	Detail setBaseInfo `json:"detail" bson:"detail"`
}

type setBaseInfo struct {
	SetID    int64  `json:"bk_set_id" bson:"bk_set_id"`
	SetName  string `json:"bk_set_name" bson:"bk_set_name"`
	ParentID int64  `json:"bk_parent_id" bson:"bk_parent_id"`
}

type mainlineAssociation struct {
	AssociateTo string `json:"bk_asst_obj_id" bson:"bk_asst_obj_id"`
	ObjectID    string `json:"bk_obj_id" bson:"bk_obj_id"`
}

type customArchive struct {
	Detail customInstanceBase `json:"detail" bson:"detail"`
}
type customInstanceBase struct {
	ObjectID     string `json:"bk_obj_id" bson:"bk_obj_id"`
	InstanceID   int64  `json:"bk_inst_id" bson:"bk_inst_id"`
	InstanceName string `json:"bk_inst_name" bson:"bk_inst_name"`
	ParentID     int64  `json:"bk_parent_id" bson:"bk_parent_id"`
}

type watchObserver struct {
	// key is biz custom object id
	// value is this watch's stop channel notifier.
	observer map[string]chan struct{}
	lock     sync.Mutex
}

func (w *watchObserver) add(objID string, stopNotifier chan struct{}) {
	w.lock.Lock()
	w.observer[objID] = stopNotifier
	w.lock.Unlock()
	return
}

func (w *watchObserver) exist(objID string) bool {
	w.lock.Lock()
	_, exist := w.observer[objID]
	w.lock.Unlock()
	return exist
}

func (w *watchObserver) delete(objID string) chan struct{} {

	w.lock.Lock()
	stopNotifier := w.observer[objID]
	delete(w.observer, objID)
	w.lock.Unlock()
	return stopNotifier
}

func (w *watchObserver) getAllObjects() []string {
	all := make([]string, 0)
	w.lock.Lock()
	for obj := range w.observer {
		all = append(all, obj)
	}
	w.lock.Unlock()

	return all
}
