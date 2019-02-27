package authcenter

import (
	"configcenter/src/common/metadata"
	"fmt"
)

type AuthConfig struct {
	// blueking's auth center addresses
	Address []string
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
	ResourceType string `json:"resource_type"`
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
