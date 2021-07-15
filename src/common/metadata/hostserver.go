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

package metadata

import (
	"context"
	"fmt"
	"time"

	"configcenter/src/common"
	"configcenter/src/common/blog"
	"configcenter/src/common/errors"
	"configcenter/src/common/mapstr"
	"configcenter/src/common/querybuilder"
	"configcenter/src/common/util"
)

type DeleteHostBatchOpt struct {
	HostID string `json:"bk_host_id"`
}

type HostInstanceProperties struct {
	PropertyID    string      `json:"bk_property_id"`
	PropertyName  string      `json:"bk_property_name"`
	PropertyValue interface{} `json:"bk_property_value"`
}

type HostInstancePropertiesResult struct {
	BaseResp `json:",inline"`
	Data     []HostInstanceProperties `json:"data"`
}

type HostSnapResult struct {
	BaseResp `json:",inline"`
	Data     map[string]interface{} `json:"data"`
}

type HostSnapBatchResult struct {
	BaseResp `json:",inline"`
	Data     []map[string]interface{} `json:"data"`
}

type HostInputType string

const (
	ExcelType   HostInputType = "excel"
	CollectType HostInputType = "collect"
)

type HostList struct {
	ApplicationID int64                            `json:"bk_biz_id"`
	ModuleID      int64                            `json:"bk_module_id"`
	HostInfo      map[int64]map[string]interface{} `json:"host_info"`
	InputType     HostInputType                    `json:"input_type"`
}

type AddHostToResourcePoolHostList struct {
	HostInfo  []map[string]interface{} `json:"host_info"`
	Directory int64                    `json:"directory"`
}

type AddHostToResourcePoolResult struct {
	Success []AddOneHostToResourcePoolResult `json:"success,omitempty"`
	Error   []AddOneHostToResourcePoolResult `json:"error,omitempty"`
}

type AddOneHostToResourcePoolResult struct {
	Index    int    `json:"index"`
	HostID   int64  `json:"bk_host_id,omitempty"`
	ErrorMsg string `json:"error_message,omitempty"`
}

type AddHostFromAgentHostList struct {
	HostInfo map[string]interface{} `json:"host_info"`
}

type HostSyncList struct {
	ApplicationID int64                            `json:"bk_biz_id"`
	HostInfo      map[int64]map[string]interface{} `json:"host_info"`
	ModuleID      []int64                          `json:"bk_module_id"`
	InputType     HostInputType                    `json:"input_type"`
}

type HostsModuleRelation struct {
	ApplicationID int64   `json:"bk_biz_id"`
	HostID        []int64 `json:"bk_host_id"`
	ModuleID      []int64 `json:"bk_module_id"`
	IsIncrement   bool    `json:"is_increment"`
}

type HostModuleConfig struct {
	ApplicationID int64   `json:"bk_biz_id" bson:"bk_biz_id"`
	HostID        []int64 `json:"bk_host_id" bson:"bk_host_id"`
	ModuleID      []int64 `json:"bk_module_id" bson:"bk_module_id"`
}

type RemoveHostsFromModuleOption struct {
	ApplicationID int64 `json:"bk_biz_id"`
	HostID        int64 `json:"bk_host_id"`
	ModuleID      int64 `json:"bk_module_id"`
}

type HostToAppModule struct {
	Ips         []string `json:"ips"`
	HostName    []string `json:"bk_host_name"`
	ModuleName  string   `json:"bk_module_name"`
	SetName     string   `json:"bk_set_name"`
	AppName     string   `json:"bk_biz_name"`
	OsType      string   `json:"bk_os_type"`
	OwnerID     string   `json:"bk_supplier_account"`
	PlatID      int64    `json:"bk_cloud_id"`
	IsIncrement bool     `json:"is_increment"`
}

type HostCommonSearch struct {
	AppID     int64             `json:"bk_biz_id,omitempty"`
	Ip        IPInfo            `json:"ip"`
	Condition []SearchCondition `json:"condition"`
	Page      BasePage          `json:"page"`
	Pattern   string            `json:"pattern,omitempty"`
}

type SetCommonSearch struct {
	AppID     int64             `json:"bk_biz_id,omitempty"`
	Condition []SearchCondition `json:"condition"`
	Page      BasePage          `json:"page"`
}

type FindHostsBySrvTplOpt struct {
	ServiceTemplateIDs []int64  `json:"bk_service_template_ids"`
	ModuleIDs          []int64  `json:"bk_module_ids"`
	Fields             []string `json:"fields"`
	Page               BasePage `json:"page"`
}

func (o *FindHostsBySrvTplOpt) Validate() (rawError errors.RawErrorInfo) {
	if len(o.ServiceTemplateIDs) == 0 || len(o.ServiceTemplateIDs) > common.BKMaxInstanceLimit {
		return errors.RawErrorInfo{
			ErrCode: common.CCErrArrayLengthWrong,
			Args:    []interface{}{"bk_service_template_ids", common.BKMaxInstanceLimit},
		}
	}

	if len(o.ModuleIDs) > common.BKMaxInstanceLimit {
		return errors.RawErrorInfo{
			ErrCode: common.CCErrArrayLengthWrong,
			Args:    []interface{}{"bk_module_ids", common.BKMaxInstanceLimit},
		}
	}

	if len(o.Fields) == 0 {
		return errors.RawErrorInfo{
			ErrCode: common.CCErrCommParamsInvalid,
			Args:    []interface{}{"fields"},
		}
	}

	if o.Page.IsIllegal() {
		return errors.RawErrorInfo{
			ErrCode: common.CCErrCommParamsInvalid,
			Args:    []interface{}{"page.limit"},
		}
	}

	return errors.RawErrorInfo{}
}

type FindHostsBySetTplOpt struct {
	SetTemplateIDs []int64  `json:"bk_set_template_ids"`
	SetIDs         []int64  `json:"bk_set_ids"`
	Fields         []string `json:"fields"`
	Page           BasePage `json:"page"`
}

func (o *FindHostsBySetTplOpt) Validate() (rawError errors.RawErrorInfo) {
	if len(o.SetTemplateIDs) == 0 || len(o.SetTemplateIDs) > common.BKMaxInstanceLimit {
		return errors.RawErrorInfo{
			ErrCode: common.CCErrArrayLengthWrong,
			Args:    []interface{}{"bk_set_template_ids", common.BKMaxInstanceLimit},
		}
	}

	if len(o.SetIDs) > common.BKMaxInstanceLimit {
		return errors.RawErrorInfo{
			ErrCode: common.CCErrArrayLengthWrong,
			Args:    []interface{}{"bk_set_ids", common.BKMaxInstanceLimit},
		}
	}

	if len(o.Fields) == 0 {
		return errors.RawErrorInfo{
			ErrCode: common.CCErrCommParamsInvalid,
			Args:    []interface{}{"fields"},
		}
	}

	if o.Page.IsIllegal() {
		return errors.RawErrorInfo{
			ErrCode: common.CCErrCommParamsInvalid,
			Args:    []interface{}{"page.limit"},
		}
	}

	return errors.RawErrorInfo{}
}

type FindHostsByTopoOpt struct {
	ObjID  string   `json:"bk_obj_id"`
	InstID int64    `json:"bk_inst_id"`
	Fields []string `json:"fields"`
	Page   BasePage `json:"page"`
}

func (o *FindHostsByTopoOpt) Validate() (rawError errors.RawErrorInfo) {
	if o.ObjID == "" {
		return errors.RawErrorInfo{
			ErrCode: common.CCErrCommParamsNeedSet,
			Args:    []interface{}{common.BKObjIDField},
		}
	}

	if o.ObjID == common.BKInnerObjIDApp {
		return errors.RawErrorInfo{
			ErrCode: common.CCErrCommParamsInvalid,
			Args:    []interface{}{common.BKObjIDField},
		}
	}

	if o.InstID <= 0 {
		return errors.RawErrorInfo{
			ErrCode: common.CCErrCommParamsInvalid,
			Args:    []interface{}{common.BKInstIDField},
		}
	}

	if len(o.Fields) == 0 {
		return errors.RawErrorInfo{
			ErrCode: common.CCErrCommParamsNeedSet,
			Args:    []interface{}{"fields"},
		}
	}

	if o.Page.IsIllegal() {
		return errors.RawErrorInfo{
			ErrCode: common.CCErrCommParamsInvalid,
			Args:    []interface{}{"page"},
		}
	}

	return errors.RawErrorInfo{}
}

type FindHostRelationWtihTopoOpt struct {
	Business int64    `json:"bk_biz_id"`
	ObjID    string   `json:"bk_obj_id"`
	InstIDs  []int64  `json:"bk_inst_ids"`
	Fields   []string `json:"fields"`
	Page     BasePage `json:"page"`
}

func (f *FindHostRelationWtihTopoOpt) Validate() *errors.RawErrorInfo {
	if f.ObjID == "" {
		return &errors.RawErrorInfo{
			ErrCode: common.CCErrCommParamsNeedSet,
			Args:    []interface{}{common.BKObjIDField},
		}
	}

	if f.ObjID == common.BKInnerObjIDApp {
		return &errors.RawErrorInfo{
			ErrCode: common.CCErrCommParamsInvalid,
			Args:    []interface{}{common.BKObjIDField},
		}
	}

	if f.Business <= 0 {
		return &errors.RawErrorInfo{
			ErrCode: common.CCErrCommParamsInvalid,
			Args:    []interface{}{common.BKAppIDField},
		}
	}

	if len(f.InstIDs) <= 0 || len(f.InstIDs) > 50 {
		return &errors.RawErrorInfo{
			ErrCode: common.CCErrCommParamsInvalid,
			Args:    []interface{}{common.BKInstIDField},
		}
	}

	if len(f.Fields) == 0 {
		return &errors.RawErrorInfo{
			ErrCode: common.CCErrCommParamsNeedSet,
			Args:    []interface{}{"fields"},
		}
	}

	if f.Page.Limit > common.BKMaxInstanceLimit {
		return &errors.RawErrorInfo{
			ErrCode: common.CCErrCommParamsInvalid,
			Args:    []interface{}{"page.limit"},
		}
	}

	if f.Page.IsIllegal() {
		return &errors.RawErrorInfo{
			ErrCode: common.CCErrCommParamsInvalid,
			Args:    []interface{}{"page"},
		}
	}

	return nil
}

type FindModuleHostRelationParameter struct {
	ModuleIDS    []int64  `json:"bk_module_ids"`
	ModuleFields []string `json:"module_fields"`
	HostFields   []string `json:"host_fields"`
	Page         BasePage `json:"page"`
}

func (param FindModuleHostRelationParameter) Validate() errors.RawErrorInfo {
	if len(param.ModuleIDS) == 0 {
		return errors.RawErrorInfo{
			ErrCode: common.CCErrCommParamsNeedSet,
			Args:    []interface{}{"bk_module_ids"},
		}
	}
	if len(param.ModuleIDS) > 200 {
		return errors.RawErrorInfo{
			ErrCode: common.CCErrCommXXExceedLimit,
			Args:    []interface{}{"bk_module_ids", 200},
		}
	}
	if param.Page.IsIllegal() {
		return errors.RawErrorInfo{
			ErrCode: common.CCErrCommParamsIsInvalid,
			Args:    []interface{}{"page"},
		}
	}
	if len(param.HostFields) == 0 {
		return errors.RawErrorInfo{
			ErrCode: common.CCErrCommParamsNeedSet,
			Args:    []interface{}{"host_fields"},
		}
	}
	if len(param.ModuleFields) == 0 {
		return errors.RawErrorInfo{
			ErrCode: common.CCErrCommParamsNeedSet,
			Args:    []interface{}{"module_fields"},
		}
	}
	return errors.RawErrorInfo{}
}

type ModuleHostRelation struct {
	Host    map[string]interface{}   `json:"host"`
	Modules []map[string]interface{} `json:"modules"`
}

type FindModuleHostRelationResult struct {
	Count    int                  `json:"count"`
	Relation []ModuleHostRelation `json:"relation"`
}

type FindModuleHostRelationResp struct {
	BaseResp `json:",inline"`
	Data     FindModuleHostRelationResult `json:"data"`
}

type ListHostsParameter struct {
	SetIDs             []int64                   `json:"bk_set_ids"`
	SetCond            []ConditionItem           `json:"set_cond"`
	ModuleIDs          []int64                   `json:"bk_module_ids"`
	ModuleCond         []ConditionItem           `json:"module_cond"`
	HostPropertyFilter *querybuilder.QueryFilter `json:"host_property_filter"`
	Fields             []string                  `json:"fields"`
	Page               BasePage                  `json:"page"`
}

func (option ListHostsParameter) Validate() (string, error) {
	if key, err := option.Page.Validate(false); err != nil {
		return fmt.Sprintf("page.%s", key), err
	}

	if option.HostPropertyFilter != nil {
		if key, err := option.HostPropertyFilter.Validate(); err != nil {
			return fmt.Sprintf("host_property_filter.%s", key), err
		}
		if option.HostPropertyFilter.GetDeep() > querybuilder.MaxDeep {
			return "host_property_filter.rules", fmt.Errorf("exceed max query condition deepth: %d", querybuilder.MaxDeep)
		}
	}

	if len(option.SetIDs) > 200 {
		return "bk_set_ids", fmt.Errorf("exceed max length: 200")
	}

	if len(option.ModuleIDs) > 500 {
		return "bk_module_ids", fmt.Errorf("exceed max length: 500")
	}

	return "", nil
}

type ListHostsWithNoBizParameter struct {
	HostPropertyFilter *querybuilder.QueryFilter `json:"host_property_filter"`
	Fields             []string                  `json:"fields"`
	Page               BasePage                  `json:"page"`
}

func (option ListHostsWithNoBizParameter) Validate() (string, error) {
	if key, err := option.Page.Validate(false); err != nil {
		return fmt.Sprintf("page.%s", key), err
	}

	if option.HostPropertyFilter != nil {
		if key, err := option.HostPropertyFilter.Validate(); err != nil {
			return fmt.Sprintf("host_property_filter.%s", key), err
		}
		if option.HostPropertyFilter.GetDeep() > querybuilder.MaxDeep {
			return "host_property_filter.rules", fmt.Errorf("exceed max query condition deepth: %d", querybuilder.MaxDeep)
		}
	}

	return "", nil
}

type ListBizHostsTopoParameter struct {
	SetPropertyFilter    *querybuilder.QueryFilter `json:"set_property_filter"`
	ModulePropertyFilter *querybuilder.QueryFilter `json:"module_property_filter"`
	HostPropertyFilter   *querybuilder.QueryFilter `json:"host_property_filter"`
	Fields               []string                  `json:"fields"`
	Page                 BasePage                  `json:"page"`
}

func (option ListBizHostsTopoParameter) Validate(errProxy errors.DefaultCCErrorIf) errors.CCErrorCoder {
	if option.Page.Limit == 0 {
		return errProxy.CCErrorf(common.CCErrCommParamsNeedSet, "page.limit")
	}
	if option.Page.Limit > common.BKMaxInstanceLimit {
		return errProxy.CCErrorf(common.CCErrCommXXExceedLimit, "page.limit", common.BKMaxInstanceLimit)
	}

	if option.HostPropertyFilter != nil {
		if key, err := option.HostPropertyFilter.Validate(); err != nil {
			blog.Errorf("valid host property filter failed, err: %v", err)
			return errProxy.CCErrorf(common.CCErrCommParamsInvalid, fmt.Sprintf("host_property_filter.%s", key))
		}
		if option.HostPropertyFilter.GetDeep() > querybuilder.MaxDeep {
			return errProxy.CCErrorf(common.CCErrCommXXExceedLimit, "host_property_filter.rules", querybuilder.MaxDeep)
		}
	}

	if option.SetPropertyFilter != nil {
		if key, err := option.SetPropertyFilter.Validate(); err != nil {
			blog.Errorf("valid set property filter failed, err: %v", err)
			return errProxy.CCErrorf(common.CCErrCommParamsInvalid, fmt.Sprintf("set_property_filter.%s", key))
		}
		if option.SetPropertyFilter.GetDeep() > querybuilder.MaxDeep {
			return errProxy.CCErrorf(common.CCErrCommXXExceedLimit, "set_property_filter.rules", querybuilder.MaxDeep)
		}
	}

	if option.ModulePropertyFilter != nil {
		if key, err := option.ModulePropertyFilter.Validate(); err != nil {
			blog.Errorf("valid module property filter failed, err: %v", err)
			return errProxy.CCErrorf(common.CCErrCommParamsInvalid, fmt.Sprintf("module_property_filter.%s", key))
		}
		if option.ModulePropertyFilter.GetDeep() > querybuilder.MaxDeep {
			return errProxy.CCErrorf(common.CCErrCommXXExceedLimit, "module_property_filter.rules", querybuilder.MaxDeep)
		}
	}

	return nil
}

type ListHostsDetailAndTopoOption struct {
	// return the host's topology with biz info.
	WithBiz            bool                      `json:"with_biz"`
	HostPropertyFilter *querybuilder.QueryFilter `json:"host_property_filter"`
	Fields             []string                  `json:"fields"`
	Page               BasePage                  `json:"page"`
}

func (option *ListHostsDetailAndTopoOption) Validate() *errors.RawErrorInfo {

	if key, err := option.Page.Validate(false); err != nil {
		return &errors.RawErrorInfo{
			ErrCode: common.CCErrCommParamsInvalid,
			Args:    []interface{}{"page." + key},
		}
	}

	if option.Page.Limit > common.BKMaxInstanceLimit {
		return &errors.RawErrorInfo{
			ErrCode: common.CCErrCommParamsInvalid,
			Args:    []interface{}{"page." + "limit"},
		}
	}

	if option.HostPropertyFilter != nil {
		if key, err := option.HostPropertyFilter.Validate(); err != nil {
			return &errors.RawErrorInfo{
				ErrCode: common.CCErrCommParamsInvalid,
				Args:    []interface{}{"host_property_filter." + key},
			}
		}
		if option.HostPropertyFilter.GetDeep() > querybuilder.MaxDeep {
			return &errors.RawErrorInfo{
				ErrCode: common.CCErrCommParamsInvalid,
				Args:    []interface{}{"host_property_filter exceeded max allowed deep"},
			}
		}
	}

	if len(option.Fields) == 0 {
		return &errors.RawErrorInfo{
			ErrCode: common.CCErrCommParamsInvalid,
			Args:    []interface{}{"fields"},
		}
	}

	return nil
}

type CountTopoNodeHostsOption struct {
	Nodes []TopoNode `json:"topo_nodes" mapstructure:"topo_nodes"`
}

type TimeRange struct {
	Start *time.Time
	End   *time.Time
}

type ListHosts struct {
	BizID              int64                     `json:"bk_biz_id,omitempty"`
	SetIDs             []int64                   `json:"bk_set_ids"`
	ModuleIDs          []int64                   `json:"bk_module_ids"`
	HostPropertyFilter *querybuilder.QueryFilter `json:"host_property_filter"`
	Fields             []string                  `json:"fields"`
	Page               BasePage                  `json:"page"`
}

// Validate whether ListHosts is valid
// errKey: invalid key
// er: detail reason why errKey in invalid
func (option ListHosts) Validate() (errKey string, err error) {
	if key, err := option.Page.Validate(false); err != nil {
		return fmt.Sprintf("page.%s", key), err
	}

	if option.HostPropertyFilter != nil {
		if key, err := option.HostPropertyFilter.Validate(); err != nil {
			return fmt.Sprintf("host_property_filter.%s", key), err
		}
		if option.HostPropertyFilter.GetDeep() > querybuilder.MaxDeep {
			return "host_property_filter.rules", fmt.Errorf("exceed max query condition deepth: %d", querybuilder.MaxDeep)
		}
	}

	return "", nil
}

func (option ListHosts) GetHostPropertyFilter(ctx context.Context) (map[string]interface{}, error) {
	if option.HostPropertyFilter != nil {
		mgoFilter, key, err := option.HostPropertyFilter.ToMgo()
		if err != nil {
			return nil, fmt.Errorf("invalid key:host_property_filter.%s, err: %s", key, err)
		}
		return mgoFilter, nil
	}
	return make(map[string]interface{}), nil
}

// ip search info
type IPInfo struct {
	Data  []string `json:"data"`
	Exact int64    `json:"exact"`
	Flag  string   `json:"flag"`
}

// search condition
type SearchCondition struct {
	Fields    []string        `json:"fields"`
	Condition []ConditionItem `json:"condition"`
	ObjectID  string          `json:"bk_obj_id"`
	// 非必填，只能用来查时间，且与Condition是与关系
	TimeCondition *TimeCondition `json:"time_condition,omitempty"`
}

type SearchHost struct {
	Count int             `json:"count"`
	Info  []mapstr.MapStr `json:"info"`
}

type ListHostResult struct {
	Count int                      `json:"count"`
	Info  []map[string]interface{} `json:"info"`
}

type HostTopoResult struct {
	Count int        `json:"count"`
	Info  []HostTopo `json:"info"`
}

type HostTopo struct {
	Host map[string]interface{} `json:"host"`
	Topo []Topo                 `json:"topo"`
}

type Topo struct {
	SetID   int64    `json:"bk_set_id" bson:"bk_set_id"`
	SetName string   `json:"bk_set_name" bson:"bk_set_name"`
	Module  []Module `json:"module" bson:"module"`
}

type HostDetailWithTopo struct {
	Host map[string]interface{} `json:"host"`
	Topo []*HostTopoNode        `json:"topo"`
}

type HostTopoNode struct {
	Instance *NodeInstance   `json:"inst"`
	Children []*HostTopoNode `json:"children"`
}

type NodeInstance struct {
	Object   string      `json:"obj"`
	InstName interface{} `json:"name"`
	InstID   interface{} `json:"id"`
}

type Module struct {
	ModuleID   int64  `json:"bk_module_id" bson:"bk_module_id" mapstructure:"bk_module_id"`
	ModuleName string `json:"bk_module_name" bson:"bk_module_name" mapstructure:"bk_module_name"`
}

func (sh SearchHost) ExtractHostIDs() *[]int64 {
	hostIDArray := make([]int64, 0)
	for _, h := range sh.Info {
		if _, exist := h["host"]; exist == false {
			blog.ErrorJSON("unexpected error, host: %s don't have host field.", h)
			continue
		}
		hostID, exist := h["host"].(mapstr.MapStr)[common.BKHostIDField]
		if exist == false {
			blog.ErrorJSON("unexpected error, host: %s don't have host.bk_host_id field.", h)
			continue
		}
		id, err := util.GetInt64ByInterface(hostID)
		if err != nil {
			blog.ErrorJSON("unexpected error, host: %s host.bk_host_id field is not integer.", h)
			continue
		}
		hostIDArray = append(hostIDArray, id)
	}
	return &hostIDArray
}

type SearchHostResult struct {
	BaseResp `json:",inline"`
	Data     *SearchHost `json:"data"`
}

type HostCloneInputParams struct {
	OrgIP  string `json:"bk_org_ip"`
	DstIP  string `json:"bk_dst_ip"`
	AppID  int64  `json:"bk_biz_id"`
	PlatID int64  `json:"bk_cloud_id"`
}

type SetHostConfigParams struct {
	ApplicationID int64 `json:"bk_biz_id"`
	SetID         int64 `json:"bk_set_id"`
	ModuleID      int64 `json:"bk_module_id"`
}

type CloneHostPropertyParams struct {
	AppID int64 `json:"bk_biz_id"`
	// source and destination host inner ip
	OrgIP string `json:"bk_org_ip"`
	DstIP string `json:"bk_dst_ip"`
	// source and destination host id
	OrgID   int64 `json:"bk_org_id"`
	DstID   int64 `json:"bk_dst_id"`
	CloudID int64 `json:"bk_cloud_id"`
}

// TransferHostAcrossBusinessParameter Transfer host across business request parameter
type TransferHostAcrossBusinessParameter struct {
	SrcAppID    int64   `json:"src_bk_biz_id"`
	DstAppID    int64   `json:"dst_bk_biz_id"`
	HostID      []int64 `json:"bk_host_id"`
	DstModuleID int64   `json:"bk_module_id"`
}

// HostModuleRelationParameter get host and module  relation parameter
type HostModuleRelationParameter struct {
	AppID  int64   `json:"bk_biz_id"`
	HostID []int64 `json:"bk_host_id"`
}

func (h *HostModuleRelationParameter) Validate() (rawError errors.RawErrorInfo) {
	if len(h.HostID) == 0 || len(h.HostID) > common.BKMaxInstanceLimit {
		return errors.RawErrorInfo{
			ErrCode: common.CCErrArrayLengthWrong,
			Args:    []interface{}{"bk_host_id", common.BKMaxInstanceLimit},
		}
	}

	return errors.RawErrorInfo{}
}

// DeleteHostFromBizParameter delete host from business
type DeleteHostFromBizParameter struct {
	AppID     int64   `json:"bk_biz_id"`
	HostIDArr []int64 `json:"bk_host_ids"`
}

// CloudAreaParameter search cloud area parameter
type CloudAreaSearchParam struct {
	SearchCloudOption `json:",inline"`
	SyncTaskIDs       bool `json:"sync_task_ids"`
}

// CloudAreaHostCount cloud area host count param
type CloudAreaHostCount struct {
	CloudIDs []int64 `json:"bk_cloud_ids"`
}

func (c *CloudAreaHostCount) Validate() (rawError errors.RawErrorInfo) {
	maxLimit := 50
	if len(c.CloudIDs) == 0 || len(c.CloudIDs) > maxLimit {
		return errors.RawErrorInfo{
			ErrCode: common.CCErrArrayLengthWrong,
			Args:    []interface{}{"bk_cloud_ids", maxLimit},
		}
	}

	return errors.RawErrorInfo{}
}

type CloudAreaHostCountResult struct {
	BaseResp `json:",inline"`
	Data     []CloudAreaHostCountElem `json:"data"`
}

type CloudAreaHostCountElem struct {
	CloudID   int64 `json:"bk_cloud_id"`
	HostCount int64 `json:"host_count"`
}

type CreateManyCloudAreaResult struct {
	BaseResp `json:",inline"`
	Data     []CreateManyCloudAreaElem `json:"data"`
}

type CreateManyCloudAreaElem struct {
	CloudID int64  `json:"bk_cloud_id"`
	ErrMsg  string `json:"err_msg"`
}

type TopoNode struct {
	ObjectID   string `field:"bk_obj_id" json:"bk_obj_id" mapstructure:"bk_obj_id"`
	InstanceID int64  `field:"bk_inst_id" json:"bk_inst_id" mapstructure:"bk_inst_id"`
}

func (node TopoNode) Key() string {
	return fmt.Sprintf("%s:%d", node.ObjectID, node.InstanceID)
}

func (node TopoNode) String() string {
	return fmt.Sprintf("%s:%d", node.ObjectID, node.InstanceID)
}

type TopoNodeHostCount struct {
	Node      TopoNode `field:"topo_node" json:"topo_node" mapstructure:"topo_node"`
	HostCount int      `field:"host_count" json:"host_count" mapstructure:"host_count"`
}

type TransferHostWithAutoClearServiceInstanceOption struct {
	HostIDs []int64 `field:"bk_host_ids" json:"bk_host_ids"`

	RemoveFromNode *TopoNode `field:"remove_from_node" json:"remove_from_node"`
	AddToModules   []int64   `field:"add_to_modules" json:"add_to_modules"`
	// 主机从 RemoveFromNode 移除后如果不再属于其它模块， 默认转移到空闲机模块
	// DefaultInternalModule 支持调整这种模型行为，可设置成待回收模块或者故障机模块
	DefaultInternalModule int64 `field:"default_internal_module" json:"default_internal_module"`

	Options TransferOptions `field:"options" json:"options"`
}

type TransferOptions struct {
	ServiceInstanceOptions     []CreateServiceInstanceOption `field:"service_instance_options" json:"service_instance_options"`
	HostApplyConflictResolvers []HostApplyConflictResolver   `field:"host_apply_conflict_resolvers" json:"host_apply_conflict_resolvers" bson:"host_apply_conflict_resolvers" mapstructure:"host_apply_conflict_resolvers"`
}

type HostTransferPlan struct {
	HostID                  int64            `field:"bk_host_id" json:"bk_host_id"`
	FinalModules            []int64          `field:"final_modules" json:"final_modules"`
	ToRemoveFromModules     []int64          `field:"to_remove_from_modules" json:"to_remove_from_modules"`
	ToAddToModules          []int64          `field:"to_add_to_modules" json:"to_add_to_modules"`
	IsTransferToInnerModule bool             `field:"is_transfer_to_inner_module" json:"is_transfer_to_inner_module"`
	HostApplyPlan           OneHostApplyPlan `field:"host_apply_plan" json:"host_apply_plan" mapstructure:"host_apply_plan"`
}

type RemoveFromModuleInfo struct {
	ModuleID         int64             `field:"bk_module_id" json:"bk_module_id"`
	ServiceInstances []ServiceInstance `field:"service_instances" json:"service_instances"`
}

type AddToModuleInfo struct {
	ModuleID        int64                  `field:"bk_module_id" json:"bk_module_id"`
	ServiceTemplate *ServiceTemplateDetail `field:"service_template" json:"service_template"`
}

type HostTransferPreview struct {
	HostID              int64                  `field:"bk_host_id" json:"bk_host_id"`
	FinalModules        []int64                `field:"final_modules" json:"final_modules"`
	ToRemoveFromModules []RemoveFromModuleInfo `field:"to_remove_from_modules" json:"to_remove_from_modules"`
	ToAddToModules      []AddToModuleInfo      `field:"to_add_to_modules" json:"to_add_to_modules"`
	HostApplyPlan       OneHostApplyPlan       `field:"host_apply_plan" json:"host_apply_plan"`
}

type UpdateHostCloudAreaFieldOption struct {
	BizID   int64   `field:"bk_biz_id" json:"bk_biz_id" mapstructure:"bk_biz_id"`
	HostIDs []int64 `field:"bk_host_ids" json:"bk_host_ids" mapstructure:"bk_host_ids"`
	CloudID int64   `field:"bk_cloud_id" json:"bk_cloud_id" mapstructure:"bk_cloud_id"`
}

// UpdateHostPropertyBatchParameter batch update host property parameter
type UpdateHostPropertyBatchParameter struct {
	Update []updateHostProperty `json:"update"`
}

type updateHostProperty struct {
	HostID     int64                  `json:"bk_host_id"`
	Properties map[string]interface{} `json:"properties"`
}
