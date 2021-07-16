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

package metadata

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

type LoginUserInfoOwnerUinList struct {
	OwnerID    string `json:"id"`
	OwnerName  string `json:"name"`
	Role       int64  `json:"role"`
}

type LoginUserInfo struct {
	UserName      string                      `json:"username"`
	ChName        string                      `json:"chname"`
	Phone         string                      `json:"phone"`
	Email         string                      `json:"email"`
	Role          string                      `json:"-"`
	BkToken       string                      `json:"bk_token"`
	OnwerUin      string                      `json:"current_supplier"`
	OwnerUinArr   []LoginUserInfoOwnerUinList `json:"supplier_list"` //user all owner uin
	IsOwner       bool                        `json:"-"`             // is master
	Extra         map[string]interface{}      `json:"extra"`         //custom information
	Language      string                      `json:"-"`
	AvatarUrl     string                      `json:"avatar_url"`
	SupplierID    int64                       `json:"supplier_id"`
	MultiSupplier bool                        `json:"multi_supplier"`
}

type LoginPluginInfo struct {
	Name    string // plugin info
	Version string // In what version is used
	//CookieEnv string // Reserved word, not used now,  When the cookie has the current key, it is used preferentially.
	HandleFunc LoginUserPluginInerface
}

type LoginUserPluginParams struct {
	Url          string
	IsMultiOwner bool
	Cookie       []*http.Cookie // Reserved word, not used now
	Header       http.Header    // Reserved word, not used now
}

type LoginUserPluginInerface interface {
	LoginUser(c *gin.Context, config map[string]string, isMultiOwner bool) (user *LoginUserInfo, loginSucc bool)
	GetUserList(c *gin.Context, config map[string]string) ([]*LoginSystemUserInfo, error)
	GetUserListForFront(c *gin.Context, config map[string]string) ([]*LoginSystemUserInfo, error)
	GetLoginUrl(c *gin.Context, config map[string]string, input *LogoutRequestParams) string
}

type LoginSystemUserInfo struct {
	CnName string `json:"chinese_name"`
	EnName string `json:"english_name"`
}

type LonginSystemUserListResult struct {
	BaseResp `json:",inline"`
	Data     []*LoginSystemUserInfo `json:"data"`
}

type LoginUserInfoDetail struct {
	UserName      string                      `json:"username"`
	ChName        string                      `json:"chname"`
	OnwerUin      string                      `json:"current_supplier"`
	OwnerUinArr   []LoginUserInfoOwnerUinList `json:"supplier_list"` //user all owner uin
	AvatarUrl     string                      `json:"avatar_url"`
	MultiSupplier bool                        `json:"multi_supplier"`
}

type LoginUserInfoResult struct {
	BaseResp `json:",inline"`
	Data     LoginUserInfoDetail `json:"data"`
}

type LoginChangeSupplierResult struct {
	BaseResp `json:",inline"`
	Data     struct {
		ID string `json:"bk_supplier_account"`
	} `json:"data"`
}

type LogoutResult struct {
	BaseResp `json:",inline"`
	Data     struct {
		LogoutURL string `json:"url"`
	} `json:"data"`
}

type LogoutRequestParams struct {
	HTTPScheme string `json:"http_scheme"`
}

type ExcelAssocationOperate int

const (
	_ ExcelAssocationOperate = iota
	ExcelAssocationOperateError
	ExcelAssocationOperateAdd
	//ExcelAssocationOperateUpdate
	ExcelAssocationOperateDelete
)

type ExcelAssocation struct {
	ObjectAsstID string                 `json:"bk_obj_asst_id"`
	Operate      ExcelAssocationOperate `json:"operate"`
	SrcPrimary   string                 `json:"src_primary_key"`
	DstPrimary   string                 `json:"dst_primary_key"`
}
