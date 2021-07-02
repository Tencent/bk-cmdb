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
	"errors"
	"fmt"
	"strings"
	"sync"

	"configcenter/src/ac/meta"
	"configcenter/src/common/auth"
	cc "configcenter/src/common/backbone/configcenter"
	"configcenter/src/scene_server/auth_server/sdk/operator"
)

const (
	IamRequestHeader   = "X-Request-Id"
	iamAppCodeHeader   = "X-Bk-App-Code"
	iamAppSecretHeader = "X-Bk-App-Secret"

	SystemIDCMDB     = "bk_cmdb"
	SystemNameCMDBEn = "cmdb"
	SystemNameCMDB   = "配置平台"

	SystemIDIAM = "bk_iam"
)

type AuthConfig struct {
	// blueking's auth center addresses
	Address []string
	// app code is used for authorize used.
	AppCode string
	// app secret is used for authorized
	AppSecret string
	// the system id that cmdb used in auth center.
	// default value: bk_cmdb
	SystemID string
}

func ParseConfigFromKV(prefix string, configMap map[string]string) (AuthConfig, error) {
	var cfg AuthConfig
	if !auth.EnableAuthorize() {
		return AuthConfig{}, nil
	}
	address, err := cc.String(prefix + ".address")
	if err != nil {
		return cfg, errors.New(`missing "address" configuration for auth center`)
	}

	cfg.Address = strings.Split(strings.Replace(address, " ", "", -1), ",")
	if len(cfg.Address) == 0 {
		return cfg, errors.New(`invalid "address" configuration for auth center`)
	}
	for i := range cfg.Address {
		if !strings.HasSuffix(cfg.Address[i], "/") {
			cfg.Address[i] = cfg.Address[i] + "/"
		}
	}

	appSecret, err := cc.String(prefix + ".appSecret")
	if err != nil {
		return cfg, errors.New(`invalid "appSecret" configuration for auth center`)
	}
	cfg.AppSecret = appSecret
	if len(cfg.AppSecret) == 0 {
		return cfg, errors.New(`invalid "appSecret" configuration for auth center`)
	}

	appCode, err := cc.String(prefix + ".appCode")
	if err != nil {
		return cfg, errors.New(`missing "appCode" configuration for auth center`)
	}
	cfg.AppCode = appCode
	if len(cfg.AppCode) == 0 {
		return cfg, errors.New(`invalid "appCode" configuration for auth center`)
	}

	cfg.SystemID = SystemIDCMDB

	return cfg, nil
}

type System struct {
	ID                 string     `json:"id,omitempty"`
	Name               string     `json:"name,omitempty"`
	EnglishName        string     `json:"name_en,omitempty"`
	Description        string     `json:"description,omitempty"`
	EnglishDescription string     `json:"description_en,omitempty"`
	Clients            string     `json:"clients,omitempty"`
	ProviderConfig     *SysConfig `json:"provider_config"`
}

// SystemQueryField is system query field for searching system info
type SystemQueryField string

const (
	FieldBaseInfo               SystemQueryField = "base_info"
	FieldResourceTypes          SystemQueryField = "resource_types"
	FieldActions                SystemQueryField = "actions"
	FieldActionGroups           SystemQueryField = "action_groups"
	FieldInstanceSelections     SystemQueryField = "instance_selections"
	FieldResourceCreatorActions SystemQueryField = "resource_creator_actions"
	FieldCommonActions          SystemQueryField = "common_actions"
)

type SysConfig struct {
	Host string `json:"host,omitempty"`
	Auth string `json:"auth,omitempty"`
}

type SystemResp struct {
	BaseResponse
	Data RegisteredSystemInfo `json:"data"`
}

type RegisteredSystemInfo struct {
	BaseInfo               System                 `json:"base_info"`
	ResourceTypes          []ResourceType         `json:"resource_types"`
	Actions                []ResourceAction       `json:"actions"`
	ActionGroups           []ActionGroup          `json:"action_groups"`
	InstanceSelections     []InstanceSelection    `json:"instance_selections"`
	ResourceCreatorActions ResourceCreatorActions `json:"resource_creator_actions"`
	CommonActions          []CommonAction         `json:"common_actions"`
}

type BaseResponse struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

type AuthError struct {
	RequestID string
	Reason    error
}

func (a *AuthError) Error() string {
	if len(a.RequestID) == 0 {
		return a.Reason.Error()
	}
	return fmt.Sprintf("iam request id: %s, err: %s", a.RequestID, a.Reason.Error())
}

type TypeID string

const (
	SysModelGroup TypeID = "sys_model_group"

	// special model resource for selection of instance, not including models whose instances are managed separately
	SysInstanceModel         TypeID = "sys_instance_model"
	SysModel                 TypeID = "sys_model"
	SysAssociationType       TypeID = "sys_association_type"
	SysAuditLog              TypeID = "sys_audit_log"
	SysOperationStatistic    TypeID = "sys_operation_statistic"
	SysResourcePoolDirectory TypeID = "sys_resource_pool_directory"
	SysHostRscPoolDirectory  TypeID = "sys_host_rsc_pool_directory"
	SysCloudArea             TypeID = "sys_cloud_area"
	SysCloudAccount          TypeID = "sys_cloud_account"
	SysCloudResourceTask     TypeID = "sys_cloud_resource_task"
	SysEventWatch            TypeID = "event_watch"
	Host                     TypeID = "host"
	UserCustom               TypeID = "usercustom"

	// for resource type, which is not need to be authorized
	SkipType TypeID = "skip_type"
)

const (
	Business             TypeID = "biz"
	BusinessForHostTrans TypeID = "biz_for_host_trans"
	// Set                       ResourceTypeID = "set"
	// Module                    ResourceTypeID = "module"
	BizCustomQuery            TypeID = "biz_custom_query"
	BizTopology               TypeID = "biz_topology"
	BizCustomField            TypeID = "biz_custom_field"
	BizProcessServiceTemplate TypeID = "biz_process_service_template"
	BizProcessServiceCategory TypeID = "biz_process_service_category"
	BizProcessServiceInstance TypeID = "biz_process_service_instance"
	BizSetTemplate            TypeID = "biz_set_template"
	BizHostApply              TypeID = "biz_host_apply"
)

// describe resource type defined and registered to iam.
type ResourceType struct {
	// unique id
	ID TypeID `json:"id"`
	// unique name
	Name   string `json:"name"`
	NameEn string `json:"name_en"`
	// unique description
	Description    string         `json:"description"`
	DescriptionEn  string         `json:"description_en"`
	Parents        []Parent       `json:"parents"`
	ProviderConfig ResourceConfig `json:"provider_config"`
	Version        int64          `json:"version"`
}

type ResourceConfig struct {
	// the url to get this resource.
	Path string `json:"path"`
}

type Parent struct {
	// only one value for cmdb.
	// default value: bk_cmdb
	SystemID   string `json:"system_id"`
	ResourceID TypeID `json:"id"`
}

type ActionType string

const (
	Create ActionType = "create"
	Delete ActionType = "delete"
	View   ActionType = "view"
	Edit   ActionType = "edit"
	List   ActionType = "list"
)

var ActionTypeIDNameMap = map[ActionType]string{
	Create: "新建",
	Edit:   "编辑",
	Delete: "删除",
	View:   "查询",
}

type ActionID string

const (
	EditBusinessHost                   ActionID = "edit_biz_host"
	BusinessHostTransferToResourcePool ActionID = "unassign_biz_host"
	HostTransferAcrossBusiness         ActionID = "host_transfer_across_business"

	CreateBusinessCustomQuery ActionID = "create_biz_dynamic_query"
	EditBusinessCustomQuery   ActionID = "edit_biz_dynamic_query"
	DeleteBusinessCustomQuery ActionID = "delete_biz_dynamic_query"

	EditBusinessCustomField ActionID = "edit_biz_custom_field"

	CreateBusinessServiceCategory ActionID = "create_biz_service_category"
	EditBusinessServiceCategory   ActionID = "edit_biz_service_category"
	DeleteBusinessServiceCategory ActionID = "delete_biz_service_category"

	CreateBusinessServiceInstance ActionID = "create_biz_service_instance"
	EditBusinessServiceInstance   ActionID = "edit_biz_service_instance"
	DeleteBusinessServiceInstance ActionID = "delete_biz_service_instance"

	CreateBusinessServiceTemplate ActionID = "create_biz_service_template"
	EditBusinessServiceTemplate   ActionID = "edit_biz_service_template"
	DeleteBusinessServiceTemplate ActionID = "delete_biz_service_template"

	CreateBusinessSetTemplate ActionID = "create_biz_set_template"
	EditBusinessSetTemplate   ActionID = "edit_biz_set_template"
	DeleteBusinessSetTemplate ActionID = "delete_biz_set_template"

	CreateBusinessTopology ActionID = "create_biz_topology"
	EditBusinessTopology   ActionID = "edit_biz_topology"
	DeleteBusinessTopology ActionID = "delete_biz_topology"

	EditBusinessHostApply ActionID = "edit_biz_host_apply"

	CreateResourcePoolHost              ActionID = "create_resource_pool_host"
	EditResourcePoolHost                ActionID = "edit_resource_pool_host"
	DeleteResourcePoolHost              ActionID = "delete_resource_pool_host"
	ResourcePoolHostTransferToBusiness  ActionID = "assign_host_to_biz"
	ResourcePoolHostTransferToDirectory ActionID = "host_transfer_in_resource_pool"

	CreateResourcePoolDirectory ActionID = "create_resource_pool_directory"
	EditResourcePoolDirectory   ActionID = "edit_resource_pool_directory"
	DeleteResourcePoolDirectory ActionID = "delete_resource_pool_directory"

	CreateBusiness       ActionID = "create_business"
	EditBusiness         ActionID = "edit_business"
	ArchiveBusiness      ActionID = "archive_business"
	FindBusiness         ActionID = "find_business"
	ViewBusinessResource ActionID = "find_business_resource"

	CreateCloudArea ActionID = "create_cloud_area"
	EditCloudArea   ActionID = "edit_cloud_area"
	DeleteCloudArea ActionID = "delete_cloud_area"

	CreateCloudAccount ActionID = "create_cloud_account"
	EditCloudAccount   ActionID = "edit_cloud_account"
	DeleteCloudAccount ActionID = "delete_cloud_account"
	FindCloudAccount   ActionID = "find_cloud_account"

	CreateCloudResourceTask ActionID = "create_cloud_resource_task"
	EditCloudResourceTask   ActionID = "edit_cloud_resource_task"
	DeleteCloudResourceTask ActionID = "delete_cloud_resource_task"
	FindCloudResourceTask   ActionID = "find_cloud_resource_task"

	CreateSysModel ActionID = "create_sys_model"
	EditSysModel   ActionID = "edit_sys_model"
	DeleteSysModel ActionID = "delete_sys_model"

	CreateAssociationType ActionID = "create_association_type"
	EditAssociationType   ActionID = "edit_association_type"
	DeleteAssociationType ActionID = "delete_association_type"

	CreateModelGroup ActionID = "create_model_group"
	EditModelGroup   ActionID = "edit_model_group"
	DeleteModelGroup ActionID = "delete_model_group"

	EditBusinessLayer ActionID = "edit_business_layer"

	EditModelTopologyView ActionID = "edit_model_topology_view"

	FindOperationStatistic ActionID = "find_operation_statistic"
	EditOperationStatistic ActionID = "edit_operation_statistic"

	FindAuditLog ActionID = "find_audit_log"

	WatchHostEvent           ActionID = "watch_host_event"
	WatchHostRelationEvent   ActionID = "watch_host_relation_event"
	WatchBizEvent            ActionID = "watch_biz_event"
	WatchSetEvent            ActionID = "watch_set_event"
	WatchModuleEvent         ActionID = "watch_module_event"
	WatchProcessEvent        ActionID = "watch_process_event"
	WatchCommonInstanceEvent ActionID = "watch_object_instance_event"
	GlobalSettings           ActionID = "global_settings"

	// Unknown is an action that can not be recognized
	Unsupported ActionID = "unsupported"
	// Skip is an action that no need to auth
	Skip ActionID = "skip"
)

const (
	// IAM侧资源的通用模型实例前缀标识
	IAMSysInstTypePrefix = meta.CMDBSysInstTypePrefix
)

type ResourceAction struct {
	// must be a unique id in the whole system.
	ID ActionID `json:"id"`
	// must be a unique name in the whole system.
	Name                 string               `json:"name"`
	NameEn               string               `json:"name_en"`
	Type                 ActionType           `json:"type"`
	RelatedResourceTypes []RelateResourceType `json:"related_resource_types"`
	RelatedActions       []ActionID           `json:"related_actions"`
	Version              int                  `json:"version"`
}

// 选择类型, 资源在权限中心产品上配置权限时的作用范围
type SelectionMode string

const (
	// 仅可选择实例, 默认值
	modeInstance SelectionMode = "instance"
	// 仅可配置属性, 此时instance_selections配置不生效
	modeAttribute SelectionMode = "attribute"
	// 可以同时选择实例和配置属性
	modeAll SelectionMode = "all"
)

type RelateResourceType struct {
	SystemID           string                     `json:"system_id"`
	ID                 TypeID                     `json:"id"`
	NameAlias          string                     `json:"name_alias"`
	NameAliasEn        string                     `json:"name_alias_en"`
	Scope              *Scope                     `json:"scope"`
	SelectionMode      SelectionMode              `json:"selection_mode"`
	InstanceSelections []RelatedInstanceSelection `json:"related_instance_selections"`
}

type Scope struct {
	Op      string         `json:"op"`
	Content []ScopeContent `json:"content"`
}

type ScopeContent struct {
	Op    string `json:"op"`
	Field string `json:"field"`
	Value string `json:"value"`
}

type RelatedInstanceSelection struct {
	ID       InstanceSelectionID `json:"id"`
	SystemID string              `json:"system_id"`
	// if true, then this selected instance with not be calculated to calculate the auth.
	// as is will be ignored, the only usage for this selection is to support a convenient
	// way for user to find it's resource instances.
	IgnoreAuthPath bool `json:"ignore_iam_path"`
}

// groups related resource actions to make action selection more organized
type ActionGroup struct {
	// must be a unique name in the whole system.
	Name      string         `json:"name"`
	NameEn    string         `json:"name_en"`
	SubGroups []ActionGroup  `json:"sub_groups,omitempty"`
	Actions   []ActionWithID `json:"actions,omitempty"`
}

type ActionWithID struct {
	ID ActionID `json:"id"`
}

type InstanceSelectionID string

const (
	// 业务的两种视图，管理的资源也相同，仅名称做区分
	BusinessSelection                  InstanceSelectionID = "business"
	BusinessHostTransferSelection      InstanceSelectionID = "business_host_transfer"
	BizHostInstanceSelection           InstanceSelectionID = "biz_host_instance"
	BizCustomQuerySelection            InstanceSelectionID = "biz_custom_query"
	BizProcessServiceTemplateSelection InstanceSelectionID = "biz_process_service_template"
	BizSetTemplateSelection            InstanceSelectionID = "biz_set_template"
	SysHostInstanceSelection           InstanceSelectionID = "sys_host_instance"
	SysModelGroupSelection             InstanceSelectionID = "sys_model_group"
	SysModelSelection                  InstanceSelectionID = "sys_model"
	SysInstanceModelSelection          InstanceSelectionID = "sys_instance_model"
	SysAssociationTypeSelection        InstanceSelectionID = "sys_association_type"
	SysCloudAreaSelection              InstanceSelectionID = "sys_cloud_area"
	SysCloudAccountSelection           InstanceSelectionID = "sys_cloud_account"
	SysCloudResourceTaskSelection      InstanceSelectionID = "sys_cloud_resource_task"
	// 主机池目录的两种视图，管理的资源也相同，仅名称做区分
	SysResourcePoolDirectorySelection InstanceSelectionID = "sys_resource_pool_directory"
	SysHostRscPoolDirectorySelection  InstanceSelectionID = "sys_host_rsc_pool_directory"
)

type InstanceSelection struct {
	// unique
	ID InstanceSelectionID `json:"id"`
	// unique
	Name string `json:"name"`
	// unique
	NameEn            string          `json:"name_en"`
	ResourceTypeChain []ResourceChain `json:"resource_type_chain"`
}

type ResourceChain struct {
	SystemID string `json:"system_id"`
	ID       TypeID `json:"id"`
}

type iamDiscovery struct {
	servers []string
	index   int
	sync.Mutex
}

func (s *iamDiscovery) GetServers() ([]string, error) {
	s.Lock()
	defer s.Unlock()

	num := len(s.servers)
	if num == 0 {
		return []string{}, errors.New("oops, there is no server can be used")
	}

	if s.index < num-1 {
		s.index = s.index + 1
		return append(s.servers[s.index-1:], s.servers[:s.index-1]...), nil
	} else {
		s.index = 0
		return append(s.servers[num-1:], s.servers[:num-1]...), nil
	}
}

func (s *iamDiscovery) GetServersChan() chan []string {
	return nil
}

// resource type with id, used to represent resource layer from root to leaf
type RscTypeAndID struct {
	ResourceType TypeID `json:"resource_type"`
	ResourceID   string `json:"resource_id,omitempty"`
}

// iam resource, system is resource's iam system id, type is resource type, resource id and attribute are used for filtering
type Resource struct {
	System    string                 `json:"system"`
	Type      TypeID                 `json:"type"`
	ID        string                 `json:"id,omitempty"`
	Attribute map[string]interface{} `json:"attribute,omitempty"`
}

// specifies resource creation actions' related actions that resource creator will have permissions to
type ResourceCreatorActions struct {
	Config []ResourceCreatorAction `json:"config"`
}

type ResourceCreatorAction struct {
	ResourceID       TypeID                  `json:"id"`
	Actions          []CreatorRelatedAction  `json:"actions"`
	SubResourceTypes []ResourceCreatorAction `json:"sub_resource_types,omitempty"`
}

type CreatorRelatedAction struct {
	ID         ActionID `json:"id"`
	IsRequired bool     `json:"required"`
}

// CommonAction specifies a common operation's related iam actions
type CommonAction struct {
	Name        string         `json:"name"`
	EnglishName string         `json:"name_en"`
	Actions     []ActionWithID `json:"actions"`
}

// DynamicAction is dynamic model action
type DynamicAction struct {
	ActionID     ActionID
	ActionType   ActionType
	ActionNameCN string
	ActionNameEN string
}

type DeleteCMDBResourceParam struct {
	ActionIDs            []ActionID
	InstanceSelectionIDs []InstanceSelectionID
	TypeIDs              []TypeID
}

type ListPoliciesParams struct {
	ActionID  ActionID
	Page      int64
	PageSize  int64
	Timestamp int64
}

type ListPoliciesResp struct {
	BaseResponse
	Data *ListPoliciesData `json:"data"`
}

type ListPoliciesData struct {
	Metadata PolicyMetadata `json:"metadata"`
	Count    int64          `json:"count"`
	Results  []PolicyResult `json:"results"`
}

type PolicyMetadata struct {
	System    string       `json:"system"`
	Action    ActionWithID `json:"action"`
	Timestamp int64        `json:"timestamp"`
}

type PolicyResult struct {
	Version    string           `json:"version"`
	ID         int64            `json:"id"`
	Subject    PolicySubject    `json:"subject"`
	Expression *operator.Policy `json:"expression"`
	ExpiredAt  int64            `json:"expired_at"`
}

type PolicySubject struct {
	Type string `json:"type"`
	ID   string `json:"id"`
	Name string `json:"name"`
}

type SimplifiedInstance struct {
	InstanceID   int64  `json:"bk_inst_id" bson:"bk_inst_id"`
	InstanceName string `json:"bk_inst_name" bson:"bk_inst_name"`
}
