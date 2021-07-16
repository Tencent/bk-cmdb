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

package topology

import (
	"context"
	"net/http"

	"configcenter/src/common"
	"configcenter/src/common/errors"
	"configcenter/src/common/metadata"
)

// ListBusiness list business with id list and return with a json array string which is []string json.
func (b *baseCache) ListBusiness(ctx context.Context, h http.Header, opt *metadata.ListWithIDOption) (
	jsonArray string, err error) {

	resp, err := b.client.Post().
		WithContext(ctx).
		Body(opt).
		SubResourcef("/findmany/cache/biz").
		WithHeaders(h).
		Do().
		IntoJsonString()

	if err != nil {
		return "", errors.New(common.CCErrCommHTTPDoRequestFailed, err.Error())
	}

	if !resp.Result {
		return "", errors.New(resp.Code, resp.ErrMsg)
	}

	return resp.Data, nil
}

// ListModules list modules with id list and return with a json array string which is []string json.
func (b *baseCache) ListModules(ctx context.Context, h http.Header, opt *metadata.ListWithIDOption) (
	jsonArray string, err error) {

	resp, err := b.client.Post().
		WithContext(ctx).
		Body(opt).
		SubResourcef("/findmany/cache/module").
		WithHeaders(h).
		Do().
		IntoJsonString()

	if err != nil {
		return "", errors.New(common.CCErrCommHTTPDoRequestFailed, err.Error())
	}

	if !resp.Result {
		return "", errors.New(resp.Code, resp.ErrMsg)
	}

	return resp.Data, nil
}

// ListSets list sets with id list and return with a json array string which is []string json.
func (b *baseCache) ListSets(ctx context.Context, h http.Header, opt *metadata.ListWithIDOption) (
	jsonArray string, err error) {

	resp, err := b.client.Post().
		WithContext(ctx).
		Body(opt).
		SubResourcef("/findmany/cache/set").
		WithHeaders(h).
		Do().
		IntoJsonString()

	if err != nil {
		return "", errors.New(common.CCErrCommHTTPDoRequestFailed, err.Error())
	}

	if !resp.Result {
		return "", errors.New(resp.Code, resp.ErrMsg)
	}

	return resp.Data, nil
}

func (b *baseCache) SearchBusiness(ctx context.Context, h http.Header, bizID int64) (string, error) {
	resp, err := b.client.Post().
		WithContext(ctx).
		Body(nil).
		SubResourcef("/find/cache/biz/%d", bizID).
		WithHeaders(h).
		Do().
		IntoJsonString()
	if err != nil {
		return "", errors.New(common.CCErrCommHTTPDoRequestFailed, err.Error())
	}
	if !resp.Result {
		return "", errors.New(resp.Code, resp.ErrMsg)
	}
	return resp.Data, nil
}

func (b *baseCache) SearchSet(ctx context.Context, h http.Header, setID int64) (string, error) {
	resp, err := b.client.Post().
		WithContext(ctx).
		Body(nil).
		SubResourcef("/find/cache/set/%d", setID).
		WithHeaders(h).
		Do().
		IntoJsonString()
	if err != nil {
		return "", errors.New(common.CCErrCommHTTPDoRequestFailed, err.Error())
	}
	if !resp.Result {
		return "", errors.New(resp.Code, resp.ErrMsg)
	}
	return resp.Data, nil
}

func (b *baseCache) SearchModule(ctx context.Context, h http.Header, moduleID int64) (string, error) {
	resp, err := b.client.Post().
		WithContext(ctx).
		Body(nil).
		SubResourcef("/find/cache/module/%d", moduleID).
		WithHeaders(h).
		Do().
		IntoJsonString()
	if err != nil {
		return "", errors.New(common.CCErrCommHTTPDoRequestFailed, err.Error())
	}
	if !resp.Result {
		return "", errors.New(resp.Code, resp.ErrMsg)
	}
	return resp.Data, nil
}

func (b *baseCache) SearchCustomLayer(ctx context.Context, h http.Header, objID string, instID int64) (string, error) {
	resp, err := b.client.Post().
		WithContext(ctx).
		Body(nil).
		SubResourcef("/find/cache/%s/%d", objID, instID).
		WithHeaders(h).
		Do().
		IntoJsonString()
	if err != nil {
		return "", errors.New(common.CCErrCommHTTPDoRequestFailed, err.Error())
	}
	if !resp.Result {
		return "", errors.New(resp.Code, resp.ErrMsg)
	}
	return resp.Data, nil
}
