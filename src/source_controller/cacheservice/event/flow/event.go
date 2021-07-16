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

	"configcenter/src/apimachinery/discovery"
	"configcenter/src/common/blog"
	"configcenter/src/common/metadata"
	"configcenter/src/source_controller/cacheservice/event"
	"configcenter/src/storage/dal"
	"configcenter/src/storage/dal/mongo/local"
	"configcenter/src/storage/stream"
)

func NewEvent(watch stream.LoopInterface, isMaster discovery.ServiceManageInterface, watchDB dal.DB, ccDB dal.DB) error {
	watchMongoDB, ok := watchDB.(*local.Mongo)
	if !ok {
		blog.Errorf("watch event, but watch db is not an instance of local mongo to start transaction")
		return fmt.Errorf("watch db is not an instance of local mongo")
	}

	e := Event{
		watch:    watch,
		isMaster: isMaster,
		watchDB:  watchMongoDB,
		ccDB:     ccDB,
	}

	if err := e.runHost(context.Background()); err != nil {
		blog.Errorf("run host event flow failed, err: %v", err)
		return err
	}

	if err := e.runModuleHostRelation(context.Background()); err != nil {
		blog.Errorf("run module host config event flow failed, err: %v", err)
		return err
	}

	if err := e.runBiz(context.Background()); err != nil {
		blog.Errorf("run biz event flow failed, err: %v", err)
		return err
	}

	if err := e.runSet(context.Background()); err != nil {
		blog.Errorf("run set event flow failed, err: %v", err)
		return err
	}

	if err := e.runModule(context.Background()); err != nil {
		blog.Errorf("run module event flow failed, err: %v", err)
		return err
	}

	if err := e.runObjectBase(context.Background()); err != nil {
		blog.Errorf("run object base event flow failed, err: %v", err)
		return err
	}

	if err := e.runProcess(context.Background()); err != nil {
		blog.Errorf("run process event flow failed, err: %v", err)
		return err
	}

	if err := e.runProcessInstanceRelation(context.Background()); err != nil {
		blog.Errorf("run process instance relation event flow failed, err: %v", err)
		return err
	}

	gc := &gc{
		ccDB:     ccDB,
		isMaster: isMaster,
	}
	gc.cleanDelArchiveData(context.Background())

	return nil
}

type Event struct {
	watch    stream.LoopInterface
	watchDB  *local.Mongo
	ccDB     dal.DB
	isMaster discovery.ServiceManageInterface
}

func (e *Event) runHost(ctx context.Context) error {
	opts := flowOptions{
		key:         event.HostKey,
		watch:       e.watch,
		watchDB:     e.watchDB,
		ccDB:        e.ccDB,
		isMaster:    e.isMaster,
		EventStruct: new(metadata.HostMapStr),
	}

	return newFlow(ctx, opts, getHostDeleteEventDetails)
}

func (e *Event) runModuleHostRelation(ctx context.Context) error {
	opts := flowOptions{
		key:         event.ModuleHostRelationKey,
		watch:       e.watch,
		watchDB:     e.watchDB,
		ccDB:        e.ccDB,
		isMaster:    e.isMaster,
		EventStruct: new(map[string]interface{}),
	}

	return newFlow(ctx, opts, getDeleteEventDetails)
}

func (e *Event) runBiz(ctx context.Context) error {
	opts := flowOptions{
		key:         event.BizKey,
		watch:       e.watch,
		watchDB:     e.watchDB,
		ccDB:        e.ccDB,
		isMaster:    e.isMaster,
		EventStruct: new(map[string]interface{}),
	}

	return newFlow(ctx, opts, getDeleteEventDetails)
}

func (e *Event) runSet(ctx context.Context) error {
	opts := flowOptions{
		key:         event.SetKey,
		watch:       e.watch,
		watchDB:     e.watchDB,
		ccDB:        e.ccDB,
		isMaster:    e.isMaster,
		EventStruct: new(map[string]interface{}),
	}

	return newFlow(ctx, opts, getDeleteEventDetails)
}

func (e *Event) runModule(ctx context.Context) error {
	opts := flowOptions{
		key:         event.ModuleKey,
		watch:       e.watch,
		watchDB:     e.watchDB,
		ccDB:        e.ccDB,
		isMaster:    e.isMaster,
		EventStruct: new(map[string]interface{}),
	}

	return newFlow(ctx, opts, getDeleteEventDetails)
}

func (e *Event) runObjectBase(ctx context.Context) error {
	opts := flowOptions{
		key:         event.ObjectBaseKey,
		watch:       e.watch,
		watchDB:     e.watchDB,
		ccDB:        e.ccDB,
		isMaster:    e.isMaster,
		EventStruct: new(map[string]interface{}),
	}

	return newInstanceFlow(ctx, opts, getDeleteEventDetails)
}

func (e *Event) runProcess(ctx context.Context) error {
	opts := flowOptions{
		key:         event.ProcessKey,
		watch:       e.watch,
		watchDB:     e.watchDB,
		ccDB:        e.ccDB,
		isMaster:    e.isMaster,
		EventStruct: new(map[string]interface{}),
	}

	return newFlow(ctx, opts, getDeleteEventDetails)
}

func (e *Event) runProcessInstanceRelation(ctx context.Context) error {
	opts := flowOptions{
		key:         event.ProcessInstanceRelationKey,
		watch:       e.watch,
		watchDB:     e.watchDB,
		ccDB:        e.ccDB,
		isMaster:    e.isMaster,
		EventStruct: new(map[string]interface{}),
	}

	return newFlow(ctx, opts, getDeleteEventDetails)
}
