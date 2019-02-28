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

package auth

import (
	"context"

	"configcenter/src/apimachinery/util"
	"configcenter/src/auth/authcenter"
	"configcenter/src/auth/meta"
)

type Authorize interface {
	Authorizer
	ResourceHandler
}

type Authorizer interface {
	// Authorize works to check if a user has the authority to operate resources.
	Authorize(ctx context.Context, a *meta.Attribute) (decision meta.Decision, err error)
}

// ResourceManager is used to handle the resources register to authorize center.
// request id is a identifier for a request, returned by IAM.
type ResourceHandler interface {
	// register a resource
	Register(ctx context.Context, r *meta.ResourceAttribute) error
	// deregister a resource
	Deregister(ctx context.Context, r *meta.ResourceAttribute) error
	// update a resource's info
	Update(ctx context.Context, r *meta.ResourceAttribute) error
	// get a resource's info
	Get(ctx context.Context) error
}

// NewAuthorize is used to initialized a Authorize instance interface,
// which is used for request authorize and resource handle.
// This allows bk-cmdb to support other kind of auth center.
// tls can be nil if it is not care.
// authConfig is a way to parse configuration info for the connection to a auth center.
func NewAuthorize(tls *util.TLSClientConfig, authConfig map[string]string) (Authorize, error) {
	return authcenter.NewAuthCenter(tls, authConfig)
}
