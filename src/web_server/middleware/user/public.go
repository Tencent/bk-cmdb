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

package user

import (
	"encoding/json"
	"net/http"
	"plugin"

	"configcenter/src/common"
	"configcenter/src/common/backbone"
	"configcenter/src/common/blog"
	"configcenter/src/common/metadata"
	"configcenter/src/common/util"
	"configcenter/src/web_server/app/options"
	"configcenter/src/web_server/middleware/user/plugins"

	"github.com/gin-gonic/gin"
	"github.com/holmeswang/contrib/sessions"
	"gopkg.in/redis.v5"
)

type publicUser struct {
	config   options.Config
	engine   *backbone.Engine
	cacheCli *redis.Client
	loginPlg *plugin.Plugin
}

// LoginUser  user login
func (m *publicUser) LoginUser(c *gin.Context) bool {
	rid := util.GetHTTPCCRequestID(c.Request.Header)

	isMultiOwner := false
	loginSuccess := false
	var userInfo *metadata.LoginUserInfo
	multipleOwner := m.config.Session.MultipleOwner
	if common.LoginSystemMultiSupplierTrue == multipleOwner {
		isMultiOwner = true
	}

	if m.loginPlg != nil {
		loginUserFunc, err := m.loginPlg.Lookup("LoginUser")
		if err != nil {
			blog.Errorf("look login func error, rid: %s", rid)
			return false
		}
		userInfo, loginSuccess = loginUserFunc.(func(c *gin.Context, config map[string]string, isMultiOwner bool) (user *metadata.LoginUserInfo, loginSuccess bool))(c, m.config.ConfigMap, isMultiOwner)
	} else {
		user := plugins.CurrentPlugin(c, m.config.LoginVersion)
		userInfo, loginSuccess = user.LoginUser(c, m.config.ConfigMap, isMultiOwner)
	}

	if !loginSuccess {
		blog.Infof("login user with plugin failed, rid: %s", rid)
		return false
	}
	if true == isMultiOwner || true == userInfo.MultiSupplier {
		ownerM := NewOwnerManager(userInfo.UserName, userInfo.OnwerUin, userInfo.Language)
		ownerM.CacheCli = m.cacheCli
		ownerM.Engine = m.engine
		// 初始化失败，不影响登录
		_, err := ownerM.InitOwner()
		if nil != err {
			blog.ErrorJSON("init onwer resource pool failed, err:%s, user:%s, rid: %s", err, userInfo, rid)
		}
	}
	strOwnerUinList := []byte("")
	if 0 != len(userInfo.OwnerUinArr) {
		strOwnerUinList, _ = json.Marshal(userInfo.OwnerUinArr)
	}

	session := sessions.Default(c)

	session.Set(common.WEBSessionUinKey, userInfo.UserName)
	session.Set(common.WEBSessionChineseNameKey, userInfo.ChName)
	session.Set(common.WEBSessionPhoneKey, userInfo.Phone)
	session.Set(common.WEBSessionEmailKey, userInfo.Email)
	session.Set(common.WEBSessionRoleKey, userInfo.Role)
	session.Set(common.HTTPCookieBKToken, userInfo.BkToken)
	session.Set(common.WEBSessionOwnerUinKey, userInfo.OnwerUin)
	session.Set(common.WEBSessionAvatarUrlKey, userInfo.AvatarUrl)
	session.Set(common.WEBSessionOwnerUinListeKey, string(strOwnerUinList))
	if userInfo.MultiSupplier {
		session.Set(common.WEBSessionMultiSupplierKey, common.LoginSystemMultiSupplierTrue)
	} else {
		session.Set(common.WEBSessionMultiSupplierKey, common.LoginSystemMultiSupplierFalse)
	}

	if err := session.Save(); err != nil {
		blog.Warnf("save session failed, err: %s, rid: %s", err.Error(), rid)
	}
	return true
}

// GetUserList get user list from PaaS
func (m *publicUser) GetUserList(c *gin.Context) (int, interface{}) {
	rid := util.GetHTTPCCRequestID(c.Request.Header)
	var err error
	var userList []*metadata.LoginSystemUserInfo
	rspBody := metadata.LonginSystemUserListResult{}
	rspBody.Result = true
	httpStatus := http.StatusOK
	if nil == m.loginPlg {
		user := plugins.CurrentPlugin(c, m.config.LoginVersion)
		userList, err = user.GetUserListForFront(c, m.config.ConfigMap)
	} else {
		getUserListFunc, err := m.loginPlg.Lookup("GetUserList")
		if nil != err {
			blog.Error("GetUserList interface not implemented, rid: %s", rid)
			rspBody.Code = common.CCErrCommHTTPDoRequestFailed
			rspBody.ErrMsg = err.Error()
			return http.StatusInternalServerError, rspBody

		}
		userList, err = getUserListFunc.(func(c *gin.Context, config map[string]string) ([]*metadata.LoginSystemUserInfo, error))(c, m.config.ConfigMap)
	}
	if nil != err {
		blog.Error("GetUserList failed, err: %+v, rid: %s", err, rid)
		rspBody.Code = common.CCErrCommHTTPDoRequestFailed
		rspBody.ErrMsg = err.Error()
		rspBody.Result = false
		httpStatus = http.StatusInternalServerError
	}
	rspBody.Result = true
	rspBody.Data = userList
	return httpStatus, rspBody
}

func (m *publicUser) GetLoginUrl(c *gin.Context) string {
	rid := util.GetHTTPCCRequestID(c.Request.Header)

	params := new(metadata.LogoutRequestParams)
	err := json.NewDecoder(c.Request.Body).Decode(params)
	if nil != err || (common.LogoutHTTPSchemeHTTP != params.HTTPScheme && common.LogoutHTTPSchemeHTTPS != params.HTTPScheme) {
		params.HTTPScheme, err = c.Cookie(common.LogoutHTTPSchemeCookieKey)
		if nil != err || (common.LogoutHTTPSchemeHTTP != params.HTTPScheme && common.LogoutHTTPSchemeHTTPS != params.HTTPScheme) {
			params.HTTPScheme = common.LogoutHTTPSchemeHTTP
		}
	}

	if nil == m.loginPlg {
		user := plugins.CurrentPlugin(c, m.config.LoginVersion)
		return user.GetLoginUrl(c, m.config.ConfigMap, params)
	} else {

		getLoginUrlFunc, err := m.loginPlg.Lookup("GetLoginUrl")

		if nil != err {
			blog.Errorf("look get url func error, rid: %s", rid)
			return ""

		}
		return getLoginUrlFunc.(func(c *gin.Context, config map[string]string, input *metadata.LogoutRequestParams) string)(c, m.config.ConfigMap, params)

	}

}
