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
package iam

import (
	"context"
	"net/http"

	"configcenter/src/apimachinery/rest"
	"configcenter/src/common/metadata"
	"configcenter/src/thirdparty/esbserver/esbutil"
)

type IamClientInterface interface {
	GetNoAuthSkipUrl(ctx context.Context, header http.Header, p metadata.IamPermission) (string, error)
	RegisterResourceCreatorAction(ctx context.Context, header http.Header, instance metadata.IamInstanceWithCreator) (
		[]metadata.IamCreatorActionPolicy, error)
	BatchRegisterResourceCreatorAction(ctx context.Context, header http.Header, instance metadata.IamInstancesWithCreator) (
		[]metadata.IamCreatorActionPolicy, error)
	BatchOperateInstanceAuth(ctx context.Context, header http.Header, req *metadata.IamBatchOperateInstanceAuthReq) (
		[]metadata.IamBatchOperateInstanceAuthRes, error)
}

func NewIamClientInterface(client rest.ClientInterface, config *esbutil.EsbConfigSrv) IamClientInterface {
	return &iam{
		client: client,
		config: config,
	}
}

type iam struct {
	config *esbutil.EsbConfigSrv
	client rest.ClientInterface
}

type esbIamPermissionParams struct {
	*esbutil.EsbCommParams
	metadata.IamPermission `json:",inline"`
}

type esbIamInstanceParams struct {
	*esbutil.EsbCommParams
	metadata.IamInstanceWithCreator `json:",inline"`
}

type esbIamInstancesParams struct {
	*esbutil.EsbCommParams
	metadata.IamInstancesWithCreator `json:",inline"`
}

type esbIamPermissionURLResp struct {
	Data struct {
		Url string `json:"url"`
	} `json:"data"`
	metadata.EsbBaseResponse `json:",inline"`
}

type esbIamCreatorActionResp struct {
	metadata.EsbBaseResponse `json:",inline"`
	Data                     []metadata.IamCreatorActionPolicy `json:"data"`
}

type esbIamBatchOperateInstanceAuthParams struct {
	*esbutil.EsbCommParams                   `json:",inline"`
	*metadata.IamBatchOperateInstanceAuthReq `json:",inline"`
}

type esbIamBatchOperateInstanceAuthResp struct {
	metadata.EsbBaseResponse `json:",inline"`
	Data                     []metadata.IamBatchOperateInstanceAuthRes `json:"data"`
}
