/*
 * Tencent is pleased to support the open source community by making 蓝鲸 available.,
 * Copyright (C) 2017,-2018 THL A29 Limited, a Tencent company. All rights reserved.
 * Licensed under the MIT License (the ",License",); you may not use this file except
 * in compliance with the License. You may obtain a copy of the License at
 * http://opensource.org/licenses/MIT
 * Unless required by applicable law or agreed to in writing, software distributed under
 * the License is distributed on an ",AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
 * either express or implied. See the License for the specific language governing permissions and
 * limitations under the License.
 */

package metadata

import (
	"fmt"

	"configcenter/src/common"
	"configcenter/src/common/errors"
	"configcenter/src/common/mapstr"
)

// BaseResp common result struct
type BaseResp struct {
	Result      bool           `json:"result" mapstructure:"result"`
	Code        int            `json:"bk_error_code" mapstructure:"bk_error_code"`
	ErrMsg      string         `json:"bk_error_msg" mapstructure:"bk_error_msg"`
	Permissions *IamPermission `json:"permission" mapstructure:"permission"`
}

// CCError 根据response返回的信息产生错误
func (br *BaseResp) CCError() errors.CCErrorCoder {
	if br.Result {
		return nil
	}
	return errors.New(br.Code, br.ErrMsg)
}

// New 根据response返回的信息产生错误
func (br *BaseResp) Error() error {
	if br.Result {
		return nil
	}
	return errors.New(br.Code, br.ErrMsg)
}

func (br *BaseResp) ToString() string {
	return fmt.Sprintf("code:%d, message:%s", br.Code, br.ErrMsg)
}

// JsonStringResp defines a response that do not parse the data field to a struct
// but decode it to a json string if possible
type JsonStringResp struct {
	BaseResp
	Data string
}

// JsonCntInfoResp defines a response that do not parse the data's count and info fields
// to a struct but decode it to a json string if possible
type JsonCntInfoResp struct {
	BaseResp
	Data CntInfoString `json:"data"`
}

type CntInfoString struct {
	Count int64 `json:"count"`
	// info is a json array string field.
	Info string `json:"info"`
}

type IamPermission struct {
	SystemID   string      `json:"system_id"`
	SystemName string      `json:"system_name"`
	Actions    []IamAction `json:"actions"`
}

type IamAction struct {
	ID                   string            `json:"id"`
	Name                 string            `json:"name"`
	RelatedResourceTypes []IamResourceType `json:"related_resource_types"`
}

type IamResourceType struct {
	SystemID   string                  `json:"system_id"`
	SystemName string                  `json:"system_name"`
	Type       string                  `json:"type"`
	TypeName   string                  `json:"type_name"`
	Instances  [][]IamResourceInstance `json:"instances,omitempty"`
	Attributes []IamResourceAttribute  `json:"attributes,omitempty"`
}

type IamResourceInstance struct {
	Type     string `json:"type"`
	TypeName string `json:"type_name"`
	ID       string `json:"id"`
	Name     string `json:"name"`
}

type IamResourceAttribute struct {
	ID     string                      `json:"id"`
	Values []IamResourceAttributeValue `json:"values"`
}

type IamResourceAttributeValue struct {
	ID string `json:"id"`
}

type IamInstanceWithCreator struct {
	System    string                `json:"system"`
	Type      string                `json:"type"`
	ID        string                `json:"id"`
	Name      string                `json:"name"`
	Creator   string                `json:"creator"`
	Ancestors []IamInstanceAncestor `json:"ancestors,omitempty"`
}

type IamInstances struct {
	System    string        `json:"system"`
	Type      string        `json:"type"`
	Instances []IamInstance `json:"instances"`
}

type IamInstancesWithCreator struct {
	IamInstances `json:",inline"`
	Creator      string `json:"creator"`
}

type IamInstance struct {
	ID        string                `json:"id"`
	Name      string                `json:"name"`
	Ancestors []IamInstanceAncestor `json:"ancestors,omitempty"`
}

type IamInstanceAncestor struct {
	System string `json:"system"`
	Type   string `json:"type"`
	ID     string `json:"id"`
}

type IamCreatorActionPolicy struct {
	Action   ActionWithID `json:"action"`
	PolicyID int64        `json:"policy_id"`
}

type ActionWithID struct {
	ID string `json:"id"`
}

type IamBatchOperateInstanceAuthReq struct {
	Asynchronous bool             `json:"asynchronous"`
	Operate      IamAuthOperation `json:"operate"`
	System       string           `json:"system"`
	Actions      []ActionWithID   `json:"actions"`
	Subject      IamSubject       `json:"subject"`
	Resources    []IamInstances   `json:"resources"`
	ExpiredAt    int64            `json:"expired_at"`
}

type IamAuthOperation string

const (
	IamGrantOperation  = "grant"
	IamRevokeOperation = "revoke"
)

type IamSubject struct {
	Type string `json:"type"`
	Id   string `json:"id"`
}

type IamBatchOperateInstanceAuthRes struct {
	Action   ActionWithID `json:"action"`
	PolicyID int64        `json:"policy_id"`
}

// Permission  describes all the authorities that a user
// is need, when he attempts to operate a resource.
// Permission is used only when a user do not have the authority to
// access a resources with a action.
type Permission struct {
	SystemID      string `json:"system_id"`
	SystemName    string `json:"system_name"`
	ScopeType     string `json:"scope_type"`
	ScopeTypeName string `json:"scope_type_name"`
	ScopeID       string `json:"scope_id"`
	ScopeName     string `json:"scope_name"`
	ActionID      string `json:"action_id"`
	ActionName    string `json:"action_name"`
	// newly added two field.
	ResourceTypeName string `json:"resource_type_name"`
	ResourceType     string `json:"resource_type"`

	Resources [][]Resource `json:"resources"`
}

type Resource struct {
	ResourceTypeName string `json:"resource_type_name"`
	ResourceType     string `json:"resource_type"`
	ResourceName     string `json:"resource_name"`
	ResourceID       string `json:"resource_id"`
}

func NewNoPermissionResp(permission *IamPermission) BaseResp {
	return BaseResp{
		Result:      false,
		Code:        common.CCNoPermission,
		ErrMsg:      "no permissions",
		Permissions: permission,
	}
}

// SuccessBaseResp default result
var SuccessBaseResp = BaseResp{Result: true, Code: common.CCSuccess, ErrMsg: common.CCSuccessStr}

type SuccessResponse struct {
	BaseResp `json:",inline"`
	Data     interface{} `json:"data"`
}

func NewSuccessResponse(data interface{}) *SuccessResponse {
	return &SuccessResponse{
		BaseResp: SuccessBaseResp,
		Data:     data,
	}
}

// CreatedCount created count struct
type CreatedCount struct {
	Count uint64 `json:"created_count"`
}

// UpdatedCount created count struct
type UpdatedCount struct {
	Count uint64 `json:"updated_count"`
}

// UpdateAttributeIndex created bk_property_index info struct
type UpdateAttributeIndex struct {
	Id    int64 `json:"id"`
	Index int64 `json:"index"`
}

// DeletedCount created count struct
type DeletedCount struct {
	Count uint64 `json:"deleted_count"`
}

// ExceptionResult exception info
type ExceptionResult struct {
	Message     string      `json:"message"`
	Code        int64       `json:"code"`
	Data        interface{} `json:"data"`
	OriginIndex int64       `json:"origin_index"`
}

// CreatedDataResult common created result definition
type CreatedDataResult struct {
	OriginIndex int64  `json:"origin_index"`
	ID          uint64 `json:"id"`
}

// RepeatedDataResult repeated data
type RepeatedDataResult struct {
	OriginIndex int64         `json:"origin_index"`
	Data        mapstr.MapStr `json:"data"`
}

// UpdatedDataResult common update operation result
type UpdatedDataResult struct {
	OriginIndex int64  `json:"origin_index"`
	ID          uint64 `json:"id"`
}

// SetDataResult common set result definition
type SetDataResult struct {
	UpdatedCount `json:",inline"`
	CreatedCount `json:",inline"`
	Created      []CreatedDataResult `json:"created"`
	Updated      []UpdatedDataResult `json:"updated"`
	Exceptions   []ExceptionResult   `json:"exception"`
}

// CreateManyInfoResult create many function return this result struct
type CreateManyInfoResult struct {
	Created    []CreatedDataResult  `json:"created"`
	Repeated   []RepeatedDataResult `json:"repeated"`
	Exceptions []ExceptionResult    `json:"exception"`
}

// CreateManyDataResult the data struct definition in create many function result
type CreateManyDataResult struct {
	CreateManyInfoResult `json:",inline"`
}

// CreateOneDataResult the data struct definition in create one function result
type CreateOneDataResult struct {
	Created CreatedDataResult `json:"created"`
}

// SearchResp common search response
type SearchResp struct {
	BaseResp `json:",inline"`
	Data     SearchDataResult `json:"data"`
}

// SearchDataResult common search data result
type SearchDataResult struct {
	Count int64           `json:"count"`
	Info  []mapstr.MapStr `json:"info"`
}

// QueryModelDataResult used to define the model query
type QueryModelDataResult struct {
	Count int64    `json:"count"`
	Info  []Object `json:"info"`
}

// QueryModelWithAttributeDataResult used to define the model with attribute query
type QueryModelWithAttributeDataResult struct {
	Count int64             `json:"count"`
	Info  []SearchModelInfo `json:"info"`
}

// QueryModelAttributeDataResult search model attr data result
type QueryModelAttributeDataResult struct {
	Count int64       `json:"count"`
	Info  []Attribute `json:"info"`
}

// QueryModelAttributeGroupDataResult query model attribute group result definition
type QueryModelAttributeGroupDataResult struct {
	Count int64   `json:"count"`
	Info  []Group `json:"info"`
}

// QueryModelClassificationDataResult query model classification result definition
type QueryModelClassificationDataResult struct {
	Count int64            `json:"count"`
	Info  []Classification `json:"info"`
}

// SearchAssociationKindResult search association kind result definition
type SearchAssociationKindResult struct {
	Count uint64            `json:"count"`
	Info  []AssociationKind `json:"info"`
}

// ReadModelAttrResult  read model attribute api http response return result struct
type ReadModelAttrResult struct {
	BaseResp `json:",inline"`
	Data     QueryModelAttributeDataResult `json:"data"`
}

// ReadModelClassifitionResult  read model classifition api http response return result struct
type ReadModelClassifitionResult struct {
	BaseResp `json:",inline"`
	Data     QueryModelClassificationDataResult `json:"data"`
}

// ReadModelResult  read model classifition api http response return result struct
type ReadModelResult struct {
	BaseResp `json:",inline"`
	Data     QueryModelWithAttributeDataResult `json:"data"`
}

type ReadModelAttributeGroupResult struct {
	BaseResp `json:",inline"`
	Data     QueryModelAttributeGroupDataResult `json:"data"`
}

type ReadModelUniqueResult struct {
	BaseResp `json:",inline"`
	Data     QueryUniqueResult `json:"data"`
}

type ReadModelAssociationResult struct {
	BaseResp
	Data struct {
		Count uint64        `json:"count"`
		Info  []Association `json:"info"`
	}
}

type ReadInstAssociationResult struct {
	BaseResp
	Data struct {
		Count uint64     `json:"count"`
		Info  []InstAsst `json:"info"`
	}
}

// OperaterException  result
type OperaterException struct {
	BaseResp `json:",inline"`
	Data     []ExceptionResult `json:"data"`
}

type Uint64DataResponse struct {
	BaseResp `json:",inline"`
	Data     uint64 `json:"data"`
}

type TransferException struct {
	HostID []int64 `json:"bk_host_id"`
	ErrMsg string  `json:"bk_error_msg"`
}

type TransferExceptionResult struct {
	BaseResp `json:",inline"`
	Data     TransferException `json:"data"`
}
