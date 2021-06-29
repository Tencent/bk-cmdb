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
	"errors"
	"fmt"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"configcenter/src/common"
	cErr "configcenter/src/common/errors"
	"configcenter/src/common/mapstr"
	"configcenter/src/common/querybuilder"
	"configcenter/src/common/selector"
	"configcenter/src/common/util"
)

type DeleteCategoryInput struct {
	ID int64 `json:"id"`
}

type CreateProcessTemplateBatchInput struct {
	BizID             int64           `json:"bk_biz_id"`
	ServiceTemplateID int64           `json:"service_template_id"`
	Processes         []ProcessDetail `json:"processes"`
}

type DeleteProcessTemplateBatchInput struct {
	BizID            int64   `json:"bk_biz_id"`
	ProcessTemplates []int64 `json:"process_templates"`
}

type ProcessDetail struct {
	Spec *ProcessProperty `json:"spec"`
}

type ListServiceTemplateInput struct {
	BizID int64 `json:"bk_biz_id"`
	// this field can be empty, it a optional condition.
	ServiceCategoryID int64    `json:"service_category_id"`
	Page              BasePage `json:"page"`
	// search service templates by name
	Search string `json:"search"`
	// used with search, means whether search service templates with exact name or not
	IsExact bool `json:"is_exact"`
}

type DeleteServiceTemplatesInput struct {
	BizID             int64 `json:"bk_biz_id"`
	ServiceTemplateID int64 `json:"service_template_id"`
}

type GetServiceTemplateSyncStatusOption struct {
	ServiceTemplateIDs []int64 `json:"service_template_ids"`
	ModuleIDs          []int64 `json:"bk_module_ids"`
	// IsPartial if is set, check service templates status, returns need sync for template if one module need sync
	IsPartial bool `json:"is_partial"`
}

type GetServiceTemplateSyncStatusResult struct {
	BaseResp `json:",inline"`
	Data     *ServiceTemplateSyncStatus `json:"data"`
}

type ServiceTemplateSyncStatus struct {
	ServiceTemplates []SvcTempSyncStatus `json:"service_templates"`
	Modules          []ModuleSyncStatus  `json:"modules"`
}

type SvcTempSyncStatus struct {
	ServiceTemplateID int64 `json:"service_template_id"`
	NeedSync          bool  `json:"need_sync"`
}

type ModuleSyncStatus struct {
	ModuleID int64 `json:"bk_module_id"`
	NeedSync bool  `json:"need_sync"`
}

type CreateServiceInstanceForServiceTemplateInput struct {
	BizID                      int64                         `json:"bk_biz_id"`
	Name                       string                        `json:"name"`
	ModuleID                   int64                         `json:"bk_module_id"`
	Instances                  []CreateServiceInstanceDetail `json:"instances"`
	HostApplyConflictResolvers []HostApplyConflictResolver   `json:"host_apply_conflict_resolvers"`
}

type CreateServiceInstancePreviewInput struct {
	BizID    int64   `json:"bk_biz_id"`
	ModuleID int64   `json:"bk_module_id"`
	HostIDs  []int64 `json:"bk_host_ids"`
}

type CreateRawProcessInstanceInput struct {
	BizID             int64                   `json:"bk_biz_id"`
	ServiceInstanceID int64                   `json:"service_instance_Id"`
	Processes         []ProcessInstanceDetail `json:"processes"`
}

type UpdateRawProcessInstanceInput struct {
	BizID     int64                    `json:"bk_biz_id"`
	Processes []Process                `json:"-"`
	Raw       []map[string]interface{} `json:"processes"`
}

type DeleteProcessInstanceInServiceInstanceInput struct {
	BizID              int64   `json:"bk_biz_id"`
	ProcessInstanceIDs []int64 `json:"process_instance_ids"`
}

type GetServiceInstanceInModuleInput struct {
	BizID     int64              `json:"bk_biz_id"`
	ModuleID  int64              `json:"bk_module_id"`
	HostIDs   []int64            `json:"bk_host_ids"`
	Page      BasePage           `json:"page"`
	SearchKey *string            `json:"search_key"`
	Selectors selector.Selectors `json:"selectors"`
}

type GetServiceInstanceBySetTemplateInput struct {
	SetTemplateID int64    `json:"set_template_id"`
	Page          BasePage `json:"page"`
}

type DiffModuleWithTemplateOption struct {
	BizID     int64   `json:"bk_biz_id"`
	ModuleIDs []int64 `json:"bk_module_ids"`
	// PartialCompare judge whether need compare partial and finish in advance
	// it finish the compare in advance once one module has difference with service template
	PartialCompare bool `json:"partial_compare"`
}

type DiffOneModuleWithTemplateOption struct {
	BizID    int64 `json:"bk_biz_id"`
	ModuleID int64 `json:"bk_module_id"`
}

type UpdateServiceInstanceOption struct {
	Data []OneUpdatedSrvInst `json:"data"`
}

type OneUpdatedSrvInst struct {
	ServiceInstanceID int64                  `json:"service_instance_id"`
	Update            map[string]interface{} `json:"update"`
}

func (o *UpdateServiceInstanceOption) Validate() (rawError cErr.RawErrorInfo) {
	if len(o.Data) == 0 {
		return cErr.RawErrorInfo{
			ErrCode: common.CCErrCommParamsInvalid,
			Args:    []interface{}{"data"},
		}
	}

	for _, inst := range o.Data {
		if inst.ServiceInstanceID <= 0 {
			return cErr.RawErrorInfo{
				ErrCode: common.CCErrCommParamsInvalid,
				Args:    []interface{}{"data.service_instance_id must bigger than 0"},
			}
		}

		// so far, only allow to update service instance name
		if len(inst.Update) == 0 || len(inst.Update) > 1 {
			return cErr.RawErrorInfo{
				ErrCode: common.CCErrCommParamsInvalid,
				Args:    []interface{}{"can only update service instance name"},
			}
		}

		instName, ok := inst.Update[common.BKFieldName].(string)
		if !ok {
			return cErr.RawErrorInfo{
				ErrCode: common.CCErrCommParamsInvalid,
				Args:    []interface{}{"can only update service instance name"},
			}
		}
		if len(instName) == 0 {
			return cErr.RawErrorInfo{
				ErrCode: common.CCErrCommParamsInvalid,
				Args:    []interface{}{"service instance name can't be empty"},
			}
		}
		if len(instName) > common.NameFieldMaxLength {
			return cErr.RawErrorInfo{
				ErrCode: common.CCErrCommParamsInvalid,
				Args:    []interface{}{"service instance name is too long"},
			}
		}
	}

	return cErr.RawErrorInfo{}
}

type DeleteServiceInstanceOption struct {
	BizID              int64   `json:"bk_biz_id"`
	ServiceInstanceIDs []int64 `json:"service_instance_ids" field:"service_instance_ids" bson:"service_instance_ids"`
}

type CoreDeleteServiceInstanceOption struct {
	BizID              int64   `json:"bk_biz_id"`
	ServiceInstanceIDs []int64 `json:"service_instance_ids" field:"service_instance_ids" bson:"service_instance_ids"`
}

type FindServiceAndProcessInstanceOption struct {
	BizID             int64 `json:"bk_biz_id" field:"bk_biz_id" bson:"bk_biz_id"`
	ModuleID          int64 `json:"bk_module_id" field:"bk_module_id" bson:"bk_module_id"`
	ServiceTemplateID int64 `json:"service_template_id" field:"service_template_id" bson:"service_template_id"`
}

// to describe the differences between service instance and it's service template's
// process template's attribute.
type ServiceProcessInstanceDifference struct {
	ServiceInstanceID   int64             `json:"service_instance_id" field:"service_instance_id" bson:"service_instance_id"`
	ServiceInstanceName string            `json:"service_instance_name" field:"service_instance_name" bson:"service_instance_name"`
	BizID               int64             `json:"bk_biz_id" field:"bk_biz_id" bson:"bk_biz_id"`
	HostID              int64             `json:"bk_host_id" field:"bk_host_id" bson:"bk_host_id"`
	Differences         *DifferenceDetail `json:"differences" field:"differences" bson:"differences"`
}

type DifferenceDetail struct {
	Unchanged []ProcessDifferenceDetail `json:"unchanged"`
	Changed   []ProcessDifferenceDetail `json:"changed"`
	Added     []ProcessDifferenceDetail `json:"added"`
	Removed   []ProcessDifferenceDetail `json:"removed"`
}

type ProcessDifferenceDetail struct {
	ProcessTemplateID int64                     `json:"process_template_id"`
	ProcessInstance   Process                   `json:"process_instance"`
	ChangedAttributes []ProcessChangedAttribute `json:"changed_attributes"`
}

type ProcessChangedAttribute struct {
	ID                    int64       `json:"id"`
	PropertyID            string      `json:"property_id"`
	PropertyName          string      `json:"property_name"`
	PropertyValue         interface{} `json:"property_value"`
	TemplatePropertyValue interface{} `json:"template_property_value"`
}

// ModuleDiffWithTemplateDetail 模块与服务模板间的差异
type ModuleDiffWithTemplateDetail struct {
	ModuleID          int64                       `json:"bk_module_id"`
	Unchanged         []ServiceInstanceDifference `json:"unchanged"`
	Changed           []ServiceInstanceDifference `json:"changed"`
	Added             []ServiceInstanceDifference `json:"added"`
	Removed           []ServiceInstanceDifference `json:"removed"`
	ChangedAttributes []ModuleChangedAttribute    `json:"changed_attributes"`
	HasDifference     bool                        `json:"has_difference"`
}

type ModuleChangedAttribute struct {
	ID                    int64       `json:"id"`
	PropertyID            string      `json:"property_id"`
	PropertyName          string      `json:"property_name"`
	PropertyValue         interface{} `json:"property_value"`
	TemplatePropertyValue interface{} `json:"template_property_value"`
}

// ServiceInstanceDifference 服务实例内的进程信息与进程模板ID不一致的服务实例列表
type ServiceInstanceDifference struct {
	ProcessTemplateID    int64                      `json:"process_template_id"`
	ProcessTemplateName  string                     `json:"process_template_name"`
	ServiceInstanceCount int                        `json:"service_instance_count"`
	ServiceInstances     []ServiceDifferenceDetails `json:"service_instances"`
}

// ServiceDifferenceDetails 服务实例与模板差异信息
type ServiceDifferenceDetails struct {
	ServiceInstance   SrvInstBriefInfo          `json:"service_instance"`
	Process           *Process                  `json:"process"`
	ChangedAttributes []ProcessChangedAttribute `json:"changed_attributes"`
}

type SrvInstBriefInfo struct {
	ID        int64  `field:"id" json:"id"`
	Name      string `field:"name" json:"name"`
	SvcTempID int64  `field:"service_template_id" json:"service_template_id"`
}

type CreateServiceInstanceOption struct {
	ModuleID int64 `json:"bk_module_id"`
	HostID   int64 `json:"bk_host_id"`
	// Processes parameter usable only when create instance with raw
	Processes []ProcessCreateOrUpdateInfo `json:"processes"`
}

type ProcessCreateOrUpdateInfo struct {
	// ProcessTemplateID indicate which process to update if service instance bound with a template
	ProcessTemplateID int64                  `json:"process_template_id"`
	ProcessInfo       map[string]interface{} `json:"process_info"`
}

type CreateServiceInstanceDetail struct {
	HostID              int64  `json:"bk_host_id"`
	ServiceInstanceName string `json:"service_instance_name"`
	// Processes parameter usable only when create instance with raw
	Processes []ProcessInstanceDetail `json:"processes"`
}

type ProcessInstanceDetail struct {
	// ProcessTemplateID indicate which process to update if service instance bound with a template
	ProcessTemplateID int64                  `json:"process_template_id"`
	ProcessData       map[string]interface{} `json:"process_info"`
}

type ListProcessTemplateWithServiceTemplateInput struct {
	BizID               int64    `json:"bk_biz_id"`
	ProcessTemplatesIDs []int64  `json:"process_template_ids"`
	ServiceTemplateID   int64    `json:"service_template_id"`
	Page                BasePage `json:"page" field:"page" bson:"page"`
}

// Validate validates the input param
func (o *ListProcessTemplateWithServiceTemplateInput) Validate() (rawError cErr.RawErrorInfo) {

	if o.BizID <= 0 {
		return cErr.RawErrorInfo{
			ErrCode: common.CCErrCommParamsInvalid,
			Args:    []interface{}{"bk_biz_id"},
		}
	}

	pageLimit := 200
	if len(o.ProcessTemplatesIDs) > pageLimit {
		return cErr.RawErrorInfo{
			ErrCode: common.CCErrArrayLengthWrong,
			Args:    []interface{}{"bk_process_ids", pageLimit},
		}
	}

	if o.ServiceTemplateID == 0 && len(o.ProcessTemplatesIDs) == 0 {
		return cErr.RawErrorInfo{
			ErrCode: common.CCErrCommParamsInvalid,
			Args:    []interface{}{"service_template_id and process_template_ids can't be empty at the same time"},
		}
	}

	return cErr.RawErrorInfo{}
}

type UpdateProcessByIDsInput struct {
	BizID      int64                  `json:"bk_biz_id"`
	ProcessIDs []int64                `json:"process_ids"`
	UpdateData map[string]interface{} `json:"update_data"`
}

// Validate validates the input param
func (o *UpdateProcessByIDsInput) Validate() (rawError cErr.RawErrorInfo) {
	if o.BizID <= 0 {
		return cErr.RawErrorInfo{
			ErrCode: common.CCErrCommParamsInvalid,
			Args:    []interface{}{"bk_biz_id"},
		}
	}

	if len(o.ProcessIDs) == 0 {
		return cErr.RawErrorInfo{
			ErrCode: common.CCErrCommParamsInvalid,
			Args:    []interface{}{"process_ids"},
		}
	}

	if len(o.ProcessIDs) > common.BKMaxPageSize {
		return cErr.RawErrorInfo{
			ErrCode: common.CCErrExceedMaxOperationRecordsAtOnce,
			Args:    []interface{}{common.BKMaxPageSize},
		}
	}

	if len(o.UpdateData) == 0 {
		return cErr.RawErrorInfo{
			ErrCode: common.CCErrCommParamsInvalid,
			Args:    []interface{}{"update_data"},
		}
	}

	if _, ok := o.UpdateData[common.BKProcessIDField]; ok {
		return cErr.RawErrorInfo{
			ErrCode: common.CCErrCommParamsInvalid,
			Args:    []interface{}{"update_data.bk_process_id"},
		}
	}

	return cErr.RawErrorInfo{}
}

type SyncServiceInstanceByTemplateOption struct {
	BizID     int64   `json:"bk_biz_id"`
	ModuleIDs []int64 `json:"bk_module_ids"`
}

// 用于同步单个模块的服务实例
type SyncModuleServiceInstanceByTemplateOption struct {
	BizID    int64 `json:"bk_biz_id"`
	ModuleID int64 `json:"bk_module_id"`
}

type ListServiceInstancesWithHostInput struct {
	BizID     int64              `json:"bk_biz_id"`
	HostID    int64              `json:"bk_host_id"`
	SearchKey *string            `json:"search_key"`
	Selectors selector.Selectors `json:"selectors"`
	Page      BasePage           `json:"page"`
}

type ListProcessInstancesOption struct {
	BizID             int64 `json:"bk_biz_id"`
	ServiceInstanceID int64 `json:"service_instance_id"`
}

type ListProcessInstancesNameIDsOption struct {
	BizID       int64    `json:"bk_biz_id"`
	ModuleID    int64    `json:"bk_module_id"`
	ProcessName string   `json:"process_name"`
	Page        BasePage `json:"page"`
}

// Validate validates the input param
func (o *ListProcessInstancesNameIDsOption) Validate() (rawError cErr.RawErrorInfo) {
	if o.BizID <= 0 {
		return cErr.RawErrorInfo{
			ErrCode: common.CCErrCommParamsInvalid,
			Args:    []interface{}{"bk_biz_id"},
		}
	}

	if o.ModuleID <= 0 {
		return cErr.RawErrorInfo{
			ErrCode: common.CCErrCommParamsInvalid,
			Args:    []interface{}{"bk_module_id"},
		}
	}

	if o.Page.IsIllegal() {
		return cErr.RawErrorInfo{
			ErrCode: common.CCErrCommParamsInvalid,
			Args:    []interface{}{"page.limit"},
		}
	}

	return cErr.RawErrorInfo{}
}

// ListProcessRelatedInfoOption is the input param for api ListProcessRelatedInfo
type ListProcessRelatedInfoOption struct {
	Set             SetCondOfP             `json:"set"`
	Module          ModuleCondOfP          `json:"module"`
	ServiceInstance ServiceInstanceCondOfP `json:"service_instance"`
	// ProcessPropertyFilter used to filter process instances by process properties
	ProcessPropertyFilter *querybuilder.QueryFilter `json:"process_property_filter"`
	Fields                []string                  `json:"fields"`
	Page                  BasePage                  `json:"page"`
}

type SetCondOfP struct {
	SetIDs []int64 `json:"bk_set_ids"`
}

type ModuleCondOfP struct {
	ModuleIDs []int64 `json:"bk_module_ids"`
}

type ServiceInstanceCondOfP struct {
	IDs []int64 `json:"ids"`
}

// Validate validates the input param
func (o *ListProcessRelatedInfoOption) Validate() (rawError cErr.RawErrorInfo) {
	if o.ProcessPropertyFilter != nil {
		if key, err := o.ProcessPropertyFilter.Validate(); err != nil {
			return cErr.RawErrorInfo{
				ErrCode: common.CCErrCommParamsInvalid,
				Args:    []interface{}{fmt.Sprintf("%s, host_property_filter.%s", err.Error(), key)},
			}
		}
		if o.ProcessPropertyFilter.GetDeep() > querybuilder.MaxDeep {
			return cErr.RawErrorInfo{
				ErrCode: common.CCErrCommParamsInvalid,
				Args:    []interface{}{fmt.Sprintf("exceed max query condition deepth: %d, host_property_filter.rules", querybuilder.MaxDeep)},
			}
		}
	}

	if o.Page.IsIllegal() {
		return cErr.RawErrorInfo{
			ErrCode: common.CCErrCommParamsInvalid,
			Args:    []interface{}{"page.limit"},
		}
	}

	return cErr.RawErrorInfo{}
}

// ListProcessRelatedInfoResult is the result for api ListProcessRelatedInfo
type ListProcessRelatedInfoResult struct {
	Set             SetDetailOfP             `json:"set"`
	Module          ModuleDetailOfP          `json:"module"`
	Host            HostDetailOfP            `json:"host"`
	ServiceInstance ServiceInstanceDetailOfP `json:"service_instance"`
	ProcessTemplate ProcessTemplateDetailOfP `json:"process_template"`
	Process         interface{}              `json:"process"`
}

type SetDetailOfP struct {
	SetID   int64  `json:"bk_set_id"`
	SetName string `json:"bk_set_name"`
	SetEnv  string `json:"bk_set_env"`
}

type ModuleDetailOfP struct {
	ModuleID   int64  `json:"bk_module_id"`
	ModuleName string `json:"bk_module_name"`
}

type HostDetailOfP struct {
	HostID  int64  `json:"bk_host_id"`
	CloudID int64  `json:"bk_cloud_id"`
	InnerIP string `json:"bk_host_innerip"`
}

type ServiceInstanceDetailOfP struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
}

type ProcessTemplateDetailOfP struct {
	ID int64 `json:"id"`
}

type ListProcessInstancesDetailsOption struct {
	ProcessIDs []int64  `json:"bk_process_ids"`
	Fields     []string `json:"fields"`
}

// Validate validates the input param
func (o *ListProcessInstancesDetailsOption) Validate() (rawError cErr.RawErrorInfo) {
	if len(o.ProcessIDs) == 0 || len(o.ProcessIDs) > common.BKMaxInstanceLimit {
		return cErr.RawErrorInfo{
			ErrCode: common.CCErrArrayLengthWrong,
			Args:    []interface{}{"bk_process_ids", common.BKMaxInstanceLimit},
		}
	}

	return cErr.RawErrorInfo{}
}

type ListProcessInstancesDetailsByIDsOption struct {
	BizID      int64    `json:"bk_biz_id"`
	ProcessIDs []int64  `json:"process_ids"`
	Page       BasePage `json:"page"`
}

// Validate validates the input param
func (o *ListProcessInstancesDetailsByIDsOption) Validate() (rawError cErr.RawErrorInfo) {
	if o.BizID <= 0 {
		return cErr.RawErrorInfo{
			ErrCode: common.CCErrCommParamsInvalid,
			Args:    []interface{}{"bk_biz_id"},
		}
	}

	if len(o.ProcessIDs) == 0 {
		return cErr.RawErrorInfo{
			ErrCode: common.CCErrCommParamsInvalid,
			Args:    []interface{}{"process_ids"},
		}
	}

	if o.Page.IsIllegal() {
		return cErr.RawErrorInfo{
			ErrCode: common.CCErrCommParamsInvalid,
			Args:    []interface{}{"page.limit"},
		}
	}

	return cErr.RawErrorInfo{}
}

type RemoveTemplateBindingOnModuleOption struct {
	BizID    int64 `json:"bk_biz_id"`
	ModuleID int64 `json:"bk_module_id"`
}

type UpdateProcessTemplateInput struct {
	BizID             int64                  `json:"bk_biz_id"`
	ProcessTemplateID int64                  `json:"process_template_id"`
	Property          map[string]interface{} `json:"process_property"`
}

type SocketBindType string

const (
	BindLocalHost SocketBindType = "1"
	BindAll       SocketBindType = "2"
	BindInnerIP   SocketBindType = "3"
	BindOuterIP   SocketBindType = "4"
)

func (p *SocketBindType) NeedIPFromHost() bool {
	if p == nil {
		return false
	}

	switch *p {
	case BindInnerIP, BindOuterIP:
		return true
	default:
		return false
	}
}

func (p *SocketBindType) IP(host map[string]interface{}) (string, error) {
	if p == nil {
		return "", errors.New("process template bind info ip is not set or is empty")
	}

	var ip string

	switch *p {
	case BindLocalHost:
		return "127.0.0.1", nil
	case BindAll:
		return "0.0.0.0", nil
	case BindInnerIP:
		if host == nil {
			return "", errors.New("process host is not specified to get bind inner ip")
		}
		ip = util.GetStrByInterface(host[common.BKHostInnerIPField])
	case BindOuterIP:
		if host == nil {
			return "", errors.New("process host is not specified to get bind outer ip")
		}
		ip = util.GetStrByInterface(host[common.BKHostOuterIPField])
	default:
		return "", errors.New("process template bind info ip is invalid")
	}

	if ip == "" {
		return "127.0.0.1", nil
	}

	index := strings.Index(strings.Trim(ip, ","), ",")
	if index == -1 {
		return ip, nil
	}
	return ip[:index], nil
}

func (p *SocketBindType) String() string {
	// TODO: how to support internationalization?
	if p == nil {
		return ""
	}
	switch *p {
	case BindLocalHost:
		return "127.0.0.1"
	case BindAll:
		return "0.0.0.0"
	case BindInnerIP:
		return "第一内网IP"
	case BindOuterIP:
		return "第一外网IP"
	default:
		return ""
	}
}

func (p SocketBindType) Validate() error {
	validValues := []SocketBindType{BindLocalHost, BindAll, BindInnerIP, BindOuterIP}
	if util.InArray(p, validValues) == false {
		return fmt.Errorf("invalid socket bind type, value: %s, available values: %+v", p, validValues)
	}
	return nil
}

type ProtocolType string

const (
	ProtocolTypeTCP ProtocolType = "1"
	ProtocolTypeUDP ProtocolType = "2"
)

func (p ProtocolType) String() string {
	switch p {
	case ProtocolTypeTCP:
		return "TCP"
	case ProtocolTypeUDP:
		return "UDP"
	default:
		return ""
	}
}

func (p ProtocolType) Validate() error {
	validValues := []ProtocolType{ProtocolTypeTCP, ProtocolTypeUDP}
	if util.InArray(p, validValues) == false {
		return fmt.Errorf("invalid protocol type, value: %s, available values: %+v", p, validValues)
	}
	return nil
}

type Process struct {
	ProcNum         *int64         `field:"proc_num" json:"proc_num" bson:"proc_num" structs:"proc_num" mapstructure:"proc_num"`
	StopCmd         *string        `field:"stop_cmd" json:"stop_cmd" bson:"stop_cmd" structs:"stop_cmd" mapstructure:"stop_cmd"`
	RestartCmd      *string        `field:"restart_cmd" json:"restart_cmd" bson:"restart_cmd" structs:"restart_cmd" mapstructure:"restart_cmd"`
	ForceStopCmd    *string        `field:"face_stop_cmd" json:"face_stop_cmd" bson:"face_stop_cmd" structs:"face_stop_cmd" mapstructure:"face_stop_cmd"`
	ProcessID       int64          `field:"bk_process_id" json:"bk_process_id" bson:"bk_process_id" structs:"bk_process_id" mapstructure:"bk_process_id"`
	FuncName        *string        `field:"bk_func_name" json:"bk_func_name" bson:"bk_func_name" structs:"bk_func_name" mapstructure:"bk_func_name"`
	WorkPath        *string        `field:"work_path" json:"work_path" bson:"work_path" structs:"work_path" mapstructure:"work_path"`
	Priority        *int64         `field:"priority" json:"priority" bson:"priority" structs:"priority" mapstructure:"priority"`
	ReloadCmd       *string        `field:"reload_cmd" json:"reload_cmd" bson:"reload_cmd" structs:"reload_cmd" mapstructure:"reload_cmd"`
	ProcessName     *string        `field:"bk_process_name" json:"bk_process_name" bson:"bk_process_name" structs:"bk_process_name" mapstructure:"bk_process_name"`
	PidFile         *string        `field:"pid_file" json:"pid_file" bson:"pid_file" structs:"pid_file" mapstructure:"pid_file"`
	AutoStart       *bool          `field:"auto_start" json:"auto_start" bson:"auto_start" structs:"auto_start" mapstructure:"auto_start"`
	StartCheckSecs  *int64         `field:"bk_start_check_secs" json:"bk_start_check_secs" bson:"bk_start_check_secs" structs:"bk_start_check_secs" mapstructure:"bk_start_check_secs"`
	LastTime        time.Time      `field:"last_time" json:"last_time" bson:"last_time" structs:"last_time" mapstructure:"last_time"`
	CreateTime      time.Time      `field:"create_time" json:"create_time" bson:"create_time" structs:"create_time" mapstructure:"create_time"`
	BusinessID      int64          `field:"bk_biz_id" json:"bk_biz_id" bson:"bk_biz_id" structs:"bk_biz_id" mapstructure:"bk_biz_id"`
	StartCmd        *string        `field:"start_cmd" json:"start_cmd" bson:"start_cmd" structs:"start_cmd" mapstructure:"start_cmd"`
	User            *string        `field:"user" json:"user" bson:"user" structs:"user" mapstructure:"user"`
	TimeoutSeconds  *int64         `field:"timeout" json:"timeout" bson:"timeout" structs:"timeout" mapstructure:"timeout"`
	Description     *string        `field:"description" json:"description" bson:"description" structs:"description" mapstructure:"description"`
	SupplierAccount string         `field:"bk_supplier_account" json:"bk_supplier_account" bson:"bk_supplier_account" structs:"bk_supplier_account" mapstructure:"bk_supplier_account"`
	StartParamRegex *string        `field:"bk_start_param_regex" json:"bk_start_param_regex" bson:"bk_start_param_regex" structs:"bk_start_param_regex" mapstructure:"bk_start_param_regex"`
	BindInfo        []ProcBindInfo `field:"bind_info" json:"bind_info" bson:"bind_info" structs:"bind_info" mapstructure:"bind_info"`
}

func (p *Process) Map() map[string]interface{} {
	var bindInfoArr []map[string]interface{}
	for _, row := range p.BindInfo {
		bindInfoArr = append(bindInfoArr, row.toKV())
	}
	procMap := map[string]interface{}{
		common.BKProcInstNum:      p.ProcNum,
		common.BKProcStopCmd:      p.StopCmd,
		common.BKProcRestartCmd:   p.RestartCmd,
		"face_stop_cmd":           p.ForceStopCmd,
		common.BKProcessIDField:   p.ProcessID,
		common.BKFuncName:         p.FuncName,
		common.BKWorkPath:         p.WorkPath,
		"priority":                p.Priority,
		common.BKProcReloadCmd:    p.ReloadCmd,
		common.BKProcessNameField: p.ProcessName,
		common.BKProcPidFile:      p.PidFile,
		"auto_start":              p.AutoStart,
		"bk_start_check_secs":     p.StartCheckSecs,
		common.BKAppIDField:       p.BusinessID,
		common.BKProcStartCmd:     p.StartCmd,
		common.BKUser:             p.User,
		common.BKProcTimeOut:      p.TimeoutSeconds,
		common.BKDescriptionField: p.Description,
		common.BKOwnerIDField:     p.SupplierAccount,
		common.BKStartParamRegex:  p.StartParamRegex,
		common.BKProcBindInfo:     bindInfoArr,
		common.CreateTimeField:    p.CreateTime,
		common.LastTimeField:      p.LastTime,
	}

	return procMap
}

type ServiceCategory struct {
	BizID int64 `field:"bk_biz_id" json:"bk_biz_id" bson:"bk_biz_id"`

	ID   int64  `field:"id" json:"id" bson:"id"`
	Name string `field:"name" json:"name" bson:"name"`

	RootID          int64  `field:"bk_root_id" json:"bk_root_id" bson:"bk_root_id"`
	ParentID        int64  `field:"bk_parent_id" json:"bk_parent_id" bson:"bk_parent_id"`
	SupplierAccount string `field:"bk_supplier_account" json:"bk_supplier_account" bson:"bk_supplier_account"`

	// IsBuiltIn indicates internal system service category, which shouldn't be modified.
	IsBuiltIn bool `field:"is_built_in" json:"is_built_in" bson:"is_built_in"`
}

func (sc *ServiceCategory) Validate() (field string, err error) {
	if len(sc.Name) == 0 {
		return "name", errors.New("name can't be empty")
	}
	if common.ServiceCategoryMaxLength < utf8.RuneCountInString(sc.Name) {
		return "name", fmt.Errorf("name too long, input: %d > max: %d", utf8.RuneCountInString(sc.Name), common.ServiceCategoryMaxLength)
	}
	match, err := regexp.MatchString(common.FieldTypeServiceCategoryRegexp, sc.Name)
	if nil != err {
		return "name", err
	}
	if !match {
		return "name", fmt.Errorf("name not match regex, input: %s", sc.Name)
	}
	return "", nil
}

type ServiceCategoryWithStatistics struct {
	ServiceCategory ServiceCategory `field:"category" json:"category" bson:"category"`
	UsageAmount     int64           `field:"usage_amount" json:"usage_amount" bson:"usage_amount"`
}

type ServiceTemplateWithStatistics struct {
	Template             ServiceTemplate `field:"template" json:"template" bson:"template"`
	ServiceInstanceCount int64           `field:"service_instance_count" json:"service_instance_count" bson:"service_instance_count"`
	ProcessInstanceCount int64           `field:"process_instance_count" json:"process_instance_count" bson:"process_instance_count"`
}

type ServiceTemplateDetail struct {
	ServiceTemplate  ServiceTemplate   `field:"service_template" json:"service_template" bson:"service_template" mapstructure:"service_template"`
	ProcessTemplates []ProcessTemplate `field:"process_templates" json:"process_templates" bson:"process_templates" mapstructure:"process_templates"`
}

type ServiceTemplate struct {
	BizID int64 `field:"bk_biz_id" json:"bk_biz_id" bson:"bk_biz_id"`

	ID int64 `field:"id" json:"id" bson:"id"`
	// name of this service, can not be empty
	Name string `field:"name" json:"name" bson:"name"`

	// the class of this service, each field means a class label.
	// now, the class must have two labels.
	ServiceCategoryID int64 `field:"service_category_id" json:"service_category_id" bson:"service_category_id"`

	Creator         string    `field:"creator" json:"creator" bson:"creator"`
	Modifier        string    `field:"modifier" json:"modifier" bson:"modifier"`
	CreateTime      time.Time `field:"create_time" json:"create_time" bson:"create_time"`
	LastTime        time.Time `field:"last_time" json:"last_time" bson:"last_time"`
	SupplierAccount string    `field:"bk_supplier_account" json:"bk_supplier_account" bson:"bk_supplier_account"`
}

func (st *ServiceTemplate) Validate(errProxy cErr.DefaultCCErrorIf) (field string, err error) {
	st.Name, err = util.ValidTopoNameField(st.Name, "name", errProxy)
	if err != nil {
		return "name", err
	}
	return "", nil
}

// this works for the process instance which is used for a template.
type ProcessTemplate struct {
	ID          int64  `field:"id" json:"id" bson:"id"`
	ProcessName string `field:"bk_process_name" json:"bk_process_name" bson:"bk_process_name"`
	BizID       int64  `field:"bk_biz_id" json:"bk_biz_id" bson:"bk_biz_id"`
	// the service template's, which this process template belongs to.
	ServiceTemplateID int64 `field:"service_template_id" json:"service_template_id" bson:"service_template_id"`

	// stores a process instance's data includes all the process's
	// properties's value.
	Property *ProcessProperty `field:"property" json:"property" bson:"property"`

	Creator         string    `field:"creator" json:"creator" bson:"creator"`
	Modifier        string    `field:"modifier" json:"modifier" bson:"modifier"`
	CreateTime      time.Time `field:"create_time" json:"create_time" bson:"create_time"`
	LastTime        time.Time `field:"last_time" json:"last_time" bson:"last_time"`
	SupplierAccount string    `field:"bk_supplier_account" json:"bk_supplier_account" bson:"bk_supplier_account"`
}

func (pt *ProcessTemplate) Validate() (field string, err error) {
	if pt.Property == nil {
		return "property", errors.New("property field shouldn't be nil")
	}
	field, err = pt.Property.Validate()
	if err != nil {
		return field, err
	}
	return "", nil
}

func IsAsDefaultValue(asDefaultValue *bool) bool {
	if asDefaultValue != nil {
		return *asDefaultValue
	}
	return false
}

func (pt *ProcessTemplate) NewProcess(bizID int64, supplierAccount string, host map[string]interface{}) (*Process, error) {
	now := time.Now()
	processInstance := &Process{
		LastTime:        now,
		CreateTime:      now,
		BusinessID:      bizID,
		SupplierAccount: supplierAccount,
	}

	property := pt.Property

	processInstance.ProcessName = property.ProcessName.Value
	processInstance.ProcNum = property.ProcNum.Value
	processInstance.StopCmd = property.StopCmd.Value
	processInstance.RestartCmd = property.RestartCmd.Value
	processInstance.ForceStopCmd = property.ForceStopCmd.Value
	processInstance.FuncName = property.FuncName.Value
	processInstance.WorkPath = property.WorkPath.Value
	processInstance.Priority = property.Priority.Value
	processInstance.ReloadCmd = property.ReloadCmd.Value
	processInstance.PidFile = property.PidFile.Value
	processInstance.AutoStart = property.AutoStart.Value
	processInstance.StartCheckSecs = property.StartCheckSecs.Value
	processInstance.StartCmd = property.StartCmd.Value
	processInstance.User = property.User.Value
	processInstance.TimeoutSeconds = property.TimeoutSeconds.Value
	processInstance.Description = property.Description.Value
	processInstance.StartParamRegex = property.StartParamRegex.Value

	var err error
	processInstance.BindInfo, err = property.BindInfo.NewProcBindInfo(host)
	if err != nil {
		return nil, err
	}

	return processInstance, nil
}

func FilterValidFields(fields []string) []string {
	allFields := GetAllProcessPropertyFields()

	result := make([]string, 0)
	for _, field := range fields {
		if util.InStrArr(allFields, field) {
			result = append(result, field)
		}
	}
	return result
}

func GetAllProcessPropertyFields() []string {
	fields := make([]string, 0)
	fields = append(fields, "bk_func_name")
	fields = append(fields, "bk_process_name")
	fields = append(fields, "bk_start_param_regex")
	fields = append(fields, "bk_start_check_secs")
	fields = append(fields, "user")
	fields = append(fields, "stop_cmd")
	fields = append(fields, "proc_num")
	fields = append(fields, "port")
	fields = append(fields, "description")
	fields = append(fields, "protocol")
	fields = append(fields, "timeout")
	fields = append(fields, "auto_start")
	fields = append(fields, "pid_file")
	fields = append(fields, "reload_cmd")
	fields = append(fields, "restart_cmd")
	fields = append(fields, "face_stop_cmd")
	fields = append(fields, "work_path")
	fields = append(fields, "bind_ip")
	fields = append(fields, "priority")
	fields = append(fields, "start_cmd")
	fields = append(fields, common.BKProcPortEnable)
	fields = append(fields, "bk_gateway_ip")
	fields = append(fields, "bk_gateway_port")
	fields = append(fields, "bk_gateway_protocol")
	fields = append(fields, "bk_gateway_city")

	return fields
}

// ExtractChangeInfo get changes that will be applied to process instance
func (pt *ProcessTemplate) ExtractChangeInfo(i *Process, host map[string]interface{}) (mapstr.MapStr, bool, error) {
	t := pt.Property
	var changed bool
	if t == nil || i == nil {
		return nil, false, nil
	}

	process := make(mapstr.MapStr)
	if IsAsDefaultValue(t.ProcNum.AsDefaultValue) {
		if t.ProcNum.Value == nil && i.ProcNum != nil {
			process["proc_num"] = nil
			changed = true
		} else if t.ProcNum.Value != nil && i.ProcNum == nil {
			process["proc_num"] = *t.ProcNum.Value
			changed = true
		} else if t.ProcNum.Value != nil && i.ProcNum != nil && *t.ProcNum.Value != *i.ProcNum {
			process["proc_num"] = *t.ProcNum.Value
			changed = true
		}
	}

	if IsAsDefaultValue(t.StopCmd.AsDefaultValue) {
		if t.StopCmd.Value == nil && i.StopCmd != nil {
			process["stop_cmd"] = nil
			changed = true
		} else if t.StopCmd.Value != nil && i.StopCmd == nil {
			process["stop_cmd"] = *t.StopCmd.Value
			changed = true
		} else if t.StopCmd.Value != nil && i.StopCmd != nil && *t.StopCmd.Value != *i.StopCmd {
			process["stop_cmd"] = *t.StopCmd.Value
			changed = true
		}
	}

	if IsAsDefaultValue(t.RestartCmd.AsDefaultValue) {
		if t.RestartCmd.Value == nil && i.RestartCmd != nil {
			process["restart_cmd"] = nil
			changed = true
		} else if t.RestartCmd.Value != nil && i.RestartCmd == nil {
			process["restart_cmd"] = *t.RestartCmd.Value
			changed = true
		} else if t.RestartCmd.Value != nil && i.RestartCmd != nil && *t.RestartCmd.Value != *i.RestartCmd {
			process["restart_cmd"] = *t.RestartCmd.Value
			changed = true
		}
	}

	if IsAsDefaultValue(t.ForceStopCmd.AsDefaultValue) {
		if t.ForceStopCmd.Value == nil && i.ForceStopCmd != nil {
			process["face_stop_cmd"] = nil
			changed = true
		} else if t.ForceStopCmd.Value != nil && i.ForceStopCmd == nil {
			process["face_stop_cmd"] = *t.ForceStopCmd.Value
			changed = true
		} else if t.ForceStopCmd.Value != nil && i.ForceStopCmd != nil && *t.ForceStopCmd.Value != *i.ForceStopCmd {
			process["face_stop_cmd"] = *t.ForceStopCmd.Value
			changed = true
		}
	}

	if IsAsDefaultValue(t.FuncName.AsDefaultValue) {
		if t.FuncName.Value == nil && i.FuncName != nil {
			process["bk_func_name"] = nil
			changed = true
		} else if t.FuncName.Value != nil && i.FuncName == nil {
			process["bk_func_name"] = *t.FuncName.Value
			changed = true
		} else if t.FuncName.Value != nil && i.FuncName != nil && *t.FuncName.Value != *i.FuncName {
			process["bk_func_name"] = *t.FuncName.Value
			changed = true
		}
	}

	if IsAsDefaultValue(t.WorkPath.AsDefaultValue) {
		if t.WorkPath.Value == nil && i.WorkPath != nil {
			process["work_path"] = nil
			changed = true
		} else if t.WorkPath.Value != nil && i.WorkPath == nil {
			process["work_path"] = *t.WorkPath.Value
			changed = true
		} else if t.WorkPath.Value != nil && i.WorkPath != nil && *t.WorkPath.Value != *i.WorkPath {
			process["work_path"] = *t.WorkPath.Value
			changed = true
		}
	}

	if IsAsDefaultValue(t.ReloadCmd.AsDefaultValue) {
		if t.ReloadCmd.Value == nil && i.ReloadCmd != nil {
			process["reload_cmd"] = nil
			changed = true
		} else if t.ReloadCmd.Value != nil && i.ReloadCmd == nil {
			process["reload_cmd"] = *t.ReloadCmd.Value
			changed = true
		} else if t.ReloadCmd.Value != nil && i.ReloadCmd != nil && *t.ReloadCmd.Value != *i.ReloadCmd {
			process["reload_cmd"] = *t.ReloadCmd.Value
			changed = true
		}
	}

	if IsAsDefaultValue(t.ProcessName.AsDefaultValue) {
		if t.ProcessName.Value == nil && i.ProcessName != nil {
			process["bk_process_name"] = nil
			changed = true
		} else if t.ProcessName.Value != nil && i.ProcessName == nil {
			process["bk_process_name"] = *t.ProcessName.Value
			changed = true
		} else if t.ProcessName.Value != nil && i.ProcessName != nil && *t.ProcessName.Value != *i.ProcessName {
			process["bk_process_name"] = *t.ProcessName.Value
			changed = true
		}
	}

	if IsAsDefaultValue(t.AutoStart.AsDefaultValue) {
		if t.AutoStart.Value == nil && i.AutoStart != nil {
			process["auto_start"] = nil
			changed = true
		} else if t.AutoStart.Value != nil && i.AutoStart == nil {
			process["auto_start"] = *t.AutoStart.Value
			changed = true
		} else if t.AutoStart.Value != nil && i.AutoStart != nil && *t.AutoStart.Value != *i.AutoStart {
			process["auto_start"] = *t.AutoStart.Value
			changed = true
		}
	}

	if IsAsDefaultValue(t.StartCheckSecs.AsDefaultValue) {
		if t.StartCheckSecs.Value != nil && i.StartCheckSecs == nil {
			process["bk_start_check_secs"] = *t.StartCheckSecs.Value
			changed = true
		} else if t.StartCheckSecs.Value == nil && i.StartCheckSecs != nil {
			process["bk_start_check_secs"] = nil
			changed = true
		} else if t.StartCheckSecs.Value != nil && i.StartCheckSecs != nil && *t.StartCheckSecs.Value != *i.StartCheckSecs {
			process["bk_start_check_secs"] = *t.StartCheckSecs.Value
			changed = true
		}
	}

	if IsAsDefaultValue(t.StartCmd.AsDefaultValue) {
		if t.StartCmd.Value == nil && i.StartCmd != nil {
			process["start_cmd"] = nil
			changed = true
		} else if t.StartCmd.Value != nil && i.StartCmd == nil {
			process["start_cmd"] = *t.StartCmd.Value
			changed = true
		} else if t.StartCmd.Value != nil && i.StartCmd != nil && *t.StartCmd.Value != *i.StartCmd {
			process["start_cmd"] = *t.StartCmd.Value
			changed = true
		}
	}

	if IsAsDefaultValue(t.User.AsDefaultValue) {
		if t.User.Value == nil && i.User != nil {
			process["user"] = nil
			changed = true
		} else if t.User.Value != nil && i.User == nil {
			process["user"] = *t.User.Value
			changed = true
		} else if t.User.Value != nil && i.User != nil && *t.User.Value != *i.User {
			process["user"] = *t.User.Value
			changed = true
		}
	}

	if IsAsDefaultValue(t.TimeoutSeconds.AsDefaultValue) {
		if t.TimeoutSeconds.Value != nil && i.TimeoutSeconds == nil {
			process["timeout"] = *t.TimeoutSeconds.Value
			changed = true
		} else if t.TimeoutSeconds.Value == nil && i.TimeoutSeconds != nil {
			process["timeout"] = nil
			changed = true
		} else if t.TimeoutSeconds.Value != nil && i.TimeoutSeconds != nil && *t.TimeoutSeconds.Value != *i.TimeoutSeconds {
			process["timeout"] = *t.TimeoutSeconds.Value
			changed = true
		}
	}

	if IsAsDefaultValue(t.Description.AsDefaultValue) {
		if t.Description.Value == nil && i.Description != nil {
			process["description"] = nil
			changed = true
		} else if t.Description.Value != nil && i.Description == nil {
			process["description"] = *t.Description.Value
			changed = true
		} else if t.Description.Value != nil && i.Description != nil && *t.Description.Value != *i.Description {
			process["description"] = *t.Description.Value
			changed = true
		}
	}

	if IsAsDefaultValue(t.StartParamRegex.AsDefaultValue) {
		if t.StartParamRegex.Value == nil && i.StartParamRegex != nil {
			process["bk_start_param_regex"] = nil
			changed = true
		} else if t.StartParamRegex.Value != nil && i.StartParamRegex == nil {
			process["bk_start_param_regex"] = *t.StartParamRegex.Value
			changed = true
		} else if t.StartParamRegex.Value != nil && i.StartParamRegex != nil && *t.StartParamRegex.Value != *i.StartParamRegex {
			process["bk_start_param_regex"] = *t.StartParamRegex.Value
			changed = true
		}
	}

	if IsAsDefaultValue(t.PidFile.AsDefaultValue) {
		if t.PidFile.Value == nil && i.PidFile != nil {
			process["pid_file"] = nil
			changed = true
		} else if t.PidFile.Value != nil && i.PidFile == nil {
			process["pid_file"] = *t.PidFile.Value
			changed = true
		} else if t.PidFile.Value != nil && i.PidFile != nil && *t.PidFile.Value != *i.PidFile {
			process["pid_file"] = *t.PidFile.Value
			changed = true
		}
	}

	if IsAsDefaultValue(t.Priority.AsDefaultValue) {
		if t.Priority.Value == nil && i.Priority != nil {
			process["priority"] = nil
			changed = true
		} else if t.Priority.Value != nil && i.Priority == nil {
			process["priority"] = *t.Priority.Value
			changed = true
		} else if t.Priority.Value != nil && i.Priority != nil && *t.Priority.Value != *i.Priority {
			process["priority"] = *t.Priority.Value
			changed = true
		}
	}

	bindInfo, bindInfoChanged, bindInfoIsNamePortChanged, err := t.BindInfo.ExtractChangeInfoBindInfo(i, host)
	if err != nil {
		return nil, false, err
	}

	process[common.BKProcBindInfo] = bindInfo
	if bindInfoChanged {
		changed = true
	}
	if bindInfoIsNamePortChanged {
		bindInfoIsNamePortChanged = true
	}

	return process, changed, nil
}

// FilterEditableFields only return editable fields
func (pt *ProcessTemplate) GetEditableFields(fields []string) []string {
	editableFields := pt.ExtractEditableFields()
	result := make([]string, 0)
	for _, field := range fields {
		if util.InStrArr(editableFields, field) {
			result = append(result, field)
		}
	}
	return result
}

func (pt *ProcessTemplate) ExtractEditableFields() []string {
	editableFields := make([]string, 0)
	property := pt.Property
	if IsAsDefaultValue(property.FuncName.AsDefaultValue) == false {
		editableFields = append(editableFields, "bk_func_name")
	}
	if IsAsDefaultValue(property.ProcessName.AsDefaultValue) == false {
		editableFields = append(editableFields, "bk_process_name")
	}
	if IsAsDefaultValue(property.StartParamRegex.AsDefaultValue) == false {
		editableFields = append(editableFields, "bk_start_param_regex")
	}
	if IsAsDefaultValue(property.StartCheckSecs.AsDefaultValue) == false {
		editableFields = append(editableFields, "bk_start_check_secs")
	}
	if IsAsDefaultValue(property.User.AsDefaultValue) == false {
		editableFields = append(editableFields, "user")
	}
	if IsAsDefaultValue(property.StopCmd.AsDefaultValue) == false {
		editableFields = append(editableFields, "stop_cmd")
	}
	if IsAsDefaultValue(property.ProcNum.AsDefaultValue) == false {
		editableFields = append(editableFields, "proc_num")
	}

	if IsAsDefaultValue(property.Description.AsDefaultValue) == false {
		editableFields = append(editableFields, "description")
	}

	if IsAsDefaultValue(property.TimeoutSeconds.AsDefaultValue) == false {
		editableFields = append(editableFields, "timeout")
	}
	if IsAsDefaultValue(property.AutoStart.AsDefaultValue) == false {
		editableFields = append(editableFields, "auto_start")
	}
	if IsAsDefaultValue(property.PidFile.AsDefaultValue) == false {
		editableFields = append(editableFields, "pid_file")
	}
	if IsAsDefaultValue(property.ReloadCmd.AsDefaultValue) == false {
		editableFields = append(editableFields, "reload_cmd")
	}
	if IsAsDefaultValue(property.RestartCmd.AsDefaultValue) == false {
		editableFields = append(editableFields, "restart_cmd")
	}
	if IsAsDefaultValue(property.ForceStopCmd.AsDefaultValue) == false {
		editableFields = append(editableFields, "face_stop_cmd")
	}
	if IsAsDefaultValue(property.WorkPath.AsDefaultValue) == false {
		editableFields = append(editableFields, "work_path")
	}

	if IsAsDefaultValue(property.Priority.AsDefaultValue) == false {
		editableFields = append(editableFields, "priority")
	}
	if IsAsDefaultValue(property.StartCmd.AsDefaultValue) == false {
		editableFields = append(editableFields, "start_cmd")
	}
	editableFields = append(editableFields, common.BKProcBindInfo)

	return editableFields
}

// InstanceUpdate is used for update instance's value
func (pt *ProcessTemplate) ExtractInstanceUpdateData(input *Process, host map[string]interface{}) (
	map[string]interface{}, error) {

	data := make(map[string]interface{})
	property := pt.Property
	if IsAsDefaultValue(property.FuncName.AsDefaultValue) == false {
		if input.FuncName != nil {
			data["bk_func_name"] = *input.FuncName
		}
	}
	if IsAsDefaultValue(property.ProcessName.AsDefaultValue) == false {
		if input.ProcessName != nil {
			data["bk_process_name"] = *input.ProcessName
		}
	}
	if IsAsDefaultValue(property.StartParamRegex.AsDefaultValue) == false {
		if input.StartParamRegex != nil {
			data["bk_start_param_regex"] = *input.StartParamRegex
		}
	}
	if IsAsDefaultValue(property.StartCheckSecs.AsDefaultValue) == false {
		if input.StartCheckSecs != nil {
			data["bk_start_check_secs"] = *input.StartCheckSecs
		}
	}
	if IsAsDefaultValue(property.User.AsDefaultValue) == false {
		if input.User != nil {
			data["user"] = *input.User
		}
	}
	if IsAsDefaultValue(property.StopCmd.AsDefaultValue) == false {
		if input.StopCmd != nil {
			data["stop_cmd"] = *input.StopCmd
		}
	}
	if IsAsDefaultValue(property.ProcNum.AsDefaultValue) == false {
		if input.ProcNum != nil {
			data["proc_num"] = *input.ProcNum
		}
	}

	if IsAsDefaultValue(property.Description.AsDefaultValue) == false {
		if input.Description != nil {
			data["description"] = *input.Description
		}
	}

	if IsAsDefaultValue(property.TimeoutSeconds.AsDefaultValue) == false {
		if input.TimeoutSeconds != nil {
			data["timeout"] = *input.TimeoutSeconds
		}
	}
	if IsAsDefaultValue(property.AutoStart.AsDefaultValue) == false {
		if input.AutoStart != nil {
			data["auto_start"] = *input.AutoStart
		}
	}
	if IsAsDefaultValue(property.PidFile.AsDefaultValue) == false {
		if input.PidFile != nil {
			data["pid_file"] = *input.PidFile
		}
	}
	if IsAsDefaultValue(property.ReloadCmd.AsDefaultValue) == false {
		if input.ReloadCmd != nil {
			data["reload_cmd"] = *input.ReloadCmd
		}
	}
	if IsAsDefaultValue(property.RestartCmd.AsDefaultValue) == false {
		if input.RestartCmd != nil {
			data["restart_cmd"] = *input.RestartCmd
		}
	}
	if IsAsDefaultValue(property.ForceStopCmd.AsDefaultValue) == false {
		if input.ForceStopCmd != nil {
			data["face_stop_cmd"] = *input.ForceStopCmd
		}
	}
	if IsAsDefaultValue(property.WorkPath.AsDefaultValue) == false {
		if input.WorkPath != nil {
			data["work_path"] = *input.WorkPath
		}
	}

	if IsAsDefaultValue(property.Priority.AsDefaultValue) == false {
		if input.Priority != nil {
			data["priority"] = *input.Priority
		}
	}
	if IsAsDefaultValue(property.StartCmd.AsDefaultValue) == false {
		if input.StartCmd != nil {
			data["start_cmd"] = *input.StartCmd
		}
	}

	// bind info 每次都是全量更新
	var err error
	data[common.BKProcBindInfo], err = pt.Property.BindInfo.ExtractInstanceUpdateData(input, host)
	if err != nil {
		return nil, err
	}

	return data, nil
}

type ProcessProperty struct {
	ProcNum      PropertyInt64  `field:"proc_num" json:"proc_num" bson:"proc_num" validate:"max=10000,min=1"`
	StopCmd      PropertyString `field:"stop_cmd" json:"stop_cmd" bson:"stop_cmd"`
	RestartCmd   PropertyString `field:"restart_cmd" json:"restart_cmd" bson:"restart_cmd"`
	ForceStopCmd PropertyString `field:"face_stop_cmd" json:"face_stop_cmd" bson:"face_stop_cmd"`
	FuncName     PropertyString `field:"bk_func_name" json:"bk_func_name" bson:"bk_func_name" validate:"required"`
	WorkPath     PropertyString `field:"work_path" json:"work_path" bson:"work_path"`
	//BindIP             PropertyBindIP   `field:"bind_ip" json:"bind_ip" bson:"bind_ip"`
	Priority    PropertyInt64  `field:"priority" json:"priority" bson:"priority" validate:"max=10000,min=1"`
	ReloadCmd   PropertyString `field:"reload_cmd" json:"reload_cmd" bson:"reload_cmd"`
	ProcessName PropertyString `field:"bk_process_name" json:"bk_process_name" bson:"bk_process_name" validate:"required"`
	//Port               PropertyPort     `field:"port" json:"port" bson:"port"`
	PidFile        PropertyString `field:"pid_file" json:"pid_file" bson:"pid_file"`
	AutoStart      PropertyBool   `field:"auto_start" json:"auto_start" bson:"auto_start"`
	StartCheckSecs PropertyInt64  `field:"bk_start_check_secs" json:"bk_start_check_secs" bson:"bk_start_check_secs" validate:"max=600,min=1"`
	StartCmd       PropertyString `field:"start_cmd" json:"start_cmd" bson:"start_cmd"`
	User           PropertyString `field:"user" json:"user" bson:"user"`
	TimeoutSeconds PropertyInt64  `field:"timeout" json:"timeout" bson:"timeout" validate:"max=10000,min=1"`
	//Protocol           PropertyProtocol `field:"protocol" json:"protocol" bson:"protocol"`
	Description     PropertyString `field:"description" json:"description" bson:"description"`
	StartParamRegex PropertyString `field:"bk_start_param_regex" json:"bk_start_param_regex" bson:"bk_start_param_regex"`
	//PortEnable         PropertyBool     `field:"bk_enable_port" json:"bk_enable_port" bson:"bk_enable_port"`
	//GatewayIP       PropertyString   `field:"bk_gateway_ip" json:"bk_gateway_ip" bson:"bk_gateway_ip"`
	//GatewayPort     PropertyString   `field:"bk_gateway_port" json:"bk_gateway_port" bson:"bk_gateway_port"`
	//GatewayProtocol PropertyProtocol `field:"bk_gateway_protocol" json:"bk_gateway_protocol" bson:"bk_gateway_protocol"`
	//GatewayCity     PropertyString   `field:"bk_gateway_city" json:"bk_gateway_city" bson:"bk_gateway_city"`

	BindInfo ProcPropertyBindInfo `field:"bind_info" json:"bind_info" bson:"bind_info" structs:"bind_info" mapstructure:"bind_info"`
}

func (pt *ProcessProperty) Validate() (field string, err error) {
	// call all field's Validate method one by one
	propertyInterfaceType := reflect.TypeOf((*ProcessPropertyInterface)(nil)).Elem()
	selfVal := reflect.ValueOf(pt).Elem()
	selfType := reflect.TypeOf(pt).Elem()
	fieldCount := selfVal.NumField()
	for fieldIdx := 0; fieldIdx < fieldCount; fieldIdx++ {
		field := selfType.Field(fieldIdx)
		fieldVal := selfVal.Field(fieldIdx)
		tag := field.Tag.Get("json")
		fieldName := strings.Split(tag, ",")[0]

		if fieldName == common.BKProcBindInfo {
			continue
		}
		// check implements interface
		fieldValType := fieldVal.Addr().Type()
		if !fieldValType.Implements(propertyInterfaceType) {
			msg := fmt.Sprintf("field %s of type: %s should implements %s", field.Name, fieldVal.Type().Elem().Name(), propertyInterfaceType.Name())
			panic(msg)
		}

		// call validate method by interface
		checkResult := fieldVal.Addr().MethodByName("Validate").Call([]reflect.Value{})
		out := checkResult[0]
		if !out.IsNil() {
			err := out.Interface().(error)

			return fieldName, err
		}
	}

	if fieldName, err := pt.BindInfo.Validate(); err != nil {
		return fieldName, err
	}

	if pt.ProcessName.Value == nil || len(*pt.ProcessName.Value) == 0 {
		return "bk_process_name", fmt.Errorf("field [%s] is required", "bk_process_name")
	}
	if pt.FuncName.Value == nil || len(*pt.FuncName.Value) == 0 {
		return "bk_func_name", fmt.Errorf("field [%s] is required", "bk_func_name")
	}
	if pt.StartCheckSecs.Value != nil {
		if *pt.StartCheckSecs.Value < 1 || *pt.StartCheckSecs.Value > 10000 {
			return "bk_start_check_secs", fmt.Errorf("field %s value must in range [1, 600]", "bk_start_check_secs")
		}
	}
	if pt.TimeoutSeconds.Value != nil {
		if *pt.TimeoutSeconds.Value < 1 || *pt.TimeoutSeconds.Value > 10000 {
			return "timeout", fmt.Errorf("field %s value must in range [1, 10000]", "timeout")
		}
	}
	if pt.ProcNum.Value != nil {
		if *pt.ProcNum.Value < 1 || *pt.ProcNum.Value > 10000 {
			return "proc_num", fmt.Errorf("field %s value must in range [1, 10000]", "proc_num")
		}
	}
	if pt.Priority.Value != nil {
		if *pt.Priority.Value < common.MinProcessPrio || *pt.Priority.Value > common.MaxProcessPrio {
			return "priority",
			fmt.Errorf("field %s value must in range [%d, %d]", "priority",common.MinProcessPrio,common.MaxProcessPrio)
		}
	}

	return "", nil
}

// Update all not nil field from input to pt
// rawProperty allows us set property field to nil
//  参数rawProperty，input 数据是一样的，只不过一个是map,一个struct。 因为struct 是有默认行为的。 rawProperty为了获取用户是否输入
func (pt *ProcessProperty) Update(input ProcessProperty, rawProperty map[string]interface{}) {
	selfType := reflect.TypeOf(pt).Elem()
	selfVal := reflect.ValueOf(pt).Elem()
	inputVal := reflect.ValueOf(input)
	fieldCount := selfVal.NumField()
	updateIgnoreField := []string{"FuncName"}
	for fieldIdx := 0; fieldIdx < fieldCount; fieldIdx++ {
		fieldName := selfType.Field(fieldIdx).Name
		if util.InArray(fieldName, updateIgnoreField) == true {
			continue
		}
		fieldTag := selfType.Field(fieldIdx).Tag.Get("json")
		// 如果rawProperty不存在的字段，表示没有传递该字段，表示的型行为是不修改改字段
		if _, ok := rawProperty[fieldTag]; ok == false {
			continue
		}
		// bind info 是特有方法更新
		if fieldTag == common.BKProcBindInfo {
			continue
		}
		inputField := inputVal.Field(fieldIdx)
		selfField := selfVal.Field(fieldIdx)
		subFieldCount := inputField.NumField()
		// subFields: Value, AsDefaultValue
		for subFieldIdx := 0; subFieldIdx < subFieldCount; subFieldIdx++ {
			selfFieldValuePtr := selfField.Field(subFieldIdx)
			inputFieldPtr := inputField.Field(subFieldIdx)
			if inputFieldPtr.IsNil() {
				selfFieldValuePtr.Set(inputFieldPtr)
				continue
			}
			inputFieldValue := inputFieldPtr.Elem()

			if selfFieldValuePtr.Kind() == reflect.Ptr {
				if selfFieldValuePtr.IsNil() && selfFieldValuePtr.CanSet() {
					selfFieldValuePtr.Set(reflect.New(selfFieldValuePtr.Type().Elem()))
				}
			}

			selfFieldValue := selfFieldValuePtr.Elem()
			selfFieldValue.Set(inputFieldValue)
		}
	}
	bindInfo := &pt.BindInfo
	bindInfo.Update(input, rawProperty)

	return
}

type ProcessPropertyInterface interface {
	Validate() error
}

type PropertyInt64 struct {
	Value *int64 `field:"value" json:"value" bson:"value"`

	// AsDefaultValue records the relations between process instance's property and
	// whether it's used as a default value, the empty value can also be a default value.
	// If the property's value is used as a default value, then this property
	// can not be changed in all the process instance's created by this process
	// template. or, it can only be changed to this default value.
	AsDefaultValue *bool `field:"as_default_value" json:"as_default_value" bson:"as_default_value"`
}

func (ti *PropertyInt64) Validate() error {
	return nil
}

// PropertyInt64String is a string field that parse into int64
type PropertyInt64String struct {
	Value *string `field:"value" json:"value" bson:"value"`

	// AsDefaultValue records the relations between process instance's property and
	// whether it's used as a default value, the empty value can also be a default value.
	// If the property's value is used as a default value, then this property
	// can not be changed in all the process instance's created by this process
	// template. or, it can only be changed to this default value.
	AsDefaultValue *bool `field:"as_default_value" json:"as_default_value" bson:"as_default_value"`
}

func (ti *PropertyInt64String) Validate() error {
	if ti.Value != nil {
		// 兼容前端
		if *ti.Value == "" {
			ti.Value = nil
			return nil
		}

		if _, err := strconv.ParseInt(*ti.Value, 10, 64); err != nil {
			return err
		}
	}
	return nil
}

type PropertyBool struct {
	Value          *bool `field:"value" json:"value" bson:"value"`
	AsDefaultValue *bool `field:"as_default_value" json:"as_default_value" bson:"as_default_value"`
}

func (ti *PropertyBool) Validate() error {
	return nil
}

type PropertyString struct {
	Value          *string `field:"value" json:"value" bson:"value"`
	AsDefaultValue *bool   `field:"as_default_value" json:"as_default_value" bson:"as_default_value"`
}

func (ti *PropertyString) Validate() error {
	if ti == nil {
		return nil
	}
	if ti.Value != nil {
		value := *ti.Value
		if len(value) == 0 {
			return nil
		}
	}
	return nil
}

var (
	ProcessPortFormat = regexp.MustCompile(`^(((([1-9][0-9]{0,3})|([1-5][0-9]{4})|(6[0-4][0-9]{3})|(65[0-4][0-9]{2})|(655[0-2][0-9])|(6553[0-5]))-(([1-9][0-9]{0,3})|([1-5][0-9]{4})|(6[0-4][0-9]{3})|(65[0-4][0-9]{2})|(655[0-2][0-9])|(6553[0-5])))|((([1-9][0-9]{0,3})|([1-5][0-9]{4})|(6[0-4][0-9]{3})|(65[0-4][0-9]{2})|(655[0-2][0-9])|(6553[0-5]))))$`)
)

type PropertyPort struct {
	Value          *string `field:"value" json:"value" bson:"value"`
	AsDefaultValue *bool   `field:"as_default_value" json:"as_default_value" bson:"as_default_value"`
}

func (ti *PropertyPort) Validate() error {
	if ti.Value == nil || len(*ti.Value) == 0 {
		return errors.New("port is not set or is empty")
	}

	if matched := ProcessPortFormat.MatchString(*ti.Value); matched == false {
		return fmt.Errorf("port format invalid")
	}
	var tmpPortArr []propertyPortItem
	strPortItemArr := strings.Split(*ti.Value, ",")
	for _, strPortItem := range strPortItemArr {
		portArr := strings.Split(strPortItem, "-")
		var start, end int64
		var err error
		start, err = util.GetInt64ByInterface(portArr[0])
		if err != nil {
			return fmt.Errorf("parse start port to int failed, err: %v", err)
		}
		if len(portArr) > 1 {
			end, err = util.GetInt64ByInterface(portArr[1])
			if err != nil {
				return fmt.Errorf("parse end port to int failed, err: %v", err)
			}
		} else {
			end = start
		}
		if start > end {
			return fmt.Errorf("port format invalid, start > end")
		}
		for _, tmpItem := range tmpPortArr {
			if !(end < tmpItem.start || start > tmpItem.end) {
				return fmt.Errorf("port format invalid,  port duplicate:" + strPortItem)
			}
		}
		tmpPortArr = append(tmpPortArr, propertyPortItem{start: start, end: end})
	}

	return nil
}

// propertyPortItem 记录进程端口起始范围
type propertyPortItem struct {
	start int64
	end   int64
}

type PropertyBindIP struct {
	Value          *SocketBindType `field:"value" json:"value" bson:"value"`
	AsDefaultValue *bool           `field:"as_default_value" json:"as_default_value" bson:"as_default_value"`
}

func (ti *PropertyBindIP) Validate() error {
	if ti.Value == nil || len(*ti.Value) == 0 {
		return errors.New("ip is not set or is empty")
	}

	if err := ti.Value.Validate(); err != nil {
		return err
	}
	return nil
}

type PropertyProtocol struct {
	Value          *ProtocolType `field:"value" json:"value" bson:"value"`
	AsDefaultValue *bool         `field:"as_default_value" json:"as_default_value" bson:"as_default_value"`
}

func (ti *PropertyProtocol) Validate() error {
	if ti.Value == nil || len(*ti.Value) == 0 {
		return errors.New("protocol is not set or is empty")
	}

	if err := ti.Value.Validate(); err != nil {
		return err
	}
	return nil
}

// ServiceInstance is a service, which created when a host binding with a service template.
type ServiceInstance struct {
	BizID  int64           `field:"bk_biz_id" json:"bk_biz_id" bson:"bk_biz_id"`
	ID     int64           `field:"id" json:"id" bson:"id"`
	Name   string          `field:"name" json:"name" bson:"name"`
	Labels selector.Labels `field:"labels" json:"labels" bson:"labels"`

	// the template id can not be updated, once the service is created.
	// it can be 0 when the service is not created with a service template.
	ServiceTemplateID int64 `field:"service_template_id" json:"service_template_id" bson:"service_template_id"`
	HostID            int64 `field:"bk_host_id" json:"bk_host_id" bson:"bk_host_id"`

	// the module that this service belongs to.
	ModuleID int64 `field:"bk_module_id" json:"bk_module_id" bson:"bk_module_id"`

	Creator         string    `field:"creator" json:"creator" bson:"creator"`
	Modifier        string    `field:"modifier" json:"modifier" bson:"modifier"`
	CreateTime      time.Time `field:"create_time" json:"create_time" bson:"create_time"`
	LastTime        time.Time `field:"last_time" json:"last_time" bson:"last_time"`
	SupplierAccount string    `field:"bk_supplier_account" json:"bk_supplier_account" bson:"bk_supplier_account"`
}

func (si *ServiceInstance) Validate() (field string, err error) {
	/*
		if len(si.Name) == 0 {
			return "name", errors.New("name can't be empty")
		}
	*/

	if len(si.Name) > common.NameFieldMaxLength {
		return "name", fmt.Errorf("name too long, input: %d > max: %d", len(si.Name), common.NameFieldMaxLength)
	}
	return "", nil
}

type ServiceInstanceDetail struct {
	ServiceInstance
	ProcessInstances []ProcessInstanceNG `field:"process_instances" json:"process_instances" bson:"process_instances"`
}

type ServiceInstanceWithTopoPath struct {
	ServiceInstance
	TopoPath []TopoInstanceNodeSimplify `field:"topo_path" json:"topo_path" bson:"topo_path"`
}

// ProcessInstanceRelation record which service instance and process template are current process binding, process identified by ProcessID
type ProcessInstanceRelation struct {
	BizID int64 `field:"bk_biz_id" json:"bk_biz_id" bson:"bk_biz_id"`

	// unique field, 1:1 mapping with ProcessInstance.
	ProcessID         int64 `field:"bk_process_id" json:"bk_process_id" bson:"bk_process_id"`
	ServiceInstanceID int64 `field:"service_instance_id" json:"service_instance_id" bson:"service_instance_id"`

	// ProcessTemplateID indicate which template are current process instantiate from.
	ProcessTemplateID int64 `field:"process_template_id" json:"process_template_id" bson:"process_template_id"`

	// redundant field for accelerating processes by HostID
	HostID          int64  `field:"bk_host_id" json:"bk_host_id" bson:"bk_host_id"`
	SupplierAccount string `field:"bk_supplier_account" json:"bk_supplier_account" bson:"bk_supplier_account"`
}

func (pir *ProcessInstanceRelation) Validate() (field string, err error) {
	return "", nil
}

type HostProcessRelation struct {
	HostID    int64 `json:"bk_host_id" bson:"bk_host_id"`
	ProcessID int64 `json:"bk_process_id" bson:"bk_process_id"`
}

type ProcessInstanceNameIDs struct {
	ProcessName string  `json:"bk_process_name"`
	ProcessIDs  []int64 `json:"process_ids"`
}

type ProcessInstanceDetailByID struct {
	ProcessID           int64                   `json:"process_id"`
	ServiceInstanceName string                  `json:"service_instance_name"`
	Property            mapstr.MapStr           `json:"property"`
	Relation            ProcessInstanceRelation `json:"relation"`
}

type ProcessInstance struct {
	Property mapstr.MapStr           `json:"property"`
	Relation ProcessInstanceRelation `json:"relation"`
}

type ProcessInstanceNG struct {
	Process  Process                 `json:"process"`
	Relation ProcessInstanceRelation `json:"relation"`
}

type Proc2Module struct {
	BizID           int64  `json:"bk_biz_id"`
	ModuleName      string `json:"bk_module_name"`
	ProcessID       int64  `json:"bk_process_id"`
	SupplierAccount string `json:"bk_supplier_account"`
}

type LabelAggregationOption struct {
	BizID    int64  `json:"bk_biz_id"`
	ModuleID *int64 `json:"bk_module_id" bson:"bk_module_id" field:"bk_module_id"`
}

type SrvInstNameParams struct {
	ServiceInstanceID int64                  `json:"service_instance_id"`
	Host              map[string]interface{} `json:"host"`
	Process           *Process               `json:"process"`
}
