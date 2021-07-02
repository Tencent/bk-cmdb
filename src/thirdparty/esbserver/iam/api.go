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
	"fmt"
	"net/http"

	"configcenter/src/common/metadata"
	"configcenter/src/thirdparty/esbserver/esbutil"
)

// returns the url which can helps to launch the bk-iam when user do not have the authority to access resource(s).
func (i *iam) GetNoAuthSkipUrl(ctx context.Context, header http.Header, p metadata.IamPermission) (string, error) {
	resp := new(esbIamPermissionURLResp)
	url := "/v2/iam/application/"
	params := &esbIamPermissionParams{
		EsbCommParams: esbutil.GetEsbRequestParams(i.config.GetConfig(), header),
		IamPermission: p,
	}

	err := i.client.Post().
		SubResourcef(url).
		WithContext(ctx).
		WithHeaders(header).
		Body(params).
		Do().
		Into(&resp)
	if err != nil {
		return "", err
	}
	if !resp.Result || resp.Code != 0 {
		return "", fmt.Errorf("code: %d, message: %s", resp.Code, resp.Message)
	}

	return resp.Data.Url, nil
}

// RegisterResourceCreatorAction register iam resource instance with creator, returns related actions with policy id that the creator gained
func (i *iam) RegisterResourceCreatorAction(ctx context.Context, header http.Header, instance metadata.IamInstanceWithCreator) (
	[]metadata.IamCreatorActionPolicy, error) {

	resp := new(esbIamCreatorActionResp)
	url := "/v2/iam/authorization/resource_creator_action/"
	params := &esbIamInstanceParams{
		EsbCommParams:          esbutil.GetEsbRequestParams(i.config.GetConfig(), header),
		IamInstanceWithCreator: instance,
	}

	err := i.client.Post().
		SubResourcef(url).
		WithContext(ctx).
		WithHeaders(header).
		Body(params).
		Do().
		Into(&resp)
	if err != nil {
		return nil, err
	}
	if !resp.Result || resp.Code != 0 {
		return nil, fmt.Errorf("code: %d, message: %s", resp.Code, resp.Message)
	}

	return resp.Data, nil
}

// BatchRegisterResourceCreatorAction batch register iam resource instances with creator, returns related actions with policy id that the creator gained
func (i *iam) BatchRegisterResourceCreatorAction(ctx context.Context, header http.Header, instances metadata.IamInstancesWithCreator) (
	[]metadata.IamCreatorActionPolicy, error) {

	resp := new(esbIamCreatorActionResp)
	url := "/v2/iam/authorization/batch_resource_creator_action/"
	params := &esbIamInstancesParams{
		EsbCommParams:           esbutil.GetEsbRequestParams(i.config.GetConfig(), header),
		IamInstancesWithCreator: instances,
	}

	err := i.client.Post().
		SubResourcef(url).
		WithContext(ctx).
		WithHeaders(header).
		Body(params).
		Do().
		Into(&resp)
	if err != nil {
		return nil, err
	}
	if !resp.Result || resp.Code != 0 {
		return nil, fmt.Errorf("code: %d, message: %s", resp.Code, resp.Message)
	}

	return resp.Data, nil
}

// BatchOperateInstanceAuth batch grant or revoke iam resource instances' authorization
func (i *iam) BatchOperateInstanceAuth(ctx context.Context, header http.Header, req *metadata.IamBatchOperateInstanceAuthReq) (
	[]metadata.IamBatchOperateInstanceAuthRes, error) {

	resp := new(esbIamBatchOperateInstanceAuthResp)
	url := "/v2/iam/authorization/batch_instance/"
	params := &esbIamBatchOperateInstanceAuthParams{
		EsbCommParams:                  esbutil.GetEsbRequestParams(i.config.GetConfig(), header),
		IamBatchOperateInstanceAuthReq: req,
	}

	err := i.client.Post().
		SubResourcef(url).
		WithContext(ctx).
		WithHeaders(header).
		Body(params).
		Do().
		Into(&resp)
	if err != nil {
		return nil, err
	}
	if !resp.Result || resp.Code != 0 {
		return nil, fmt.Errorf("code: %d, message: %s", resp.Code, resp.Message)
	}

	return resp.Data, nil
}
