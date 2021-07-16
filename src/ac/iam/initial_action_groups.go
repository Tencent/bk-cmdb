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

// GenerateActionGroups generate all the resource action groups registered to IAM.
func GenerateActionGroups() []ActionGroup {
	ActionGroups := make([]ActionGroup, 0)

	// generate business manage action groups, contains business related actions
	ActionGroups = append(ActionGroups, genBusinessManageActionGroups()...)

	// generate resource manage action groups, contains resource related actions
	ActionGroups = append(ActionGroups, genResourceManageActionGroups()...)

	// generate model manage action groups, contains model related actions
	ActionGroups = append(ActionGroups, genModelManageActionGroups()...)

	// generate operation statistic action groups, contains operation statistic and audit related actions
	ActionGroups = append(ActionGroups, genOperationStatisticActionGroups()...)

	// generate global settings action groups, contains global settings related actions
	ActionGroups = append(ActionGroups, genGlobalSettingsActionGroups()...)

	return ActionGroups
}

func genBusinessManageActionGroups() []ActionGroup {
	return []ActionGroup{
		{
			Name:   "业务管理",
			NameEn: "Business Manage",
			Actions: []ActionWithID{
				{
					ID: ViewBusinessResource,
				},
			},
			SubGroups: []ActionGroup{
				{
					Name:   "业务主机",
					NameEn: "Business Host",
					Actions: []ActionWithID{
						{
							ID: EditBusinessHost,
						},
						{
							ID: BusinessHostTransferToResourcePool,
						},
						{
							ID: HostTransferAcrossBusiness,
						},
					},
				},
				{
					Name:   "业务拓扑",
					NameEn: "Business Topology",
					Actions: []ActionWithID{
						{
							ID: CreateBusinessTopology,
						},
						{
							ID: EditBusinessTopology,
						},
						{
							ID: DeleteBusinessTopology,
						},
					},
				},
				{
					Name:   "服务实例",
					NameEn: "Service Instance",
					Actions: []ActionWithID{
						{
							ID: CreateBusinessServiceInstance,
						},
						{
							ID: EditBusinessServiceInstance,
						},
						{
							ID: DeleteBusinessServiceInstance,
						},
					},
				},
				{
					Name:   "服务模版",
					NameEn: "Service Template",
					Actions: []ActionWithID{
						{
							ID: CreateBusinessServiceTemplate,
						},
						{
							ID: EditBusinessServiceTemplate,
						},
						{
							ID: DeleteBusinessServiceTemplate,
						},
					},
				},
				{
					Name:   "集群模版",
					NameEn: "Set Template",
					Actions: []ActionWithID{
						{
							ID: CreateBusinessSetTemplate,
						},
						{
							ID: EditBusinessSetTemplate,
						},
						{
							ID: DeleteBusinessSetTemplate,
						},
					},
				},
				{
					Name:   "服务分类",
					NameEn: "Service Category",
					Actions: []ActionWithID{
						{
							ID: CreateBusinessServiceCategory,
						},
						{
							ID: EditBusinessServiceCategory,
						},
						{
							ID: DeleteBusinessServiceCategory,
						},
					},
				},
				{
					Name:   "动态分组",
					NameEn: "Dynamic Grouping",
					Actions: []ActionWithID{
						{
							ID: CreateBusinessCustomQuery,
						},
						{
							ID: EditBusinessCustomQuery,
						},
						{
							ID: DeleteBusinessCustomQuery,
						},
					},
				},
				{
					Name:   "业务自定义字段",
					NameEn: "Business Custom Field",
					Actions: []ActionWithID{
						{
							ID: EditBusinessCustomField,
						},
					},
				},
				{
					Name:   "主机自动应用",
					NameEn: "Business Host Apply",
					Actions: []ActionWithID{
						{
							ID: EditBusinessHostApply,
						},
					},
				},
			},
		},
	}
}

func genResourceManageActionGroups() []ActionGroup {
	return []ActionGroup{
		{
			Name:   "资源管理",
			NameEn: "Resource Manage",
			SubGroups: []ActionGroup{
				{
					Name:   "主机池",
					NameEn: "Host Pool",
					Actions: []ActionWithID{
						{
							ID: CreateResourcePoolHost,
						},
						{
							ID: EditResourcePoolHost,
						},
						{
							ID: DeleteResourcePoolHost,
						},
						{
							ID: ResourcePoolHostTransferToBusiness,
						},
						{
							ID: ResourcePoolHostTransferToDirectory,
						},
						{
							ID: CreateResourcePoolDirectory,
						},
						{
							ID: EditResourcePoolDirectory,
						},
						{
							ID: DeleteResourcePoolDirectory,
						},
					},
				},
				{
					Name:   "业务",
					NameEn: "Business",
					Actions: []ActionWithID{
						{
							ID: CreateBusiness,
						},
						{
							ID: EditBusiness,
						},
						{
							ID: ArchiveBusiness,
						},
						{
							ID: FindBusiness,
						},
					},
				},
				{
					Name:   "实例",
					NameEn: "Configuration Instance",
					Actions: []ActionWithID{
						{
							ID: CreateSysInstance,
						},
						{
							ID: EditSysInstance,
						},
						{
							ID: DeleteSysInstance,
						},
					},
				},
				{
					Name:   "云账户",
					NameEn: "Cloud Account",
					Actions: []ActionWithID{
						{
							ID: CreateCloudAccount,
						},
						{
							ID: EditCloudAccount,
						},
						{
							ID: DeleteCloudAccount,
						},
						{
							ID: FindCloudAccount,
						},
					},
				},
				{
					Name:   "云资源任务",
					NameEn: "Cloud Resource Task",
					Actions: []ActionWithID{
						{
							ID: CreateCloudResourceTask,
						},
						{
							ID: EditCloudResourceTask,
						},
						{
							ID: DeleteCloudResourceTask,
						},
						{
							ID: FindCloudResourceTask,
						},
					},
				},
				{
					Name:   "云区域",
					NameEn: "Cloud Area",
					Actions: []ActionWithID{
						{
							ID: CreateCloudArea,
						},
						{
							ID: EditCloudArea,
						},
						{
							ID: DeleteCloudArea,
						},
					},
				},
				{
					Name:   "事件监听",
					NameEn: "Event Watch",
					Actions: []ActionWithID{
						{
							ID: WatchHostEvent,
						},
						{
							ID: WatchHostRelationEvent,
						},
						{
							ID: WatchBizEvent,
						},
						{
							ID: WatchSetEvent,
						},
						{
							ID: WatchModuleEvent,
						},
						{
							ID: WatchProcessEvent,
						},
						{
							ID: WatchCommonInstanceEvent,
						},
						{
							ID: WatchMainlineInstanceEvent,
						},
					},
				},
			},
		},
	}
}

func genModelManageActionGroups() []ActionGroup {
	return []ActionGroup{
		{
			Name:   "模型管理",
			NameEn: "Model Mange",
			SubGroups: []ActionGroup{
				{
					Name:   "模型分组",
					NameEn: "Model Group",
					Actions: []ActionWithID{
						{
							ID: CreateModelGroup,
						},
						{
							ID: EditModelGroup,
						},
						{
							ID: DeleteModelGroup,
						},
					},
				},
				{
					Name:   "模型关系",
					NameEn: "Model Relation",
					Actions: []ActionWithID{
						{
							ID: EditBusinessLayer,
						},
						{
							ID: EditModelTopologyView,
						},
					},
				},
				{
					Name:   "模型",
					NameEn: "Model",
					Actions: []ActionWithID{
						{
							ID: CreateSysModel,
						},
						{
							ID: EditSysModel,
						},
						{
							ID: DeleteSysModel,
						},
					},
				},
				{
					Name:   "关联类型",
					NameEn: "Association Type",
					Actions: []ActionWithID{
						{
							ID: CreateAssociationType,
						},
						{
							ID: EditAssociationType,
						},
						{
							ID: DeleteAssociationType,
						},
					},
				},
			},
		},
	}
}

func genOperationStatisticActionGroups() []ActionGroup {
	return []ActionGroup{
		{
			Name:   "运营统计",
			NameEn: "Operation Statistic",
			SubGroups: []ActionGroup{
				{
					Name:   "运营统计",
					NameEn: "Operation Statistic",
					Actions: []ActionWithID{
						{
							ID: FindOperationStatistic,
						},
						{
							ID: EditOperationStatistic,
						},
					},
				},
				{
					Name:   "操作审计",
					NameEn: "Operation Audit",
					Actions: []ActionWithID{
						{
							ID: FindAuditLog,
						},
					},
				},
			},
		},
	}
}

func genGlobalSettingsActionGroups() []ActionGroup {
	return []ActionGroup{
		{
			Name:   "全局设置",
			NameEn: "Global Settings",
			SubGroups: []ActionGroup{
				{
					Name:   "全局设置",
					NameEn: "Global Settings",
					Actions: []ActionWithID{
						{
							ID: GlobalSettings,
						},
					},
				},
			},
		},
	}
}
