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

package common

import (
	"math"
	"time"
)

const (
	// HTTPCreate create method
	HTTPCreate = "POST"

	// HTTPSelectPost select method
	HTTPSelectPost = "POST"

	// HTTPSelectGet select method
	HTTPSelectGet = "GET"

	// HTTPUpdate update method
	HTTPUpdate = "PUT"

	// HTTPDelete delete method
	HTTPDelete = "DELETE"

	// BKNoLimit no limit definition
	BKNoLimit = 999999999
	// max limit of a page
	BKMaxPageSize = 1000

	// max limit of instance count
	BKMaxInstanceLimit = 500

	// 一次最大操作记录数
	BKMaxRecordsAtOnce = 2000

	// BKDefaultLimit the default limit definition
	BKDefaultLimit = 20

	// BKAuditLogPageLimit the audit log page limit
	BKAuditLogPageLimit = 200

	// BKMaxExportRecord the limit to export
	BKMaxExportLimit = 10000

	// BKInstParentStr the inst parent name
	BKInstParentStr = "bk_parent_id"

	// BKDefaultOwnerID the default owner value
	BKDefaultOwnerID = "0"

	// BKSuperOwnerID the super owner value
	BKSuperOwnerID = "superadmin"

	// BKDefaultDirSubArea the default dir subarea
	BKDefaultDirSubArea = 0

	// BKTimeTypeParseFlag the time flag
	BKTimeTypeParseFlag = "cc_time_type"

	// BKTopoBusinessLevelLimit the mainline topo level limit
	BKTopoBusinessLevelLimit = "level.businessTopoMax"

	// BKTopoBusinessLevelDefault the mainline topo level default level
	BKTopoBusinessLevelDefault = 7

	BKParamMaxLength = 1000
)

const (
	// BKInnerObjIDApp the inner object
	BKInnerObjIDApp = "biz"

	// BKInnerObjIDSet the inner object
	BKInnerObjIDSet = "set"

	// BKInnerObjIDModule the inner object
	BKInnerObjIDModule = "module"

	// BKInnerObjIDHost the inner object
	BKInnerObjIDHost = "host"

	// BKInnerObjIDObject the inner object
	BKInnerObjIDObject = "object"

	// BKInnerObjIDProc the inner object
	BKInnerObjIDProc = "process"

	// BKInnerObjIDConfigTemp the inner object
	BKInnerObjIDConfigTemp = "config_template"

	// BKInnerObjIDTempVerion the inner object
	BKInnerObjIDTempVersion = "template_version"

	// BKInnerObjIDPlat the inner object
	BKInnerObjIDPlat = "plat"

	// BKInnerObjIDSwitch the inner object
	BKInnerObjIDSwitch = "bk_switch"
	// BKInnerObjIDRouter the inner object
	BKInnerObjIDRouter = "bk_router"
	// BKInnerObjIDBlance the inner object
	BKInnerObjIDBlance = "bk_load_balance"
	// BKInnerObjIDFirewall the inner object
	BKInnerObjIDFirewall = "bk_firewall"
	// BKInnerObjIDWeblogic the inner object
	BKInnerObjIDWeblogic = "bk_weblogic"
	// BKInnerObjIDTomcat the inner object
	BKInnerObjIDTomcat = "bk_tomcat"
	// BKInnerObjIDApache the inner object
	BKInnerObjIDApache = "bk_apache"
)

// Revision
const (
	RevisionEnterprise = "enterprise"
	RevisionCommunity  = "community"
	RevisionOpensource = "opensource"
)

const (
	// used only for host search
	BKDBMULTIPLELike = "$multilike"

	// BKDBIN the db operator
	BKDBIN = "$in"

	// BKDBOR the db operator
	BKDBOR = "$or"

	// BKDBAND the db operator
	BKDBAND = "$and"

	// BKDBLIKE the db operator
	BKDBLIKE = "$regex"

	// BKDBOPTIONS the db operator,used with $regex
	// detail to see https://docs.mongodb.com/manual/reference/operator/query/regex/#op._S_options
	BKDBOPTIONS = "$options"

	// BKDBEQ the db operator
	BKDBEQ = "$eq"

	// BKDBNE the db operator
	BKDBNE = "$ne"

	// BKDBNIN the db oeprator
	BKDBNIN = "$nin"

	// BKDBLT the db operator
	BKDBLT = "$lt"

	// BKDBLTE the db operator
	BKDBLTE = "$lte"

	// BKDBGT the db operator
	BKDBGT = "$gt"

	// BKDBGTE the db opeartor
	BKDBGTE = "$gte"

	// BKDBExists the db opeartor
	BKDBExists = "$exists"

	// BKDBNot the db opeartor
	BKDBNot = "$not"

	// BKDBCount the db opeartor
	BKDBCount = "$count"

	// BKDBGroup the db opeartor
	BKDBGroup = "$group"

	// BKDBMatch the db opeartor
	BKDBMatch = "$match"

	// BKDBSum the db opeartor
	BKDBSum = "$sum"

	// BKDBPush the db opeartor
	BKDBPush = "$push"

	// BKDBUNSET the db opeartor
	BKDBUNSET = "$unset"

	// BKDBAddToSet The $addToSet operator adds a value to an array unless the value is already present, in which case $addToSet does nothing to that array.
	BKDBAddToSet = "$addToSet"

	// BKDBPull The $pull operator removes from an existing array all instances of a value or values that match a specified condition.
	BKDBPull = "$pull"

	// BKDBAll matches arrays that contain all elements specified in the query.
	BKDBAll = "$all"

	// BKDBProject passes along the documents with the requested fields to the next stage in the pipeline
	BKDBProject = "$project"

	// BKDBSize counts and returns the total number of items in an array
	BKDBSize = "$size"

	// BKDBSort the db operator
	BKDBSort = "$sort"

	// BKDBReplaceRoot the db operator
	BKDBReplaceRoot = "$replaceRoot"
)

const (
	// DefaultResModuleName the default idle module name
	DefaultResModuleName string = "空闲机"
	// DefaultFaultModuleName the default fault module name
	DefaultFaultModuleName string = "故障机"
	// DefaultRecycleModuleName the default fault module name
	DefaultRecycleModuleName string = "待回收"
)

const (
	// BKFieldID the id definition
	BKFieldID   = "id"
	BKFieldName = "name"

	// BKDefaultField the default field
	BKDefaultField = "default"

	// BKOwnerIDField the owner field
	BKOwnerIDField = "bk_supplier_account"

	// BKAppIDField the appid field
	BKAppIDField = "bk_biz_id"

	// BKIPArr the ip address
	BKIPArr = "ipArr"

	// BKAssetIDField  the asset id field
	BKAssetIDField = "bk_asset_id"

	// BKSNField  the sn  field
	BKSNField = "bk_sn"

	// BKHostInnerIPField the host innerip field
	BKHostInnerIPField = "bk_host_innerip"

	// BKHostCloudRegionField the host cloud region field
	BKHostCloudRegionField = "bk_cloud_region"

	// BKHostOuterIPField the host outerip field
	BKHostOuterIPField = "bk_host_outerip"

	// BKCloudInstIDField the cloud instance id field
	BKCloudInstIDField = "bk_cloud_inst_id"

	// BKCloudHostStatusField the cloud host status field
	BKCloudHostStatusField = "bk_cloud_host_status"

	// TimeTransferModel the time transferModel field
	TimeTransferModel = "2006-01-02 15:04:05"

	// BKCloudTaskID the cloud sync task id
	BKCloudTaskID = "bk_task_id"

	// BKNewAddHost the cloud sync new add hosts
	BKNewAddHost = "new_add"

	// BKImportFrom the host import from field
	BKImportFrom = "import_from"

	// BKHostIDField the host id field
	BKHostIDField = "bk_host_id"

	// BKHostNameField the host name field
	BKHostNameField = "bk_host_name"

	// BKAppNameField the app name field
	BKAppNameField = "bk_biz_name"

	// BKSetIDField the setid field
	BKSetIDField = "bk_set_id"

	// BKSetNameField the set name field
	BKSetNameField = "bk_set_name"

	// BKModuleIDField the module id field
	BKModuleIDField = "bk_module_id"

	// BKModuleNameField the module name field
	BKModuleNameField = "bk_module_name"

	HostApplyEnabledField = "host_apply_enabled"

	// BKSubscriptionIDField the subscription id field
	BKSubscriptionIDField = "subscription_id"
	// BKSubscriptionNameField the subscription name field
	BKSubscriptionNameField = "subscription_name"

	// BKOSTypeField the os type field
	BKOSTypeField = "bk_os_type"

	// BKOSNameField the os name field
	BKOSNameField = "bk_os_name"

	// BKHttpGet the http get
	BKHttpGet = "GET"

	// BKTencentCloudTimeOut the tencent cloud timeout
	BKTencentCloudTimeOut = 10

	// TencentCloudUrl the tencent cloud url
	TencentCloudUrl = "cvm.tencentcloudapi.com"

	// TencentCloudSignMethod the tencent cloud sign method
	TencentCloudSignMethod = "HmacSHA1"

	// BKCloudIDField the cloud id field
	BKCloudIDField = "bk_cloud_id"

	// BKCloudNameField the cloud name field
	BKCloudNameField = "bk_cloud_name"

	// BKObjIDField the obj id field
	BKObjIDField = "bk_obj_id"

	// BKObjNameField the obj name field
	BKObjNameField = "bk_obj_name"

	// BKObjIconField the obj icon field
	BKObjIconField = "bk_obj_icon"

	// BKInstIDField the inst id field
	BKInstIDField = "bk_inst_id"

	// BKInstNameField the inst name field
	BKInstNameField = "bk_inst_name"

	// ExportCustomFields the use custom display columns
	ExportCustomFields = "export_custom_fields"

	// BKProcIDField the proc id field
	BKProcIDField = "bk_process_id"

	// BKConfTempId is the config template id field
	BKConfTempIdField = "bk_conftemp_id"

	// BKProcNameField the proc name field
	BKProcNameField = "bk_process_name"

	// BKTemlateIDField the process template id field
	BKTemlateIDField = "template_id"

	// BKVesionIDField the version id field
	BKVersionIDField = "version_id"

	// BKTemplateNameField the template name field
	BKTemplateNameField = "template_name"

	// BKFileNameField the file name field
	BKFileNameField = "file_name"

	// BKPropertyIDField the propety id field
	BKPropertyIDField = "bk_property_id"

	// BKPropertyNameField the property name field
	BKPropertyNameField = "bk_property_name"

	BKPropertyIndexField = "bk_property_index"

	// BKPropertyTypeField the property type field
	BKPropertyTypeField = "bk_property_type"

	BKPropertyGroupField = "bk_property_group"

	// BKPropertyValueField the property value field
	BKPropertyValueField = "bk_property_value"

	// BKObjAttIDField the obj att id field
	BKObjAttIDField = "bk_object_att_id"

	// BKClassificationIDField the classification id field
	BKClassificationIDField = "bk_classification_id"

	// BKClassificationNameField the classification name field
	BKClassificationNameField = "bk_classification_name"

	// BKClassificationIconField the classification icon field
	BKClassificationIconField = "bk_classification_icon"

	// BKPropertyGroupIDField the property group id field
	BKPropertyGroupIDField = "bk_group_id"

	// BKPropertyGroupNameField the property group name field
	BKPropertyGroupNameField = "bk_group_name"

	// BKPropertyGroupIndexField the property group index field
	BKPropertyGroupIndexField = "bk_group_index"

	// BKAsstObjIDField the property obj id field
	BKAsstObjIDField = "bk_asst_obj_id"

	// BKAsstInstIDField the property inst id field
	BKAsstInstIDField = "bk_asst_inst_id"

	// BKOptionField the option field
	BKOptionField = "option"

	// BKAuditTypeField the audit type field
	BKAuditTypeField = "audit_type"

	// BKResourceTypeField the audit resource type field
	BKResourceTypeField = "resource_type"

	// BKOperateFromField the platform where operation from field
	BKOperateFromField = "operate_from"

	// BKOperationDetailField the audit operation detail field
	BKOperationDetailField = "operation_detail"

	// BKOperationTimeField the audit operation time field
	BKOperationTimeField = "operation_time"

	// BKResourceIDField the audit resource ID field
	BKResourceIDField = "resource_id"

	// BKResourceNameField the audit resource name field
	BKResourceNameField = "resource_name"

	// BKLabelField the audit resource name field
	BKLabelField = "label"

	// BKSetEnvField the set env field
	BKSetEnvField = "bk_set_env"

	// BKSetStatusField the set status field
	BKSetStatusField = "bk_service_status"

	// BKSetDescField the set desc field
	BKSetDescField = "bk_set_desc"

	// BKSetCapacityField the set capacity field
	BKSetCapacityField = "bk_capacity"

	// BKPort the port
	BKPort = "port"

	// BKProcPortEnable whether enable port,  enable port use for monitor app. default value
	BKProcPortEnable = "bk_enable_port"

	// BKProcGatewayIP the process gateway ip
	BKProcGatewayIP = "bk_gateway_ip"

	// BKProcGatewayPort the process gateway port
	BKProcGatewayPort = "bk_gateway_port"

	// BKProcGatewayProtocol the process gateway protocol
	BKProcGatewayProtocol = "bk_gateway_protocol"

	// BKProcGatewayCity the process gateway city
	BKProcGatewayCity = "bk_gateway_city"

	// BKProcBindInfo the process bind info
	BKProcBindInfo = "bind_info"

	// BKUser the user
	BKUser = "user"

	// BKProtocol the protocol
	BKProtocol = "protocol"

	// BKIP the ip
	BKIP = "ip"

	// BKEnable the enable
	BKEnable = "enable"

	// the process object name
	BKProcessObjectName = "process"

	// BKProcessIDField the process id field
	BKProcessIDField = "bk_process_id"

	BKServiceInstanceIDField = "service_instance_id"
	BKServiceTemplateIDField = "service_template_id"
	BKProcessTemplateIDField = "process_template_id"
	BKServiceCategoryIDField = "service_category_id"

	BKSetTemplateIDField = "set_template_id"

	HostApplyRuleIDField = "host_apply_rule_id"

	BKParentIDField = "bk_parent_id"
	BKRootIDField   = "bk_root_id"

	// BKProcessNameField the process name field
	BKProcessNameField = "bk_process_name"

	// BKFuncIDField the func id field
	BKFuncIDField = "bk_func_id"

	// BKFuncName the function name
	BKFuncName = "bk_func_name"

	BKStartParamRegex = "bk_start_param_regex"

	// BKBindIP the bind ip
	BKBindIP = "bind_ip"

	// BKWorkPath the work path
	BKWorkPath = "work_path"

	// BKIsPre the ispre field
	BKIsPre = "ispre"

	// BKIsIncrementField the isincrement field
	BKIsIncrementField = "is_increment"

	BKIsCollapseField = "is_collapse"

	// BKProxyListField the proxy list field
	BKProxyListField = "bk_proxy_list"

	// BKIPListField the ip list field
	BKIPListField = "ip_list"

	// BKInvalidIPSField the invalid ips field
	BKInvalidIPSField = "invalid_ips"

	// BKGseProxyField the gse proxy
	BKGseProxyField = "bk_gse_proxy"

	// BKSubAreaField the sub area field
	BKSubAreaField = "bk_cloud_id"

	// BKProcField the proc field
	BKProcField = "bk_process"

	// BKMaintainersField the maintainers field
	BKMaintainersField = "bk_biz_maintainer"

	// BKProductPMField the product pm field
	BKProductPMField = "bk_biz_productor"

	// BKTesterField the tester field
	BKTesterField = "bk_biz_tester"

	// BKOperatorField the operator field
	BKOperatorField = "operator" // the operator of app of module, is means a job position

	// BKLifeCycleField the life cycle field
	BKLifeCycleField = "life_cycle"

	// BKDeveloperField the developer field
	BKDeveloperField = "bk_biz_developer"

	// BKLanguageField the language field
	BKLanguageField = "language"

	// BKBakOperatorField the bak operator field
	BKBakOperatorField = "bk_bak_operator"

	// BKTimeZoneField the time zone field
	BKTimeZoneField = "time_zone"

	// BKIsRequiredField the required field
	BKIsRequiredField = "isrequired"

	// BKModuleTypeField the module type field
	BKModuleTypeField = "bk_module_type"

	// BKOrgIPField the org ip field
	BKOrgIPField = "bk_org_ip"

	// BKDstIPField the dst ip field
	BKDstIPField = "bk_dst_ip"

	// BKDescriptionField the description field
	BKDescriptionField = "description"

	// BKIsOnlyField the isonly name field
	BKIsOnlyField = "isonly"
	// BKGseTaskIdField the gse taskid
	BKGseTaskIDField = "task_id"
	// BKTaskIdField the gse taskid
	BKTaskIDField = "task_id"
	// BKGseOpTaskIDField the gse taskid
	BKGseOpTaskIDField = "gse_task_id"
	BKProcPidFile      = "pid_file"
	BKProcStartCmd     = "start_cmd"
	BKProcStopCmd      = "stop_cmd"
	BKProcReloadCmd    = "reload_cmd"
	BKProcRestartCmd   = "restart_cmd"
	BKProcTimeOut      = "timeout"
	BKProcWorkPath     = "work_path"
	BKProcInstNum      = "proc_num"

	// BKInstKeyField the inst key field for metric discover
	BKInstKeyField = "bk_inst_key"

	// for net collect device
	BKDeviceIDField    = "device_id"
	BKDeviceNameField  = "device_name"
	BKDeviceModelField = "device_model"
	BKVendorField      = "bk_vendor"

	// for net collect property of device
	BKNetcollectPropertyIDField = "netcollect_property_id"
	BKOIDField                  = "oid"
	BKPeriodField               = "period"
	BKActionField               = "action"
	BKProcinstanceID            = "proc_instance_id"

	// BKGseOpProcTaskDetailField gse operate process return detail
	BKGseOpProcTaskDetailField = "detail"
	BKGroupField               = "group"

	BKAttributeIDField = "bk_attribute_id"

	BKSubscribeID = "subscribeID"

	BKTokenField       = "token"
	BKCursorField      = "cursor"
	BKClusterTimeField = "cluster_time"
	BKEventTypeField   = "type"
	BKStartAtTimeField = "start_at_time"
)

const (
	BKRequestField = "bk_request_id"
	BKTxnIDField   = "bk_txn_id"
)

// DefaultResSetName the inner module set
const DefaultResSetName string = "空闲机池"

// WhiteListAppName the white list app name
const WhiteListAppName = "蓝鲸"

// WhiteListSetName the white list set name
const WhiteListSetName = "公共组件"

// WhiteListModuleName the white list module name
const WhiteListModuleName = "gitserver"

// the inst record's logging information
const (
	// CreatorField the creator
	CreatorField = "creator"

	// CreateTimeField the create time field
	CreateTimeField = "create_time"

	// ConfirmTimeField the cloud resource confirm time filed
	ConfirmTimeField = "confirm_time"

	// StartTimeField the cloud sync start time field
	StartTimeFiled = "start_time"

	// ModifierField the modifier field
	ModifierField = "modifier"

	// LastTimeField the last time field
	LastTimeField = "last_time"
)

const (
	// ValidCreate valid create
	ValidCreate = "create"

	// ValidUpdate valid update
	ValidUpdate = "update"
)

// DefaultResSetFlag the default resource set flat
const DefaultResSetFlag int = 1

const DefaultFlagDefaultValue int = 0

// DefaultAppFlag the default app flag
const DefaultAppFlag int = 1

// DefaultAppName the default app name
const DefaultAppName string = "资源池"

const DefaultCloudName string = "default area"

const DefaultInstName string = "实例名"

// BKAppName the default app name
const BKAppName string = "蓝鲸"

// bk_classification_id value
const BKNetwork = "bk_network"

const (
	SNMPActionGet = "get"

	SNMPActionGetNext = "getnext"
)

const (
	// DefaultResModuleFlag the default resource module flag
	DefaultResModuleFlag int = 1

	// DefaultFaultModuleFlag the default fault module flag
	DefaultFaultModuleFlag int = 2

	// NormalModuleFlag create module by user , default =0
	NormalModuleFlag int = 0

	// NormalSetDefaultFlag user create set default field value
	NormalSetDefaultFlag int64 = 0

	// DefaultRecycleModuleFlag default recycle module flag
	DefaultRecycleModuleFlag int = 3

	// DefaultResSelfDefinedModuleFlag the default resource self-defined module flag
	DefaultResSelfDefinedModuleFlag int = 4
)

const (
	DefaultModuleType string = "1"
)

const (
	// FieldTypeSingleChar the single char filed type
	FieldTypeSingleChar string = "singlechar"

	// FieldTypeLongChar the long char field type
	FieldTypeLongChar string = "longchar"

	// FieldTypeInt the int field type
	FieldTypeInt string = "int"

	// FieldTypeFloat the float field type
	FieldTypeFloat string = "float"

	// FieldTypeEnum the enum field type
	FieldTypeEnum string = "enum"

	// FieldTypeDate the date field type
	FieldTypeDate string = "date"

	// FieldTypeTime the time field type
	FieldTypeTime string = "time"

	// FieldTypeUser the user field type
	FieldTypeUser string = "objuser"

	// FieldTypeTimeZone the timezone field type
	FieldTypeTimeZone string = "timezone"

	// FieldTypeBool the bool type
	FieldTypeBool string = "bool"

	// FieldTypeList the list type
	FieldTypeList string = "list"

	// FieldTypeTable the table type, inner type.
	FieldTypeTable string = "table"

	// FieldTypeOrganization the organization field type
	FieldTypeOrganization string = "organization"

	// FieldTypeSingleLenChar the single char length limit
	FieldTypeSingleLenChar int = 256

	// FieldTypeLongLenChar the long char length limit
	FieldTypeLongLenChar int = 2000

	// FieldTypeUserLenChar the user char length limit
	FieldTypeUserLenChar int = 2000

	//FieldTypeStrictCharRegexp the single char regex expression
	FieldTypeStrictCharRegexp string = `^[a-zA-Z]\w*$`

	//FieldTypeServiceCategoryRegexp the service category regex expression
	FieldTypeServiceCategoryRegexp string = `^([\w\p{Han}]|[:\-\(\)])+$`

	//FieldTypeMainlineRegexp the mainline instance name regex expression
	FieldTypeMainlineRegexp string = `^[^\\\|\/:\*,<>"\?#\s]+$`

	//FieldTypeSingleCharRegexp the single char regex expression
	//FieldTypeSingleCharRegexp string = `^([\w\p{Han}]|[，。？！={}|?<>~～、：＃；％＊——……＆·＄（）‘’“”\[\]『』〔〕｛｝【】￥￡♀‖〖〗《》「」:,;\."'\/\\\+\-\s#@\(\)])+$`
	FieldTypeSingleCharRegexp string = `\S`

	//FieldTypeLongCharRegexp the long char regex expression\
	//FieldTypeLongCharRegexp string = `^([\w\p{Han}]|[，。？！={}|?<>~～、：＃；％＊——……＆·＄（）‘’“”\[\]『』〔〕｛｝【】￥￡♀‖〖〗《》「」:,;\."'\/\\\+\-\s#@\(\)])+$`
	FieldTypeLongCharRegexp string = `\S`
)

const (
	// HostAddMethodExcel add a host method
	HostAddMethodExcel = "1"

	// HostAddMethodAgent add a  agent method
	HostAddMethodAgent = "2"

	// HostAddMethodAPI add api method
	HostAddMethodAPI = "3"

	// HostAddMethodExcelIndexOffset the height of the table header
	HostAddMethodExcelIndexOffset = 3

	// HostAddMethodExcelAssociationIndexOffset
	HostAddMethodExcelAssociationIndexOffset = 2

	/*EXCEL color AARRGGBB :
	AA means Alpha
	RRGGBB means Red, in hex.
	GG means Red, in hex.
	BB means Red, in hex.
	*/

	// ExcelHeaderFirstRowColor cell bg color
	ExcelHeaderFirstRowColor = "FF92D050"
	// ExcelHeaderFirstRowFontColor  font color
	ExcelHeaderFirstRowFontColor = "00000000"
	// ExcelHeaderFirstRowRequireFontColor require font color
	ExcelHeaderFirstRowRequireFontColor = "FFFF0000"
	// ExcelHeaderOtherRowColor cell bg color
	ExcelHeaderOtherRowColor = "FFC6EFCE"
	// ExcelHeaderOtherRowFontColor font color
	ExcelHeaderOtherRowFontColor = "FF000000"
	// ExcelCellDefaultBorderColor black color
	ExcelCellDefaultBorderColor = "FFD4D4D4"
	// ExcelHeaderFirstColumnColor light gray
	ExcelHeaderFirstColumnColor = "fee9da"
	// ExcelFirstColumnCellColor dark gray
	ExcelFirstColumnCellColor = "fabf8f"

	// ExcelAsstPrimaryKeySplitChar split char
	ExcelAsstPrimaryKeySplitChar = ","
	// ExcelAsstPrimaryKeyJoinChar split char
	ExcelAsstPrimaryKeyJoinChar = "="
	// ExcelAsstPrimaryKeyRowChar split char
	ExcelAsstPrimaryKeyRowChar = "\n"

	// ExcelDelAsstObjectRelation delete asst object relation
	ExcelDelAsstObjectRelation = "/"

	// ExcelDataValidationListLen excel dropdown list item count
	ExcelDataValidationListLen = 50

	// ExcelCommentSheetCotentLangPrefixKey excel comment sheet centent language prefixe key
	ExcelCommentSheetCotentLangPrefixKey = "import_comment"

	// ExcelFirstColumnFieldName export excel first column for tips
	ExcelFirstColumnFieldName = "field_name"
	ExcelFirstColumnFieldType = "field_type"
	ExcelFirstColumnFieldID   = "field_id"
	ExcelFirstColumnInstData  = "inst_data"

	ExcelFirstColumnAssociationAttribute = "excel_association_attribute"
	ExcelFirstColumnFieldDescription     = "excel_field_description"

	// the value of ignored excel cell
	ExcelCellIgnoreValue = "--"
)

const (
	// InputTypeExcel  data from excel
	InputTypeExcel = "excel"

	// InputTypeApiHostSync data from api for synchronize new host
	InputTypeApiNewHostSync = "api_sync_host"

	// BatchHostAddMaxRow batch sync add host max row
	BatchHostAddMaxRow = 128

	// ExcelImportMaxRow excel import max row
	ExcelImportMaxRow = 1000
)

const (
	// HTTPBKAPIErrorMessage apiserver error message
	HTTPBKAPIErrorMessage = "bk_error_msg"

	// HTTPBKAPIErrorCode apiserver error code
	HTTPBKAPIErrorCode = "bk_error_code"
)

// KvMap the map definition
type KvMap map[string]interface{}

const (
	// CCSystemOperatorUserName the system user
	CCSystemOperatorUserName  = "cc_system"
	CCSystemCollectorUserName = "cc_collector"
)

// APIRsp the result the http requst
type APIRsp struct {
	HTTPCode int         `json:"-"`
	Result   bool        `json:"result"`
	Code     int         `json:"code"`
	Message  interface{} `json:"message"`
	Data     interface{} `json:"data"`
}

const (
	// BKCacheKeyV3Prefix the prefix definition
	BKCacheKeyV3Prefix = "cc:v3:"
)

// event cache keys
const (
	EventCacheEventIDKey = BKCacheKeyV3Prefix + "event:inst_id"
	RedisSnapKeyPrefix   = BKCacheKeyV3Prefix + "snapshot:"
)

// api cache keys
const (
	ApiCacheLimiterRulePrefix = BKCacheKeyV3Prefix + "api:limiter_rule:"
)

const (
	// BKHTTPHeaderUser current request http request header fields name for login user
	BKHTTPHeaderUser = "BK_User"
	// BKHTTPLanguage the language key word
	BKHTTPLanguage = "HTTP_BLUEKING_LANGUAGE"
	// BKHTTPOwnerID the owner
	BKHTTPOwner = "HTTP_BK_SUPPLIER_ACCOUNT"
	// BKHTTPOwnerID the owner id
	BKHTTPOwnerID           = "HTTP_BLUEKING_SUPPLIER_ID"
	BKHTTPCookieLanugageKey = "blueking_language"
	BKHTTPRequestAppCode    = "Bk-App-Code"
	BKHTTPRequestRealIP     = "X-Real-Ip"

	// BKHTTPCCRequestID cc request id cc_request_id
	BKHTTPCCRequestID = "Cc_Request_Id"
	// BKHTTPOtherRequestID esb request id  X-Bkapi-Request-Id
	BKHTTPOtherRequestID = "X-Bkapi-Request-Id"

	BKHTTPSecretsToken   = "BK-Secrets-Token"
	BKHTTPSecretsProject = "BK-Secrets-Project"
	BKHTTPSecretsEnv     = "BK-Secrets-Env"
	// BKHTTPReadReference  query db use secondary node
	BKHTTPReadReference = "Cc_Read_Preference"
)

type ReadPreferenceMode string

func (r ReadPreferenceMode) String() string {
	return string(r)
}

// BKHTTPReadRefernceMode constants  这个位置对应的是mongodb 的read preference 的mode，如果driver 没有变化这里是不需要变更的，
// 新增mode 需要修改src/storage/dal/mongo/local/mongo.go 中的getCollectionOption 方法来支持
const (
	// NilMode not set
	NilMode ReadPreferenceMode = ""
	// PrimaryMode indicates that only a primary is
	// considered for reading. This is the default
	// mode.
	PrimaryMode ReadPreferenceMode = "1"
	// PrimaryPreferredMode indicates that if a primary
	// is available, use it; otherwise, eligible
	// secondaries will be considered.
	PrimaryPreferredMode ReadPreferenceMode = "2"
	// SecondaryMode indicates that only secondaries
	// should be considered.
	SecondaryMode ReadPreferenceMode = "3"
	// SecondaryPreferredMode indicates that only secondaries
	// should be considered when one is available. If none
	// are available, then a primary will be considered.
	SecondaryPreferredMode ReadPreferenceMode = "4"
	// NearestMode indicates that all primaries and secondaries
	// will be considered.
	NearestMode ReadPreferenceMode = "5"
)

// transaction related
const (
	TransactionIdHeader      = "cc_transaction_id_string"
	TransactionTimeoutHeader = "cc_transaction_timeout"

	// mongodb default transaction timeout is 1 minute.
	TransactionDefaultTimeout = 2 * time.Minute
)

const (
	// DefaultAppLifeCycleNormal  biz life cycle normal
	DefaultAppLifeCycleNormal = "2"
)

// Host OS type enumeration value
const (
	HostOSTypeEnumLinux   = "1"
	HostOSTypeEnumWindows = "2"
	HostOSTypeEnumAIX     = "3"
	HostOSTypeEnumUNIX    = "4"
)

const (
	MaxProcessPrio = 10000
	MinProcessPrio = -100
)

// integer const
const (
	MaxUint64  = ^uint64(0)
	MinUint64  = 0
	MaxInt64   = int64(MaxUint64 >> 1)
	MinInt64   = -MaxInt64 - 1
	MaxUint    = ^uint(0)
	MinUint    = 0
	MaxInt     = int(MaxUint >> 1)
	MinInt     = -MaxInt - 1
	MaxFloat64 = math.MaxFloat64
	MinFloat64 = -math.MaxFloat64
)

// flag
const HostCrossBizField = "hostcrossbiz"
const HostCrossBizValue = "e76fd4d1683d163e4e7e79cef45a74c1"

// config admin
const (
	ConfigAdminID         = "configadmin"
	ConfigAdminValueField = "config"
)

const (
	// APPConfigWaitTime application wait config from zookeeper time (unit sencend)
	APPConfigWaitTime = 15
)

const (
	// URLFilterWhiteList url filter white list not execute any filter
	// multiple url separated by commas
	URLFilterWhiteListSuffix = "/healthz,/version"

	URLFilterWhiteListSepareteChar = ","
)

type DataStatusFlag string

const (
	DataStatusDisabled DataStatusFlag = "disabled"
	DataStatusEnable   DataStatusFlag = "enable"
)

const (
	BKDataStatusField   = "bk_data_status"
	BKDataRecoverSuffix = "(recover)"
)

const (
	// period default value
	Infinite = "∞"
)

// netcollect
const (
	BKNetDevice   = "net_device"
	BKNetProperty = "net_property"
)

const (
	// login type
	BKBluekingLoginPluginVersion   = "blueking"
	BKOpenSourceLoginPluginVersion = "opensource"
	BKSkipLoginPluginVersion       = "skip-login"

	// monitor plugin type
	BKNoopMonitorPlugin     = "noop"
	BKBluekingMonitorPlugin = "blueking"

	HTTPCookieBKToken = "bk_token"

	WEBSessionUinKey           = "username"
	WEBSessionChineseNameKey   = "chName"
	WEBSessionPhoneKey         = "phone"
	WEBSessionEmailKey         = "email"
	WEBSessionRoleKey          = "role"
	WEBSessionOwnerUinKey      = "owner_uin"
	WEBSessionOwnerUinListeKey = "owner_uin_list"
	WEBSessionAvatarUrlKey     = "avatar_url"
	WEBSessionMultiSupplierKey = "multisupplier"

	LoginSystemMultiSupplierTrue  = "1"
	LoginSystemMultiSupplierFalse = "0"

	LogoutHTTPSchemeCookieKey = "http_scheme"
	LogoutHTTPSchemeHTTP      = "http"
	LogoutHTTPSchemeHTTPS     = "https"
)

const BKStatusField = "status"

const (
	BKProcInstanceOpUser             = "proc instance user"
	BKSynchronizeDataTaskDefaultUser = "synchronize task user"

	BKCloudSyncUser = "cloud_sync_user"
)

const (
	RedisProcSrvHostInstanceRefreshModuleKey  = BKCacheKeyV3Prefix + "prochostinstancerefresh:set"
	RedisProcSrvHostInstanceAllRefreshLockKey = BKCacheKeyV3Prefix + "lock:prochostinstancerefresh"
	RedisProcSrvQueryProcOPResultKey          = BKCacheKeyV3Prefix + "procsrv:query:opresult:set"
	RedisCloudSyncInstancePendingStart        = BKCacheKeyV3Prefix + "cloudsyncinstancependingstart:list"
	RedisCloudSyncInstanceStarted             = BKCacheKeyV3Prefix + "cloudsyncinstancestarted:list"
	RedisCloudSyncInstancePendingStop         = BKCacheKeyV3Prefix + "cloudsyncinstancependingstop:list"
	RedisMongoCacheSyncKey                    = BKCacheKeyV3Prefix + "mongodb:cache"
)

// association fields
const (
	// the id of the association kind
	AssociationKindIDField    = "bk_asst_id"
	AssociationKindNameField  = "bk_asst_name"
	AssociationObjAsstIDField = "bk_obj_asst_id"
	AssociatedObjectIDField   = "bk_asst_obj_id"
)

// association
const (
	AssociationKindMainline = "bk_mainline"
	AssociationTypeBelong   = "belong"
	AssociationTypeGroup    = "group"
	AssociationTypeRun      = "run"
	AssociationTypeConnect  = "connect"
	AssociationTypeDefault  = "default"
)

const (
	// MetadataField data business key
	MetadataField = "metadata"
)

const (
	BKBizDefault = "bizdefault"
)

const (
	// MetaDataSynchronizeField Synchronous data aggregation field
	MetaDataSynchronizeField = "sync"
	// MetaDataSynchronizeFlagField synchronize flag
	MetaDataSynchronizeFlagField = "flag"
	// MetaDataSynchronizeVersionField synchronize version
	MetaDataSynchronizeVersionField = "version"
	// MetaDataSynchronizeIdentifierField 数据需要同步cmdb系统的身份标识， 值是数组
	MetaDataSynchronizeIdentifierField = "identifier"
	// MetaDataSynchronIdentifierFlagSyncAllValue 数据可以被任何系统同步
	MetaDataSynchronIdentifierFlagSyncAllValue = "__bk_cmdb__"

	// SynchronizeSignPrefix  synchronize sign , Should appear in the configuration file
	SynchronizeSignPrefix = "sync_blueking"

	/* synchronize model description classify*/

	// SynchronizeModelTypeClassification synchroneize model classification
	SynchronizeModelTypeClassification = "model_classification"
	// SynchronizeModelTypeAttribute synchroneize model attribute
	SynchronizeModelTypeAttribute = "model_attribute"
	// SynchronizeModelTypeAttributeGroup synchroneize model attribute group
	SynchronizeModelTypeAttributeGroup = "model_atrribute_group"
	// SynchronizeModelTypeBase synchroneize model attribute
	SynchronizeModelTypeBase = "model"

	/* synchronize instance assoication sign*/

	// SynchronizeAssociationTypeModelHost synchroneize model ggroup
	SynchronizeAssociationTypeModelHost = "module_host"
)

const (
	AttributePlaceHolderMaxLength = 2000
	AttributeOptionMaxLength      = 2000
	AttributeIDMaxLength          = 128
	AttributeNameMaxLength        = 128
	AttributeUnitMaxLength        = 20
	AttributeOptionValueMaxLength = 128
	AttributeOptionArrayMaxLength = 200
	ServiceCategoryMaxLength      = 128
)

const (
	NameFieldMaxLength         = 256
	MainlineNameFieldMaxLength = 256

	// 用于表示还未设置服务模板的情况，比如没有绑定服务模板
	ServiceTemplateIDNotSet = 0
	SetTemplateIDNotSet     = 0

	MetadataLabelBiz = "metadata.label.bk_biz_id"

	DefaultServiceCategoryName = "Default"
)

const (
	ContextRequestIDField    = "request_id"
	ContextRequestUserField  = "request_user"
	ContextRequestOwnerField = "request_owner"
)

const (
	OperationCustom      = "custom"
	OperationReportType  = "report_type"
	OperationConfigID    = "config_id"
	BizModuleHostChart   = "biz_module_host_chart"
	HostOSChart          = "host_os_chart"
	HostBizChart         = "host_biz_chart"
	HostCloudChart       = "host_cloud_chart"
	HostChangeBizChart   = "host_change_biz_chart"
	ModelAndInstCount    = "model_and_inst_count"
	ModelInstChart       = "model_inst_chart"
	ModelInstChangeChart = "model_inst_change_chart"
	CreateObject         = "create object"
	DeleteObject         = "delete object"
	UpdateObject         = "update object"
	OperationDescription = "op_desc"
	OptionOther          = "其他"
	TimerPattern         = "^[\\d]+\\:[\\d]+$"
	SyncSetTaskName      = "sync-settemplate2set"

	BKHostState = "bk_state"
)

// multiple language support
type LanguageType string

const (
	Chinese LanguageType = "zh-cn"
	English LanguageType = "en"
)

// cloud sync const
const (
	BKCloudAccountID             = "bk_account_id"
	BKCloudAccountName           = "bk_account_name"
	BKCloudVendor                = "bk_cloud_vendor"
	BKCloudSyncTaskName          = "bk_task_name"
	BKCloudSyncTaskID            = "bk_task_id"
	BKCloudSyncStatus            = "bk_sync_status"
	BKCloudSyncStatusDescription = "bk_status_description"
	BKCloudLastSyncTime          = "bk_last_sync_time"
	BKCreator                    = "bk_creator"
	BKStatus                     = "bk_status"
	BKStatusDetail               = "bk_status_detail"
	BKLastEditor                 = "bk_last_editor"
	BKSecretID                   = "bk_secret_id"
	BKVpcID                      = "bk_vpc_id"
	BKVpcName                    = "bk_vpc_name"
	BKRegion                     = "bk_region"
	BKCloudSyncVpcs              = "bk_sync_vpcs"

	// 是否为被销毁的云主机
	IsDestroyedCloudHost = "is_destroyed_cloud_host"
)

const (
	BKCloudHostStatusUnknown   = "1"
	BKCloudHostStatusStarting  = "2"
	BKCloudHostStatusRunning   = "3"
	BKCloudHostStatusStopping  = "4"
	BKCloudHostStatusStopped   = "5"
	BKCloudHostStatusDestroyed = "6"
)

const (
	BKCloudAreaStatusNormal   = "1"
	BKCloudAreaStatusAbnormal = "2"
)

// configcenter
const (
	BKDefaultConfigCenter = "zookeeper"
)
