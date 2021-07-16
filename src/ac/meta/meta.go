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

package meta

type AuthAttribute struct {
	User      UserInfo
	Resources []ResourceAttribute
}

type UserInfo struct {
	// the name of this user.
	UserName string
	// the supplier id that this user belongs to.
	SupplierAccount string
}

type Item Basic
type Layers []Item

// ResourceAttribute represent one iam resource
type ResourceAttribute struct {
	Basic

	SupplierAccount string `json:"supplier_account"`
	BusinessID      int64  `json:"business_id"`
	// if this object belongs to a topology, like mainline topology,
	// layers means each object's item before this object.
	Layers Layers `json:"layers"`
}

// Basic defines the basic info for a resource.
type Basic struct {
	// the name of the affiliated resource, which could be a model name.
	Type ResourceType `json:"type"`

	// the action that user want to do with this resource.
	// this field should be empty when it's used in resource handle operation.
	Action Action `json:"action"`

	// the name of the resource, which could be a bk-route, etc.
	// this filed is not necessary for all the resources.
	Name string `json:"name"`

	// the instance id of this resource, which could be a model's instance id.
	InstanceID int64

	// InstanceIDEx is a extend for instanceID which can only be integer, but some resources only have string format id.
	InstanceIDEx string
}

// BackendResourceLayer represent one resource layer
type BackendResourceLayer struct {
	ResourceType string `json:"resource_type"`
	ResourceID   string `json:"resource_id,omitempty"`
	ResourceName string `json:"resource_name"`
}

// BackendResource represent a resource in auth backend, like iam.
type BackendResource []BackendResourceLayer

// CommonInfo contains common field which can be extracted from restful.Request
type CommonInfo struct {
	User UserInfo
}

type Decision struct {
	// the authorize decision, whether a user has been authorized or not.
	Authorized bool

	// the detailed reason for this authorize.
	Reason string
}

type ListAuthorizedResourcesParam struct {
	UserName     string       `json:"user_name"`
	BizID        int64        `json:"bk_biz_id"`
	ResourceType ResourceType `json:"resource_type"`
	Action       Action       `json:"action"`
}

type Action string

func (a Action) String() string {
	return string(a)
}

const (
	Create     Action = "create"
	CreateMany Action = "createMany"
	Update     Action = "update"
	UpdateMany Action = "updateMany"
	Delete     Action = "delete"
	DeleteMany Action = "deleteMany"
	Archive    Action = "archive"
	Find       Action = "find"
	FindMany   Action = "findMany"
	// unknown action, which is also unsupported actions.
	Unknown     Action = "unknown"
	EmptyAction Action = "" // used for register resources
	SkipAction  Action = "skip"

	Execute Action = "execute"

	// move resource pool hosts to a business idle module
	MoveResPoolHostToBizIdleModule Action = "moveResPoolHostToBizIdleModule"
	MoveResPoolHostToDirectory     Action = "moveResPoolHostToDirectory"
	AddHostToResourcePool          Action = "addHostToResourcePool"
	MoveBizHostFromModuleToResPool Action = "moveHostFromModuleToResPool"
	MoveHostToAnotherBizModule     Action = "moveHostToAnotherBizModule"

	// system base
	ModelTopologyView      Action = "modelTopologyView"
	ModelTopologyOperation Action = "modelTopologyOperation"

	// event watch
	WatchHost             Action = "host"
	WatchHostRelation     Action = "host_relation"
	WatchBiz              Action = "biz"
	WatchSet              Action = "set"
	WatchModule           Action = "module"
	WatchProcess          Action = "process"
	WatchCommonInstance   Action = "object_instance"
	WatchMainlineInstance Action = "mainline_instance"

	// can view business related resources, including business and business collection resources
	ViewBusinessResource Action = "viewBusinessResource"
)
