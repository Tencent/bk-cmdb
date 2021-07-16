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

package mongodb

import (
	"strings"
	"time"

	"configcenter/src/common"
	cc "configcenter/src/common/backbone/configcenter"
	"configcenter/src/common/blog"
	"configcenter/src/common/errors"
	"configcenter/src/common/metric"
	"configcenter/src/common/types"
	"configcenter/src/storage/dal"
	"configcenter/src/storage/dal/mongo"
	"configcenter/src/storage/dal/mongo/local"
	dbType "configcenter/src/storage/dal/types"
)

/*
 暂时不支持，多个mongodb实例连接， 暂时不值热更新，所以没有加锁
*/

var (
	db dal.RDB
	// 在并发的情况下，这里存在panic的问题
	lastInitErr   errors.CCErrorCoder
	lastConfigErr errors.CCErrorCoder
)

// Client  get default error
func Client() dal.RDB {
	return db
}

// Table 获取操作db table的对象
func Table(name string) dbType.Table {
	return db.Table(name)
}

func ParseConfig(prefix string, configMap map[string]string) (*mongo.Config, errors.CCErrorCoder) {
	lastConfigErr = nil
	config, err := cc.Mongo(prefix)
	if err != nil {
		return nil, errors.NewCCError(common.CCErrCommConfMissItem, "can't find mongo configuration")
	}
	if config.Address == "" {
		lastConfigErr = errors.NewCCError(common.CCErrCommConfMissItem,
			"Configuration file missing ["+prefix+".host] configuration item")
		return nil, lastConfigErr
	}
	if config.User == "" {
		lastConfigErr = errors.NewCCError(common.CCErrCommConfMissItem,
			"Configuration file missing ["+prefix+".usr] configuration item")
		return nil, lastConfigErr
	}
	if config.Password == "" {
		lastConfigErr = errors.NewCCError(common.CCErrCommConfMissItem,
			"Configuration file missing ["+prefix+".pwd] configuration item")
		return nil, lastConfigErr
	}
	if config.Database == "" {
		lastConfigErr = errors.NewCCError(common.CCErrCommConfMissItem,
			"Configuration file missing ["+prefix+".database] configuration item")
		return nil, lastConfigErr
	}

	return &config, nil
}

func InitClient(prefix string, config *mongo.Config) errors.CCErrorCoder {
	lastInitErr = nil
	var dbErr error
	db, dbErr = local.NewMgo(config.GetMongoConf(), time.Minute)
	if dbErr != nil {
		blog.Errorf("failed to connect the mongo server, error info is %s", dbErr.Error())
		lastInitErr = errors.NewCCError(common.CCErrCommResourceInitFailed, "'"+prefix+".mongodb' initialization failed")
		return lastInitErr
	}
	return nil
}

func Validate() errors.CCErrorCoder {
	return nil
}

func UpdateConfig(prefix string, config mongo.Config) {
	// 不支持热更行
	return
}

func Healthz() (items []metric.HealthItem) {

	item := &metric.HealthItem{
		IsHealthy: true,
		Name:      types.CCFunctionalityMongo,
	}
	items = append(items, *item)
	if db == nil {
		item.IsHealthy = false
		item.Message = "not initialized"
		return
	}
	if err := db.Ping(); err != nil {
		item.IsHealthy = false
		item.Message = "connect error. err: " + err.Error()
		return
	}

	return
}

// GetDuplicateKey get duplicate key from error, if the error is not a duplicate error, returns the raw error message
// mongodb raw error format example:
// ...{E11000 duplicate key error collection: cmdb.cc_ObjectBase_... index: bkcc_unique_... dup key:
// { bk_inst_name: \"xxx\" }}]},...
func GetDuplicateKey(err error) string {
	if err == nil {
		return ""
	}

	errString := err.Error()
	if !strings.Contains(errString, "E11000 duplicate") {
		return errString
	}

	start := strings.Index(errString, "dup key: ")
	if start == -1 {
		return errString
	}
	start += len("dup key: ")

	end := strings.LastIndex(errString, "}]},")
	if end == -1 || end < start {
		return errString
	}

	return errString[start:end]
}

// GetDuplicateValue get duplicate Value from error, if the error is not a duplicate error, returns the raw error message
// mongodb raw error format example:
// Index build failed: ... E11000 duplicate key error collection: cmdb.cc_ObjectBase_0_pub_...:  dup key:
// dup key: { field: "xxxx" }
func GetDuplicateValue(field string, err error) string {
	if field == "" {
		return ""
	}
	if err == nil {
		return ""
	}

	errString := err.Error()
	if !strings.Contains(errString, "E11000 duplicate") {
		return errString
	}

	start := strings.Index(errString, "dup key: ")
	if start == -1 {
		return errString
	}
	start += len("dup key: { " + field + ": ")

	end := strings.LastIndex(errString, " }")
	if end == -1 || end < start {
		return errString
	}

	return errString[start:end]
}
