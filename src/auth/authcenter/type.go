package authcenter

import (
	"fmt"

	"configcenter/src/common/metadata"
)

type AuthConfig struct {
	// blueking's auth center addresses
	Address []string
	// app code is used for authorize used.
	AppCode string
	// app secret is used for authorized
	AppSecret string
	// the system id that cmdb used in auth center.
	SystemID string
}

type RegisterInfo struct {
	CreatorType  string `json:"creator_type"`
	CreatorID    string `json:"creator_id"`
	ScopeInfo    `json:",inline"`
	ResourceInfo `json:",inline"`
}

type ResourceInfo struct {
	ResourceType ResourceTypeID `json:"resource_type"`
	// this filed is not always used, it's decided by the api
	// that is used.
	ResourceName string `json:"resource_name,omitempty"`
	ResourceID   string `json:"resource_id"`
}

type ScopeInfo struct {
	ScopeType string `json:"scope_type"`
	ScopeID   string `json:"scope_id"`
}

type ResourceResult struct {
	metadata.BaseResp `json:",inline"`
	RequestID         string       `json:"request_id"`
	Data              ResultStatus `json:"data"`
}

type ResultStatus struct {
	// for create resource result confirm use,
	// which true means register a resource success.
	IsCreated bool `json:"is_created"`
	// for deregister resource result confirm use,
	// which true means deregister success.
	IsDeleted bool `json:"is_deleted"`
	// for update resource result confirm use,
	// which true means update a resource success.
	IsUpdated bool `json:"is_updated"`
}

type DeregisterInfo struct {
	ScopeInfo    `json:",inline"`
	ResourceInfo `json:",inline"`
}

type UpdateInfo struct {
	ScopeInfo    `json:",inline"`
	ResourceInfo `json:",inline"`
}

type Principal struct {
	Type string `json:"principal_type"`
	ID   string `json:"principal_id"`
}

type AuthBatch struct {
	Principal       `json:",inline"`
	ScopeInfo       `json:",inline"`
	ResourceActions []ResourceAction `json:"resources_actions"`
}

type BatchResult struct {
	metadata.BaseResp `json:",inline"`
	RequestID         string        `json:"request_id"`
	Data              []BatchStatus `json:"data"`
}

type ResourceAction struct {
	ResourceInfo `json:",inline"`
	ActionID     ActionID `json:"action_id"`
}

type BatchStatus struct {
	ActionID     string `json:"action_id"`
	ResourceInfo `json:",inline"`
	// for authorize confirm use, define if a user have
	// the permission to this request.
	IsPass bool `json:"is_pass"`
}

type AuthError struct {
	RequestID string
	Reason    error
}

func (a *AuthError) Error() string {
	if len(a.RequestID) == 0 {
		return a.Reason.Error()
	}
	return fmt.Sprintf("request id: %s, err: %s", a.RequestID, a.Reason.Error())
}

type System struct {
	SystemID   string `json:"system_id,omitempty"`
	SystemName string `json:"system_name"`
	Desc       string `json:"desc"`
	// 可为空，在使用注册资源的方式时
	QueryInterface string `json:"query_interface"`
	//  关联的资源所属，有业务、全局、项目等
	ReleatedScopeTypes string `json:"releated_scope_types"`
	// 管理者，可通过权限中心产品页面修改模型相关信息
	Managers string `json:"managers"`
	// 更新者，可为system
	Updater string `json:"updater,omitempty"`
	// 创建者，可为system
	Creator string `json:"creator,omitempty"`
}

type ResourceType struct {
	ResourceTypeID       string   `json:"resource_type_id"`
	ResourceTypeName     string   `json:"resource_type_name"`
	ParentResourceTypeID string   `json:"parent_resource_type_id"`
	Actions              []Action `json:"actions"`
}

type Action struct {
	ActionID          ActionID `json:"action_id"`
	ActionName        string   `json:"action_name"`
	IsRelatedResource bool     `json:"is_related_resource"`
}

type SystemDetail struct {
	System
	Scopes []struct {
		ScopeTypeID   string         `json:"scope_type_id"`
		ResourceTypes []ResourceType `json:"resource_types"`
	} `json:"scopes"`
}

type BaseResponse struct {
	Code      int
	Message   string
	Result    bool
	RequestID string
}
