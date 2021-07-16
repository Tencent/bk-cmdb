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

var (
	businessChain = ResourceChain{
		SystemID: SystemIDCMDB,
		ID:       Business,
	}
)

// GenerateInstanceSelections generate all the instance selections registered to IAM.
func GenerateInstanceSelections() []InstanceSelection {
	return []InstanceSelection{
		{
			ID:                BusinessSelection,
			Name:              "业务列表",
			NameEn:            "Business List",
			ResourceTypeChain: []ResourceChain{businessChain},
		},
		{
			ID:     BusinessHostTransferSelection,
			Name:   "业务主机选择",
			NameEn: "Business",
			ResourceTypeChain: []ResourceChain{{
				SystemID: SystemIDCMDB,
				ID:       BusinessForHostTrans,
			}},
		},
		{
			ID:     SysResourcePoolDirectorySelection,
			Name:   "主机池目录列表",
			NameEn: "Host Pool Directory List",
			ResourceTypeChain: []ResourceChain{{
				SystemID: SystemIDCMDB,
				ID:       SysResourcePoolDirectory,
			}},
		},
		{
			ID:     SysHostRscPoolDirectorySelection,
			Name:   "主机池主机选择",
			NameEn: "Host Pool Directory",
			ResourceTypeChain: []ResourceChain{{
				SystemID: SystemIDCMDB,
				ID:       SysHostRscPoolDirectory,
			}},
		},
		{
			ID:     BizHostInstanceSelection,
			Name:   "业务主机列表",
			NameEn: "Business Host List",
			ResourceTypeChain: []ResourceChain{
				// select the business at first.
				businessChain,
				//{
				//	SystemID: SystemIDCMDB,
				//	ID:       Set,
				//},
				//{
				//	SystemID: SystemIDCMDB,
				//	ID:       Module,
				//},
				// then select the host instances.
				{
					SystemID: SystemIDCMDB,
					ID:       Host,
				},
			},
		},
		{
			ID:     BizCustomQuerySelection,
			Name:   "业务动态分组列表",
			NameEn: "Business Dynamic Grouping List",
			ResourceTypeChain: []ResourceChain{
				businessChain,
				{
					SystemID: SystemIDCMDB,
					ID:       BizCustomQuery,
				},
			},
		},
		{
			ID:     BizProcessServiceTemplateSelection,
			Name:   "服务模版列表",
			NameEn: "Service Template List",
			ResourceTypeChain: []ResourceChain{
				businessChain,
				{
					SystemID: SystemIDCMDB,
					ID:       BizProcessServiceTemplate,
				},
			},
		},
		{
			ID:     BizSetTemplateSelection,
			Name:   "集群模板列表",
			NameEn: "Set Template List",
			ResourceTypeChain: []ResourceChain{
				businessChain,
				{
					SystemID: SystemIDCMDB,
					ID:       BizSetTemplate,
				},
			},
		},
		{
			ID:     SysHostInstanceSelection,
			Name:   "主机池主机列表",
			NameEn: "Host Pool List",
			ResourceTypeChain: []ResourceChain{
				{
					SystemID: SystemIDCMDB,
					ID:       SysHostRscPoolDirectory,
				},
				{
					SystemID: SystemIDCMDB,
					ID:       Host,
				},
			},
		},
		{
			ID:     SysCloudAreaSelection,
			Name:   "云区域列表",
			NameEn: "Cloud Area List",
			ResourceTypeChain: []ResourceChain{
				{
					SystemID: SystemIDCMDB,
					ID:       SysCloudArea,
				},
			},
		},
		{
			ID:     SysInstanceSelection,
			Name:   "实例列表",
			NameEn: "Instance List",
			ResourceTypeChain: []ResourceChain{
				{
					SystemID: SystemIDCMDB,
					ID:       SysInstanceModel,
				},
				{
					SystemID: SystemIDCMDB,
					ID:       SysInstance,
				},
			},
		},
		{
			ID:     SysInstanceModelSelection,
			Name:   "实例模型列表",
			NameEn: "Instance Model List",
			ResourceTypeChain: []ResourceChain{
				{
					SystemID: SystemIDCMDB,
					ID:       SysInstanceModel,
				},
			},
		},
		{
			ID:     SysCloudAccountSelection,
			Name:   "云账户列表",
			NameEn: "Cloud Account List",
			ResourceTypeChain: []ResourceChain{
				{
					SystemID: SystemIDCMDB,
					ID:       SysCloudAccount,
				},
			},
		},
		{
			ID:     SysCloudResourceTaskSelection,
			Name:   "云资源任务列表",
			NameEn: "Cloud Resource Task List",
			ResourceTypeChain: []ResourceChain{
				{
					SystemID: SystemIDCMDB,
					ID:       SysCloudResourceTask,
				},
			},
		},
		{
			ID:     SysModelSelection,
			Name:   "模型列表",
			NameEn: "Model List",
			ResourceTypeChain: []ResourceChain{
				{
					SystemID: SystemIDCMDB,
					ID:       SysModel,
				},
			},
		},
		{
			ID:     SysAssociationTypeSelection,
			Name:   "关联类型",
			NameEn: "Association Type",
			ResourceTypeChain: []ResourceChain{
				{
					SystemID: SystemIDCMDB,
					ID:       SysAssociationType,
				},
			},
		},
		{
			ID:     SysModelGroupSelection,
			Name:   "模型分组",
			NameEn: "Model Group",
			ResourceTypeChain: []ResourceChain{
				{
					SystemID: SystemIDCMDB,
					ID:       SysModelGroup,
				},
			},
		},
		{
			ID:     SysModelEventSelection,
			Name:   "模型事件列表",
			NameEn: "Model Event List",
			ResourceTypeChain: []ResourceChain{
				{
					SystemID: SystemIDCMDB,
					ID:       SysModelEvent,
				},
			},
		},
		{
			ID:     MainlineModelEventSelection,
			Name:   "自定义拓扑层级列表",
			NameEn: "Custom Topo Layer Event List",
			ResourceTypeChain: []ResourceChain{
				{
					SystemID: SystemIDCMDB,
					ID:       MainlineModelEvent,
				},
			},
		},
	}
}
