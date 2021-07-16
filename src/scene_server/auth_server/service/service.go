/*
 * Tencent is pleased to support the open source community by making 蓝鲸 available.
 * Copyright (C) 2017-2019 THL A29 Limited, a Tencent company. All rights reserved.
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
	"time"

	"configcenter/src/ac/iam"
	"configcenter/src/common"
	"configcenter/src/common/auth"
	"configcenter/src/common/backbone"
	cc "configcenter/src/common/backbone/configcenter"
	"configcenter/src/common/blog"
	"configcenter/src/common/http/rest"
	"configcenter/src/common/metadata"
	"configcenter/src/common/util"
	"configcenter/src/common/webservice/restfulservice"
	"configcenter/src/scene_server/auth_server/logics"
	sdkauth "configcenter/src/scene_server/auth_server/sdk/auth"
	"configcenter/src/scene_server/auth_server/sdk/client"
	"configcenter/src/scene_server/auth_server/types"
	"github.com/emicklei/go-restful"
)

type AuthService struct {
	engine     *backbone.Engine
	iamClient  client.Interface
	lgc        *logics.Logics
	authorizer sdkauth.Authorizer
}

func NewAuthService(engine *backbone.Engine, iamClient client.Interface, lgc *logics.Logics, authorizer sdkauth.Authorizer) *AuthService {
	return &AuthService{
		engine:     engine,
		iamClient:  iamClient,
		lgc:        lgc,
		authorizer: authorizer,
	}
}

func (s *AuthService) checkRequestFromIamFilter() func(req *restful.Request, resp *restful.Response, chain *restful.FilterChain) {
	return func(req *restful.Request, resp *restful.Response, chain *restful.FilterChain) {
		if !auth.EnableAuthorize() {
			chain.ProcessFilter(req, resp)
			return
		}

		isAuthorized, err := checkRequestAuthorization(s.iamClient, req.Request)
		if err != nil {
			rsp := metadata.BkBaseResp{
				Code:    types.InternalServerErrorCode,
				Message: err.Error(),
			}
			_ = resp.WriteAsJson(rsp)
			return
		}
		if !isAuthorized {
			rsp := metadata.BkBaseResp{
				Code:    types.UnauthorizedErrorCode,
				Message: "request not from iam",
			}
			_ = resp.WriteAsJson(rsp)
			return
		}

		// use iam request id as cc rid
		rid := req.Request.Header.Get(iam.IamRequestHeader)
		resp.Header().Set(iam.IamRequestHeader, rid)
		if rid != "" {
			req.Request.Header.Set(common.BKHTTPCCRequestID, rid)
		} else if rid = util.GetHTTPCCRequestID(req.Request.Header); rid == "" {
			rid = util.GenerateRID()
			req.Request.Header.Set(common.BKHTTPCCRequestID, rid)
		}
		resp.Header().Set(common.BKHTTPCCRequestID, rid)

		// use iam language as cc language
		req.Request.Header.Set(common.BKHTTPLanguage, req.Request.Header.Get("Blueking-Language"))

		req.Request.Header = util.SetHTTPReadPreference(req.Request.Header, common.SecondaryPreferredMode)

		// set supplierID
		setSupplierID(req.Request)

		user := req.Request.Header.Get(common.BKHTTPHeaderUser)
		if len(user) == 0 {
			req.Request.Header.Set(common.BKHTTPHeaderUser, "auth")
		}

		chain.ProcessFilter(req, resp)
		return
	}
}

var iamToken = struct {
	token            string
	tokenRefreshTime time.Time
}{}

func checkRequestAuthorization(iamClient client.Interface, req *http.Request) (bool, error) {
	rid := req.Header.Get(iam.IamRequestHeader)
	name, pwd, ok := req.BasicAuth()
	if !ok || name != iam.SystemIDIAM {
		blog.Errorf("request have no basic authorization, rid: %s", rid)
		return false, nil
	}
	// if cached token is set within a minute, use it to check request authorization
	if iamToken.token != "" && time.Since(iamToken.tokenRefreshTime) <= time.Minute && pwd == iamToken.token {
		return true, nil
	}
	var err error
	iamToken.token, err = iamClient.GetSystemToken(context.Background())
	if err != nil {
		blog.Errorf("check request authorization get system token failed, error: %s, rid: %s", err.Error(), rid)
		return false, err
	}
	iamToken.tokenRefreshTime = time.Now()
	if pwd == iamToken.token {
		return true, nil
	}
	blog.Errorf("request password not match system token, rid: %s", rid)
	return false, nil
}

// setSupplierID set suitable supplier account for the different version type, like ee, oa version
func setSupplierID(req *http.Request) {
	supplierID := req.Header.Get(common.BKHTTPOwnerID)
	if len(supplierID) == 0 {
		sID, _ := cc.String("authServer.supplierID")
		if len(sID) == 0 {
			sID = common.BKDefaultOwnerID
		}
		req.Header.Set(common.BKHTTPOwnerID, sID)
	}
}

func (s *AuthService) WebService() *restful.Container {
	api := new(restful.WebService)
	api.Path("/auth/v3")
	api.Filter(s.engine.Metric().RestfulMiddleWare)
	// only allows iam to pull resource using these api
	api.Filter(s.checkRequestFromIamFilter())
	api.Produces(restful.MIME_JSON)

	s.initResourcePull(api)

	container := restful.NewContainer()
	container.Add(api)

	authAPI := new(restful.WebService).Produces(restful.MIME_JSON)
	authAPI.Path("/ac/v3")
	s.initAuth(authAPI)
	container.Add(authAPI)

	// common api
	commonAPI := new(restful.WebService).Produces(restful.MIME_JSON)
	commonAPI.Route(commonAPI.GET("/healthz").To(s.Healthz))
	commonAPI.Route(commonAPI.GET("/version").To(restfulservice.Version))
	container.Add(commonAPI)

	return container
}

func (s *AuthService) initResourcePull(api *restful.WebService) {
	utility := rest.NewRestUtility(rest.Config{
		ErrorIf:  s.engine.CCErr,
		Language: s.engine.Language,
	})

	utility.AddHandler(rest.Action{Verb: http.MethodPost, Path: "/find/resource", Handler: s.PullResource})

	utility.AddToRestfulWebService(api)
}

func (s *AuthService) initAuth(api *restful.WebService) {
	utility := rest.NewRestUtility(rest.Config{
		ErrorIf:  s.engine.CCErr,
		Language: s.engine.Language,
	})

	utility.AddHandler(rest.Action{Verb: http.MethodPost, Path: "/authorize/batch", Handler: s.AuthorizeBatch})
	utility.AddHandler(rest.Action{Verb: http.MethodPost, Path: "/authorize/any/batch", Handler: s.AuthorizeAnyBatch})
	utility.AddHandler(rest.Action{Verb: http.MethodPost, Path: "/findmany/authorized_resource", Handler: s.ListAuthorizedResources})
	utility.AddHandler(rest.Action{Verb: http.MethodPost, Path: "/find/no_auth_skip_url", Handler: s.GetNoAuthSkipUrl})
	utility.AddHandler(rest.Action{Verb: http.MethodPost, Path: "/find/permission_to_apply", Handler: s.GetPermissionToApply})
	utility.AddHandler(rest.Action{Verb: http.MethodPost, Path: "/register/resource_creator_action", Handler: s.RegisterResourceCreatorAction})
	utility.AddHandler(rest.Action{Verb: http.MethodPost, Path: "/register/batch_resource_creator_action", Handler: s.BatchRegisterResourceCreatorAction})

	utility.AddToRestfulWebService(api)
}
