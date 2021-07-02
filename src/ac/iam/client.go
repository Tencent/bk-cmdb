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
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"configcenter/src/apimachinery/rest"
)

const (
	codeNotFound = 1901404
)

var (
	ErrNotFound = errors.New("Not Found")
)

type iamClient struct {
	config AuthConfig
	// http client instance
	client rest.ClientInterface
	// http header info
	basicHeader http.Header
}

type IAMClientCfg struct {
	Config AuthConfig
	// http client instance
	Client rest.ClientInterface
	// http header info
	BasicHeader http.Header
}

func NewIAMClient(cfg *IAMClientCfg) *iamClient {
	return &iamClient{
		config:      cfg.Config,
		client:      cfg.Client,
		basicHeader: cfg.BasicHeader,
	}
}

// iamClientInterface is a interface includes the api provided by IAM
// unexposed interface
type iamClientInterface interface {
	// RegisterSystem register a system in IAM
	RegisterSystem(ctx context.Context, sys System) error
	// GetSystemInfo get a system info from IAM
	// if fields is empty, find all system info
	GetSystemInfo(ctx context.Context, fields []SystemQueryField) (*SystemResp, error)
	// UpdateSystemConfig update system config in IAM
	UpdateSystemConfig(ctx context.Context, config *SysConfig) error

	// RegisterResourcesTypes register resource types in IAM
	RegisterResourcesTypes(ctx context.Context, resTypes []ResourceType) error
	// UpdateResourcesTypes update resource types in IAM
	UpdateResourcesTypes(ctx context.Context, resType ResourceType) error
	// DeleteResourcesTypes delete resource types in IAM
	DeleteResourcesTypes(ctx context.Context, resTypeIDs []TypeID) error

	// RegisterActions register actions in IAM
	RegisterActions(ctx context.Context, actions []ResourceAction) error
	// UpdateAction update action in IAM
	UpdateAction(ctx context.Context, action ResourceAction) error
	// DeleteActions delete actions in IAM
	DeleteActions(ctx context.Context, actionIDs []ActionID) error

	// RegisterActionGroups register action groups in IAM
	RegisterActionGroups(ctx context.Context, actionGroups []ActionGroup) error
	// UpdateActionGroups update action groups in IAM
	UpdateActionGroups(ctx context.Context, actionGroups []ActionGroup) error

	// RegisterInstanceSelections register instance selections in IAM
	RegisterInstanceSelections(ctx context.Context, instanceSelections []InstanceSelection) error
	// UpdateInstanceSelection update instance selection in IAM
	UpdateInstanceSelection(ctx context.Context, instanceSelection InstanceSelection) error
	// DeleteInstanceSelections delete instance selections in IAM
	DeleteInstanceSelections(ctx context.Context, instanceSelectionIDs []InstanceSelectionID) error

	// RegisterResourceCreatorActions regitser resource creator actions in IAM
	RegisterResourceCreatorActions(ctx context.Context, resourceCreatorActions ResourceCreatorActions) error
	// UpdateResourceCreatorActions update resource creator actions in IAM
	UpdateResourceCreatorActions(ctx context.Context, resourceCreatorActions ResourceCreatorActions) error

	// RegisterCommonActions register common actions in IAM
	RegisterCommonActions(ctx context.Context, commonActions []CommonAction) error
	// UpdateCommonActions update common actions in IAM
	UpdateCommonActions(ctx context.Context, commonActions []CommonAction) error

	// DeleteActionPolicies delete action policies in IAM
	DeleteActionPolicies(ctx context.Context, actionID ActionID) error

	// ListPolicies list action policies in IAM
	ListPolicies(ctx context.Context, params *ListPoliciesParams) (*ListPoliciesData, error)
}

func (c *iamClient) RegisterSystem(ctx context.Context, sys System) error {
	resp := new(BaseResponse)
	result := c.client.Post().
		SubResourcef("/api/v1/model/systems").
		WithContext(ctx).
		WithHeaders(c.basicHeader).
		Body(sys).Do()
	err := result.Into(resp)
	if err != nil {
		return err
	}

	if resp.Code != 0 {
		return &AuthError{
			RequestID: result.Header.Get(IamRequestHeader),
			Reason:    fmt.Errorf("register system failed, code: %d, msg:%s", resp.Code, resp.Message),
		}
	}

	return nil
}

// if fields is empty, find all system info
func (c *iamClient) GetSystemInfo(ctx context.Context, fields []SystemQueryField) (*SystemResp, error) {
	resp := new(SystemResp)
	fieldsStr := ""
	if len(fields) > 0 {
		fieldArr := make([]string, len(fields))
		for idx, field := range fields {
			fieldArr[idx] = string(field)
		}
		fieldsStr = strings.Join(fieldArr, ",")
	}

	result := c.client.Get().
		SubResourcef("/api/v1/model/systems/%s/query", c.config.SystemID).
		WithContext(ctx).
		WithHeaders(c.basicHeader).
		WithParam("fields", fieldsStr).
		Body(nil).Do()
	err := result.Into(resp)
	if err != nil {
		return nil, err
	}

	if resp.Code != 0 {
		if resp.Code == codeNotFound {
			return resp, ErrNotFound
		}
		return nil, &AuthError{
			RequestID: result.Header.Get(IamRequestHeader),
			Reason:    fmt.Errorf("get system info failed, code: %d, msg:%s", resp.Code, resp.Message),
		}
	}

	return resp, nil
}

// Note: can only update provider_config.host field.
func (c *iamClient) UpdateSystemConfig(ctx context.Context, config *SysConfig) error {
	sys := new(System)
	config.Auth = "basic"
	sys.ProviderConfig = config
	resp := new(BaseResponse)
	result := c.client.Put().
		SubResourcef("/api/v1/model/systems/%s", c.config.SystemID).
		WithContext(ctx).
		WithHeaders(c.basicHeader).
		Body(sys).Do()
	err := result.Into(resp)
	if err != nil {
		return err
	}

	if resp.Code != 0 {
		return &AuthError{
			RequestID: result.Header.Get(IamRequestHeader),
			Reason:    fmt.Errorf("update system config failed, code: %d, msg:%s", resp.Code, resp.Message),
		}
	}

	return nil
}

func (c *iamClient) RegisterResourcesTypes(ctx context.Context, resTypes []ResourceType) error {
	resp := new(BaseResponse)
	result := c.client.Post().
		SubResourcef("/api/v1/model/systems/%s/resource-types", c.config.SystemID).
		WithContext(ctx).
		WithHeaders(c.basicHeader).
		Body(resTypes).Do()
	err := result.Into(resp)
	if err != nil {
		return err
	}

	if resp.Code != 0 {
		return &AuthError{
			RequestID: result.Header.Get(IamRequestHeader),
			Reason:    fmt.Errorf("register system failed, code: %d, msg:%s", resp.Code, resp.Message),
		}
	}

	return nil

}

func (c *iamClient) UpdateResourcesTypes(ctx context.Context, resType ResourceType) error {
	resp := new(BaseResponse)
	result := c.client.Put().
		SubResourcef("/api/v1/model/systems/%s/resource-types/%s", c.config.SystemID, resType.ID).
		WithContext(ctx).
		WithHeaders(c.basicHeader).
		Body(resType).Do()
	err := result.Into(resp)
	if err != nil {
		return err
	}

	if resp.Code != 0 {
		return &AuthError{
			RequestID: result.Header.Get(IamRequestHeader),
			Reason:    fmt.Errorf("udpate resource type %s failed, code: %d, msg:%s", resType.ID, resp.Code, resp.Message),
		}
	}

	return nil
}

func (c *iamClient) DeleteResourcesTypes(ctx context.Context, resTypeIDs []TypeID) error {

	ids := make([]struct {
		ID TypeID `json:"id"`
	}, len(resTypeIDs))
	for idx := range resTypeIDs {
		ids[idx].ID = resTypeIDs[idx]
	}

	resp := new(BaseResponse)
	result := c.client.Delete().
		SubResourcef("/api/v1/model/systems/%s/resource-types", c.config.SystemID).
		WithContext(ctx).
		WithHeaders(c.basicHeader).
		Body(ids).Do()
	err := result.Into(resp)
	if err != nil {
		return err
	}

	if resp.Code != 0 {
		return &AuthError{
			RequestID: result.Header.Get(IamRequestHeader),
			Reason:    fmt.Errorf("delete resource type %v failed, code: %d, msg:%s", resTypeIDs, resp.Code, resp.Message),
		}
	}

	return nil
}

func (c *iamClient) RegisterActions(ctx context.Context, actions []ResourceAction) error {

	resp := new(BaseResponse)
	result := c.client.Post().
		SubResourcef("/api/v1/model/systems/%s/actions", c.config.SystemID).
		WithContext(ctx).
		WithHeaders(c.basicHeader).
		Body(actions).Do()
	err := result.Into(resp)
	if err != nil {
		return err
	}

	if resp.Code != 0 {
		return &AuthError{
			RequestID: result.Header.Get(IamRequestHeader),
			Reason:    fmt.Errorf("add resource actions %v failed, code: %d, msg:%s", actions, resp.Code, resp.Message),
		}
	}

	return nil
}

func (c *iamClient) UpdateAction(ctx context.Context, action ResourceAction) error {

	resp := new(BaseResponse)
	result := c.client.Put().
		SubResourcef("/api/v1/model/systems/%s/actions/%s", c.config.SystemID, action.ID).
		WithContext(ctx).
		WithHeaders(c.basicHeader).
		Body(action).Do()
	err := result.Into(resp)
	if err != nil {
		return err
	}

	if resp.Code != 0 {
		return &AuthError{
			RequestID: result.Header.Get(IamRequestHeader),
			Reason:    fmt.Errorf("udpate resource action %v failed, code: %d, msg:%s", action, resp.Code, resp.Message),
		}
	}

	return nil
}

func (c *iamClient) DeleteActions(ctx context.Context, actionIDs []ActionID) error {
	ids := make([]struct {
		ID ActionID `json:"id"`
	}, len(actionIDs))
	for idx := range actionIDs {
		ids[idx].ID = actionIDs[idx]
	}

	resp := new(BaseResponse)
	result := c.client.Delete().
		SubResourcef("/api/v1/model/systems/%s/actions", c.config.SystemID).
		WithContext(ctx).
		WithHeaders(c.basicHeader).
		Body(ids).Do()
	err := result.Into(resp)
	if err != nil {
		return err
	}

	if resp.Code != 0 {
		return &AuthError{
			RequestID: result.Header.Get(IamRequestHeader),
			Reason:    fmt.Errorf("delete resource actions %v failed, code: %d, msg:%s", actionIDs, resp.Code, resp.Message),
		}
	}

	return nil
}

func (c *iamClient) RegisterActionGroups(ctx context.Context, actionGroups []ActionGroup) error {

	resp := new(BaseResponse)
	result := c.client.Post().
		SubResourcef("/api/v1/model/systems/%s/configs/action_groups", c.config.SystemID).
		WithContext(ctx).
		WithHeaders(c.basicHeader).
		Body(actionGroups).Do()
	err := result.Into(resp)
	if err != nil {
		return err
	}

	if resp.Code != 0 {
		return &AuthError{
			RequestID: result.Header.Get(IamRequestHeader),
			Reason:    fmt.Errorf("register action groups %v failed, code: %d, msg:%s", actionGroups, resp.Code, resp.Message),
		}
	}

	return nil
}

func (c *iamClient) UpdateActionGroups(ctx context.Context, actionGroups []ActionGroup) error {

	resp := new(BaseResponse)
	result := c.client.Put().
		SubResourcef("/api/v1/model/systems/%s/configs/action_groups", c.config.SystemID).
		WithContext(ctx).
		WithHeaders(c.basicHeader).
		Body(actionGroups).Do()
	err := result.Into(resp)
	if err != nil {
		return err
	}

	if resp.Code != 0 {
		return &AuthError{
			RequestID: result.Header.Get(IamRequestHeader),
			Reason:    fmt.Errorf("update action groups %v failed, code: %d, msg:%s", actionGroups, resp.Code, resp.Message),
		}
	}

	return nil
}

func (c *iamClient) RegisterInstanceSelections(ctx context.Context, instanceSelections []InstanceSelection) error {

	resp := new(BaseResponse)
	result := c.client.Post().
		SubResourcef("/api/v1/model/systems/%s/instance-selections", c.config.SystemID).
		WithContext(ctx).
		WithHeaders(c.basicHeader).
		Body(instanceSelections).Do()
	err := result.Into(resp)
	if err != nil {
		return err
	}

	if resp.Code != 0 {
		return &AuthError{
			RequestID: result.Header.Get(IamRequestHeader),
			Reason:    fmt.Errorf("add instance selections %v failed, code: %d, msg:%s", instanceSelections, resp.Code, resp.Message),
		}
	}

	return nil
}

func (c *iamClient) UpdateInstanceSelection(ctx context.Context, instanceSelection InstanceSelection) error {

	resp := new(BaseResponse)
	result := c.client.Put().
		SubResourcef("/api/v1/model/systems/%s/instance-selections/%s", c.config.SystemID, instanceSelection.ID).
		WithContext(ctx).
		WithHeaders(c.basicHeader).
		Body(instanceSelection).Do()
	err := result.Into(resp)
	if err != nil {
		return err
	}

	if resp.Code != 0 {
		return &AuthError{
			RequestID: result.Header.Get(IamRequestHeader),
			Reason:    fmt.Errorf("udpate instance selections %v failed, code: %d, msg:%s", instanceSelection, resp.Code, resp.Message),
		}
	}

	return nil
}

func (c *iamClient) DeleteInstanceSelections(ctx context.Context, instanceSelectionIDs []InstanceSelectionID) error {
	ids := make([]struct {
		ID InstanceSelectionID `json:"id"`
	}, len(instanceSelectionIDs))
	for idx := range instanceSelectionIDs {
		ids[idx].ID = instanceSelectionIDs[idx]
	}

	resp := new(BaseResponse)
	result := c.client.Delete().
		SubResourcef("/api/v1/model/systems/%s/instance-selections", c.config.SystemID).
		WithContext(ctx).
		WithHeaders(c.basicHeader).
		Body(ids).Do()
	err := result.Into(resp)
	if err != nil {
		return err
	}

	if resp.Code != 0 {
		return &AuthError{
			RequestID: result.Header.Get(IamRequestHeader),
			Reason:    fmt.Errorf("delete instance selections %v failed, code: %d, msg:%s", instanceSelectionIDs, resp.Code, resp.Message),
		}
	}

	return nil
}

func (c *iamClient) RegisterResourceCreatorActions(ctx context.Context, resourceCreatorActions ResourceCreatorActions) error {

	resp := new(BaseResponse)
	result := c.client.Post().
		SubResourcef("/api/v1/model/systems/%s/configs/resource_creator_actions", c.config.SystemID).
		WithContext(ctx).
		WithHeaders(c.basicHeader).
		Body(resourceCreatorActions).Do()
	err := result.Into(resp)
	if err != nil {
		return err
	}

	if resp.Code != 0 {
		return &AuthError{
			RequestID: result.Header.Get(IamRequestHeader),
			Reason:    fmt.Errorf("register resource creator actions %v failed, code: %d, msg:%s", resourceCreatorActions, resp.Code, resp.Message),
		}
	}

	return nil
}

func (c *iamClient) UpdateResourceCreatorActions(ctx context.Context, resourceCreatorActions ResourceCreatorActions) error {

	resp := new(BaseResponse)
	result := c.client.Put().
		SubResourcef("/api/v1/model/systems/%s/configs/resource_creator_actions", c.config.SystemID).
		WithContext(ctx).
		WithHeaders(c.basicHeader).
		Body(resourceCreatorActions).Do()
	err := result.Into(resp)
	if err != nil {
		return err
	}

	if resp.Code != 0 {
		return &AuthError{
			RequestID: result.Header.Get(IamRequestHeader),
			Reason:    fmt.Errorf("update resource creator actions %v failed, code: %d, msg:%s", resourceCreatorActions, resp.Code, resp.Message),
		}
	}

	return nil
}

func (c *iamClient) RegisterCommonActions(ctx context.Context, commonActions []CommonAction) error {
	resp := new(BaseResponse)
	result := c.client.Post().
		SubResourcef("/api/v1/model/systems/%s/configs/common_actions", c.config.SystemID).
		WithContext(ctx).
		WithHeaders(c.basicHeader).
		Body(commonActions).Do()

	err := result.Into(resp)
	if err != nil {
		return err
	}

	if resp.Code != 0 {
		return &AuthError{
			RequestID: result.Header.Get(IamRequestHeader),
			Reason: fmt.Errorf("register common actions %v failed, code: %d, msg: %s", commonActions, resp.Code,
				resp.Message),
		}
	}

	return nil
}

func (c *iamClient) UpdateCommonActions(ctx context.Context, commonActions []CommonAction) error {
	resp := new(BaseResponse)
	result := c.client.Put().
		SubResourcef("/api/v1/model/systems/%s/configs/common_actions", c.config.SystemID).
		WithContext(ctx).
		WithHeaders(c.basicHeader).
		Body(commonActions).Do()

	err := result.Into(resp)
	if err != nil {
		return err
	}

	if resp.Code != 0 {
		return &AuthError{
			RequestID: result.Header.Get(IamRequestHeader),
			Reason: fmt.Errorf("update common actions %v failed, code: %d, msg: %s", commonActions, resp.Code,
				resp.Message),
		}
	}

	return nil
}

func (c *iamClient) DeleteActionPolicies(ctx context.Context, actionID ActionID) error {
	resp := new(BaseResponse)
	result := c.client.Delete().
		SubResourcef("/api/v1/model/systems/%s/actions/%s/policies", c.config.SystemID, actionID).
		WithContext(ctx).
		WithHeaders(c.basicHeader).
		Do()
	err := result.Into(resp)
	if err != nil {
		return err
	}

	if resp.Code != 0 {
		return &AuthError{
			RequestID: result.Header.Get(IamRequestHeader),
			Reason:    fmt.Errorf("delete action %s policies failed, code: %d, msg: %s", actionID, resp.Code, resp.Message),
		}
	}

	return nil
}

func (c *iamClient) ListPolicies(ctx context.Context, params *ListPoliciesParams) (*ListPoliciesData, error) {
	parsedParams := map[string]string{"action_id": string(params.ActionID)}
	if params.Page != 0 {
		parsedParams["page"] = strconv.FormatInt(params.Page, 10)
	}
	if params.PageSize != 0 {
		parsedParams["page_size"] = strconv.FormatInt(params.PageSize, 10)
	}
	if params.Timestamp != 0 {
		parsedParams["timestamp"] = strconv.FormatInt(params.Timestamp, 10)
	}

	resp := new(ListPoliciesResp)
	result := c.client.Get().
		SubResourcef("/api/v1/systems/%s/policies", c.config.SystemID).
		WithContext(ctx).
		WithHeaders(c.basicHeader).
		WithParams(parsedParams).
		Body(nil).Do()

	err := result.Into(resp)
	if err != nil {
		return nil, err
	}

	if resp.Code != 0 {
		return nil, &AuthError{
			RequestID: result.Header.Get(IamRequestHeader),
			Reason:    fmt.Errorf("get system info failed, code: %d, msg:%s", resp.Code, resp.Message),
		}
	}

	return resp.Data, nil
}
