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
	"context"
	"net/http"

	"configcenter/src/ac"
	"configcenter/src/common/backbone"
	"configcenter/src/common/cryptor"
	"configcenter/src/common/errors"
	"configcenter/src/common/http/rest"
	"configcenter/src/common/rdapi"
	"configcenter/src/common/webservice/restfulservice"
	"configcenter/src/scene_server/cloud_server/logics"
	"github.com/emicklei/go-restful"
)

type Service struct {
	*backbone.Engine
	ctx     context.Context
	cryptor cryptor.Cryptor
	*logics.Logics
	EnableTxn  bool
	authorizer ac.AuthorizeInterface
}

func NewService(ctx context.Context) *Service {
	return &Service{
		ctx: ctx,
	}
}

func (s *Service) SetEncryptor(cryptor cryptor.Cryptor) {
	s.cryptor = cryptor
}

func (s *Service) SetAuthorizer(authorizer ac.AuthorizeInterface) {
	s.authorizer = authorizer
}

func (s *Service) WebService() *restful.Container {

	api := new(restful.WebService)
	api.Path("/cloud/v3")
	api.Filter(s.Engine.Metric().RestfulMiddleWare)
	getErrFunc := func() errors.CCErrorIf {
		return s.Engine.CCErr
	}
	api.Filter(rdapi.AllGlobalFilter(getErrFunc))
	api.Produces(restful.MIME_JSON)

	s.initRoute(api)
	container := restful.NewContainer()
	container.Add(api)

	// common api
	commonAPI := new(restful.WebService).Produces(restful.MIME_JSON)
	commonAPI.Route(commonAPI.GET("/healthz").To(s.Healthz))
	commonAPI.Route(commonAPI.GET("/version").To(restfulservice.Version))
	container.Add(commonAPI)

	return container
}

func (s *Service) initRoute(api *restful.WebService) {
	utility := rest.NewRestUtility(rest.Config{
		ErrorIf:  s.Engine.CCErr,
		Language: s.Engine.Language,
	})

	// cloud account
	utility.AddHandler(rest.Action{Verb: http.MethodPost, Path: "/cloud/account/verify", Handler: s.VerifyConnectivity})
	utility.AddHandler(rest.Action{Verb: http.MethodPost, Path: "/findmany/cloud/account/validity", Handler: s.SearchAccountValidity})
	utility.AddHandler(rest.Action{Verb: http.MethodPost, Path: "/create/cloud/account", Handler: s.CreateAccount})
	utility.AddHandler(rest.Action{Verb: http.MethodPost, Path: "/findmany/cloud/account", Handler: s.SearchAccount})
	utility.AddHandler(rest.Action{Verb: http.MethodPut, Path: "/update/cloud/account/{bk_account_id}", Handler: s.UpdateAccount})
	utility.AddHandler(rest.Action{Verb: http.MethodDelete, Path: "/delete/cloud/account/{bk_account_id}", Handler: s.DeleteAccount})

	// cloud sync task
	utility.AddHandler(rest.Action{Verb: http.MethodPost, Path: "/findmany/cloud/account/vpc/{bk_account_id}", Handler: s.SearchVpc})
	utility.AddHandler(rest.Action{Verb: http.MethodPost, Path: "/create/cloud/sync/task", Handler: s.CreateSyncTask})
	utility.AddHandler(rest.Action{Verb: http.MethodPost, Path: "/findmany/cloud/sync/task", Handler: s.SearchSyncTask})
	utility.AddHandler(rest.Action{Verb: http.MethodPut, Path: "/update/cloud/sync/task/{bk_task_id}", Handler: s.UpdateSyncTask})
	utility.AddHandler(rest.Action{Verb: http.MethodDelete, Path: "/delete/cloud/sync/task/{bk_task_id}", Handler: s.DeleteSyncTask})
	utility.AddHandler(rest.Action{Verb: http.MethodPost, Path: "/findmany/cloud/sync/history", Handler: s.SearchSyncHistory})
	utility.AddHandler(rest.Action{Verb: http.MethodPost, Path: "/findmany/cloud/sync/region", Handler: s.SearchSyncRegion})

	utility.AddToRestfulWebService(api)
}
