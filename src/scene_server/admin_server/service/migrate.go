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
	"encoding/json"
	"fmt"
	"net/http"
	"path/filepath"
	"time"

	"configcenter/src/common"
	"configcenter/src/common/blog"
	"configcenter/src/common/mapstr"
	"configcenter/src/common/metadata"
	"configcenter/src/common/types"
	"configcenter/src/common/util"
	"configcenter/src/common/version"
	"configcenter/src/common/watch"
	"configcenter/src/scene_server/admin_server/upgrader"
	"configcenter/src/source_controller/cacheservice/event"
	daltypes "configcenter/src/storage/dal/types"
	streamtypes "configcenter/src/storage/stream/types"

	"github.com/emicklei/go-restful"
)

func (s *Service) migrate(req *restful.Request, resp *restful.Response) {
	rHeader := req.Request.Header
	rid := util.GetHTTPCCRequestID(rHeader)
	defErr := s.CCErr.CreateDefaultCCErrorIf(util.GetLanguage(rHeader))
	ownerID := common.BKDefaultOwnerID
	updateCfg := &upgrader.Config{
		OwnerID: ownerID,
		User:    common.CCSystemOperatorUserName,
	}

	if err := s.createWatchDBChainCollections(rid); err != nil {
		blog.Errorf("create watch db chain collections failed, err: %v, rid: %s", err, rid)
		result := &metadata.RespError{
			Msg: defErr.Errorf(common.CCErrCommMigrateFailed, err.Error()),
		}
		resp.WriteError(http.StatusInternalServerError, result)
		return
	}

	preVersion, finishedVersions, err := upgrader.Upgrade(s.ctx, s.db, s.cache, updateCfg)
	if err != nil {
		blog.Errorf("db upgrade failed, err: %+v, rid: %s", err, rid)
		result := &metadata.RespError{
			Msg: defErr.Errorf(common.CCErrCommMigrateFailed, err.Error()),
		}
		resp.WriteError(http.StatusInternalServerError, result)
		return
	}

	currentVersion := preVersion
	if len(finishedVersions) > 0 {
		currentVersion = finishedVersions[len(finishedVersions)-1]
	}

	result := MigrationResponse{
		BaseResp: metadata.BaseResp{
			Result:      true,
			Code:        0,
			ErrMsg:      "",
			Permissions: nil,
		},
		Data:             "migrate success",
		PreVersion:       preVersion,
		CurrentVersion:   currentVersion,
		FinishedVersions: finishedVersions,
	}
	resp.WriteEntity(result)
}

// dbChainTTLTime the ttl time seconds of the db event chain, used to set the ttl index of mongodb
const dbChainTTLTime = 5 * 24 * 60 * 60

func (s *Service) createWatchDBChainCollections(rid string) error {
	// create watch token table to store the last watch token info for every collections
	exists, err := s.watchDB.HasTable(s.ctx, common.BKTableNameWatchToken)
	if err != nil {
		blog.Errorf("check if table %s exists failed, err: %v, rid: %s", common.BKTableNameWatchToken, err, rid)
		return err
	}

	if !exists {
		if err = s.watchDB.CreateTable(s.ctx, common.BKTableNameWatchToken); err != nil && !s.watchDB.IsDuplicatedError(err) {
			blog.Errorf("create table %s failed, err: %v, rid: %s", common.BKTableNameWatchToken, err, rid)
			return err
		}
	}

	// create watch chain node table and init the last token info as empty for all collections
	cursorTypes := watch.ListCursorTypes()
	for _, cursorType := range cursorTypes {
		key, err := event.GetResourceKeyWithCursorType(cursorType)
		if err != nil {
			blog.Errorf("get resource key with cursor type %s failed, err: %v, rid: %s", cursorType, err, rid)
			return err
		}

		exists, err := s.watchDB.HasTable(s.ctx, key.ChainCollection())
		if err != nil {
			blog.Errorf("check if table %s exists failed, err: %v, rid: %s", key.ChainCollection(), err, rid)
			return err
		}

		if !exists {
			if err = s.watchDB.CreateTable(s.ctx, key.ChainCollection()); err != nil && !s.watchDB.IsDuplicatedError(err) {
				blog.Errorf("create table %s failed, err: %v, rid: %s", key.ChainCollection(), err, rid)
				return err
			}
		}

		indexes := []daltypes.Index{
			{Name: "index_id", Keys: map[string]int32{common.BKFieldID: -1}, Background: true, Unique: true},
			{Name: "index_cursor", Keys: map[string]int32{common.BKCursorField: -1}, Background: true, Unique: true},
			{Name: "index_cluster_time", Keys: map[string]int32{common.BKClusterTimeField: -1}, Background: true,
				ExpireAfterSeconds: dbChainTTLTime},
		}

		if cursorType == watch.ObjectBase || cursorType == watch.MainlineInstance {
			subResourceIndex := daltypes.Index{
				Name: "index_sub_resource", Keys: map[string]int32{common.BKSubResourceField: 1}, Background: true,
			}
			indexes = append(indexes, subResourceIndex)
		}

		existIndexArr, err := s.watchDB.Table(key.ChainCollection()).Indexes(s.ctx)
		if err != nil {
			blog.Errorf("get exist indexes for table %s failed, err: %v, rid: %s", key.ChainCollection(), err, rid)
			return err
		}

		existIdxMap := make(map[string]bool)
		for _, index := range existIndexArr {
			existIdxMap[index.Name] = true
		}

		for _, index := range indexes {
			if _, exist := existIdxMap[index.Name]; exist {
				continue
			}

			err = s.watchDB.Table(key.ChainCollection()).CreateIndex(s.ctx, index)
			if err != nil && !s.watchDB.IsDuplicatedError(err) {
				blog.Errorf("create indexes for table %s failed, err: %v, rid: %s", key.ChainCollection(), err, rid)
				return err
			}
		}

		filter := map[string]interface{}{
			"_id": key.Collection(),
		}

		count, err := s.watchDB.Table(common.BKTableNameWatchToken).Find(filter).Count(s.ctx)
		if err != nil {
			blog.Errorf("check if last watch token exists failed, err: %v, filter: %+v", err, filter)
			return err
		}

		if count > 0 {
			continue
		}

		if key.Collection() == event.HostIdentityKey.Collection() {
			// host identity's watch token is different with other identity.
			// only set coll is ok, the other fields is useless
			data := mapstr.MapStr{
				"_id":                              key.Collection(),
				common.BKTableNameBaseHost:         watch.LastChainNodeData{Coll: common.BKTableNameBaseHost},
				common.BKTableNameModuleHostConfig: watch.LastChainNodeData{Coll: common.BKTableNameModuleHostConfig},
				common.BKTableNameBaseProcess:      watch.LastChainNodeData{Coll: common.BKTableNameBaseProcess},
			}
			if err := s.watchDB.Table(common.BKTableNameWatchToken).Insert(s.ctx, data); err != nil {
				blog.Errorf("init last watch token failed, err: %v, data: %+v", err, data)
				return err
			}
			continue
		}

		data := watch.LastChainNodeData{
			Coll:  key.Collection(),
			Token: "",
			StartAtTime: streamtypes.TimeStamp{
				Sec:  uint32(time.Now().Unix()),
				Nano: 0,
			},
		}
		if err := s.watchDB.Table(common.BKTableNameWatchToken).Insert(s.ctx, data); err != nil {
			blog.Errorf("init last watch token failed, err: %v, data: %+v", err, data)
			return err
		}
	}

	return nil
}

func (s *Service) migrateSpecifyVersion(req *restful.Request, resp *restful.Response) {
	rHeader := req.Request.Header
	rid := util.GetHTTPCCRequestID(rHeader)
	defErr := s.CCErr.CreateDefaultCCErrorIf(util.GetLanguage(rHeader))
	ownerID := common.BKDefaultOwnerID
	updateCfg := &upgrader.Config{
		OwnerID: ownerID,
		User:    common.CCSystemOperatorUserName,
	}

	input := new(MigrateSpecifyVersionRequest)
	if err := json.NewDecoder(req.Request.Body).Decode(input); err != nil {
		blog.Errorf("migrateSpecifyVersion failed, decode body err: %v, body:%+v,rid:%s", err, req.Request.Body, rid)
		_ = resp.WriteError(http.StatusOK, &metadata.RespError{Msg: defErr.Error(common.CCErrCommJSONUnmarshalFailed)})
		return
	}

	if input.CommitID != version.CCGitHash {
		_ = resp.WriteError(http.StatusOK, &metadata.RespError{Msg: defErr.Errorf(common.CCErrCommParamsInvalid, "commit_id")})
		return
	}

	err := upgrader.UpgradeSpecifyVersion(s.ctx, s.db, s.cache, updateCfg, input.Version)
	if err != nil {
		blog.Errorf("db upgrade specify failed, err: %+v, rid: %s", err, rid)
		result := &metadata.RespError{
			Msg: defErr.Errorf(common.CCErrCommMigrateFailed, err.Error()),
		}
		resp.WriteError(http.StatusInternalServerError, result)
		return
	}

	result := MigrationResponse{
		BaseResp: metadata.BaseResp{
			Result:      true,
			Code:        0,
			ErrMsg:      "",
			Permissions: nil,
		},
		Data: "migrate success. version: " + input.Version,
	}
	resp.WriteEntity(result)

}

var allConfigNames = map[string]bool{
	"redis":    true,
	"mongodb":  true,
	"common":   true,
	"extra":    true,
	"error":    true,
	"language": true,
	"all":      true,
}

var configHelpInfo = fmt.Sprintf("config_name must be one of the [redis, mongodb, common, extra, error, language, all]")

func (s *Service) refreshConfig(req *restful.Request, resp *restful.Response) {
	rHeader := req.Request.Header
	rid := util.GetHTTPCCRequestID(rHeader)
	defErr := s.CCErr.CreateDefaultCCErrorIf(util.GetLanguage(rHeader))

	input := new(struct {
		ConfigName string `json:"config_name"`
	})
	if err := json.NewDecoder(req.Request.Body).Decode(input); err != nil {
		blog.Errorf("refreshConfig failed, decode body err: %v ,body:%+v,rid:%s", err, req.Request.Body, rid)
		resp.WriteError(http.StatusOK, &metadata.RespError{Msg: defErr.Error(common.CCErrCommJSONUnmarshalFailed)})
		return
	}

	configName := "all"
	if input.ConfigName != "" {
		if ok := allConfigNames[input.ConfigName]; !ok {
			blog.Errorf("refreshConfig failed, config_name is wrong, %s, input:%#v, rid:%s", configHelpInfo, input, rid)
			resp.WriteError(http.StatusOK, &metadata.RespError{Msg: defErr.Errorf(common.CCErrCommParamsInvalid, configHelpInfo)})
			return
		}
		configName = input.ConfigName
	}

	var err error
	switch configName {
	case "redis", "mongodb", "common", "extra":
		filePath := filepath.Join(s.Config.Configures.Dir, configName+".yaml")
		key := types.CC_SERVCONF_BASEPATH + "/" + configName
		err = s.ConfigCenter.WriteConfigure(filePath, key)
	case "error":
		err = s.ConfigCenter.WriteErrorRes2Center(s.Config.Errors.Res)
	case "language":
		err = s.ConfigCenter.WriteLanguageRes2Center(s.Config.Language.Res)
	case "all":
		err = s.ConfigCenter.WriteAllConfs2Center(s.Config.Configures.Dir, s.Config.Errors.Res, s.Config.Language.Res)
	default:
		blog.Errorf("refreshConfig failed, config_name is wrong, %s, input:%#v, rid:%s", configHelpInfo, input, rid)
		resp.WriteError(http.StatusOK, &metadata.RespError{Msg: defErr.Errorf(common.CCErrCommParamsInvalid, configHelpInfo)})
		return
	}

	if err != nil {
		blog.Warnf("refreshConfig failed, input:%#v, error:%v, rid:%s", input, err, rid)
		resp.WriteError(http.StatusOK, &metadata.RespError{Msg: err})
	}

	blog.Infof("refresh config success, input:%#v", input)
	resp.WriteEntity(metadata.NewSuccessResp("refresh config success"))
}

type MigrationResponse struct {
	metadata.BaseResp `json:",inline"`
	Data              interface{} `json:"data"`
	PreVersion        string      `json:"pre_version"`
	CurrentVersion    string      `json:"current_version"`
	FinishedVersions  []string    `json:"finished_migrations"`
}

type MigrateSpecifyVersionRequest struct {
	CommitID  string `json:"commit_id"`
	TimeStamp int64  `json:"time_stamp"`
	Version   string `json:"version"`
}
