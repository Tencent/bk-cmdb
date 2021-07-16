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

package app

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"configcenter/src/common"
	"configcenter/src/common/backbone"
	cc "configcenter/src/common/backbone/configcenter"
	"configcenter/src/common/blog"
	"configcenter/src/common/types"
	"configcenter/src/scene_server/task_server/app/options"
	tasksvc "configcenter/src/scene_server/task_server/service"
	"configcenter/src/storage/dal/redis"

	"github.com/emicklei/go-restful"
)

func Run(ctx context.Context, cancel context.CancelFunc, op *options.ServerOption) error {
	svrInfo, err := types.NewServerInfo(op.ServConf)
	if err != nil {
		blog.Errorf("wrap server info failed, err: %v", err)
		return fmt.Errorf("wrap server info failed, err: %v", err)
	}

	service := new(tasksvc.Service)
	taskSrv := new(TaskServer)

	input := &backbone.BackboneParameter{
		Regdiscv:     op.ServConf.RegDiscover,
		ConfigPath:   op.ServConf.ExConfig,
		ConfigUpdate: taskSrv.onHostConfigUpdate,
		SrvInfo:      svrInfo,
	}

	engine, err := backbone.NewBackbone(ctx, input)
	if err != nil {
		blog.Errorf("new backbone failed, err: %v", err)
		return fmt.Errorf("new backbone failed, err: %v", err)
	}
	configReady := false
	for sleepCnt := 0; sleepCnt < common.APPConfigWaitTime; sleepCnt++ {
		if nil != taskSrv.Config {
			configReady = true
			break
		}
		blog.Infof("waiting for config ready ...")
		time.Sleep(time.Second)
	}
	if false == configReady {
		blog.Infof("waiting config timeout.")
		return errors.New("configuration item not found")
	}
	taskSrv.Config.Mongo, err = engine.WithMongo()
	if err != nil {
		return err
	}
	taskSrv.Config.Redis, err = engine.WithRedis()
	if err != nil {
		return err
	}
	cacheDB, err := redis.NewFromConfig(taskSrv.Config.Redis)
	if err != nil {
		blog.Errorf("new redis client failed, err: %s", err.Error())
		return fmt.Errorf("new redis client failed, err: %s", err.Error())
	}
	db, err := taskSrv.Config.Mongo.GetMongoClient()
	if err != nil {
		blog.Errorf("new mongo client failed, err: %s", err.Error())
		return fmt.Errorf("new mongo client failed, err: %s", err.Error())
	}

	service.Engine = engine
	service.Config = taskSrv.Config
	service.CacheDB = cacheDB
	service.DB = db
	taskSrv.Core = engine
	taskSrv.Service = service

	//Cron job delete history task
	go taskSrv.Service.TimerDeleteHistoryTask(ctx)
	if err := backbone.StartServer(ctx, cancel, engine, service.WebService(), true); err != nil {
		blog.Errorf("start backbone failed, err: %+v", err)
		return err
	}

	queue := service.NewQueue(taskSrv.taskQueue)
	queue.Start()
	select {
	case <-ctx.Done():
	}
	return nil
}

type TaskServer struct {
	Core      *backbone.Engine
	Config    *options.Config
	Service   *tasksvc.Service
	taskQueue map[string]tasksvc.TaskInfo
}

func (h *TaskServer) WebService() *restful.Container {
	return h.Service.WebService()
}

func (h *TaskServer) onHostConfigUpdate(previous, current cc.ProcessConfig) {
	if h.Config == nil {
		h.Config = &options.Config{}
	}
	name, _ := cc.String("taskServer.name")
	taskNameArr := strings.Split(name, ",")

	for _, name := range taskNameArr {
		if name == "" {
			continue
		}
		prefix := "taskServer." + name

		strRetry, _ := cc.String(prefix + ".retry")
		var retry int64 = 0
		var err error
		if strRetry != "" {
			retry, err = strconv.ParseInt(strRetry, 10, 64)
			if err != nil {
				retry = 0
				blog.Errorf(" parse task name %s retry %s to int error. err:%s", name, strRetry, err.Error())
			}
		}

		f := func() ([]string, error) {
			addrArray, _ := cc.String(prefix + ".addrs")
			addrs := strings.Split(addrArray, ",")
			return addrs, nil
		}
		path, _ := cc.String(prefix + ".path")
		task := tasksvc.TaskInfo{
			Name:  name,
			Addr:  f,
			Path:  path,
			Retry: retry,
		}
		if h.taskQueue == nil {
			h.taskQueue = make(map[string]tasksvc.TaskInfo, 0)
		}
		h.taskQueue[name] = task
	}

}
