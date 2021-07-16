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
	businessParent = Parent{
		SystemID:   SystemIDCMDB,
		ResourceID: Business,
	}
)

var ResourceTypeIDMap = map[TypeID]string{
	Business:                  "业务",
	BusinessForHostTrans:      "业务主机",
	SysCloudArea:              "云区域",
	SysResourcePoolDirectory:  "主机池目录",
	SysHostRscPoolDirectory:   "主机池主机",
	SysModelGroup:             "模型分组",
	SysInstanceModel:          "实例模型",
	SysModel:                  "模型",
	SysModelEvent:             "模型列表",
	MainlineModelEvent:        "资源事件",
	SysInstance:               "实例",
	SysAssociationType:        "关联类型",
	SysOperationStatistic:     "运营统计",
	SysAuditLog:               "操作审计",
	SysCloudAccount:           "云账户",
	SysCloudResourceTask:      "云资源任务",
	SysEventWatch:             "事件监听",
	Host:                      "主机",
	BizHostApply:              "主机自动应用",
	BizCustomQuery:            "动态分组",
	BizCustomField:            "自定义字段",
	BizProcessServiceInstance: "服务实例",
	BizProcessServiceCategory: "服务分类",
	BizSetTemplate:            "集群模板",
	BizTopology:               "业务拓扑",
	BizProcessServiceTemplate: "服务模板",
}

// GenerateResourceTypes generate all the resource types registered to IAM.
func GenerateResourceTypes() []ResourceType {
	resourceTypeList := make([]ResourceType, 0)

	// add public resources
	resourceTypeList = append(resourceTypeList, genPublicResources()...)

	// add business resources
	resourceTypeList = append(resourceTypeList, genBusinessResources()...)

	return resourceTypeList
}

// GetResourceParentMap generate resource types' mapping to parents.
func GetResourceParentMap() map[TypeID][]TypeID {
	resourceParentMap := make(map[TypeID][]TypeID, 0)
	for _, resourceType := range GenerateResourceTypes() {
		for _, parent := range resourceType.Parents {
			resourceParentMap[resourceType.ID] = append(resourceParentMap[resourceType.ID], parent.ResourceID)
		}
	}
	return resourceParentMap
}

func genBusinessResources() []ResourceType {
	return []ResourceType{
		{
			ID:            Host,
			Name:          ResourceTypeIDMap[Host],
			NameEn:        "Host",
			Description:   "主机",
			DescriptionEn: "hosts under a business or in resource pool",
			Parents: []Parent{{
				SystemID: SystemIDCMDB,
				//ResourceID: Module,
				ResourceID: Business,
			}, {
				SystemID:   SystemIDCMDB,
				ResourceID: SysResourcePoolDirectory,
			}},
			ProviderConfig: ResourceConfig{
				Path: "/auth/v3/find/resource",
			},
			Version: 1,
		},
		{
			ID:            BizHostApply,
			Name:          ResourceTypeIDMap[BizHostApply],
			NameEn:        "Host Apply",
			Description:   "自动应用业务主机的属性信息",
			DescriptionEn: "apply business host attribute automatically",
			Parents:       []Parent{businessParent},
			ProviderConfig: ResourceConfig{
				Path: "/auth/v3/find/resource",
			},
			Version: 1,
		},
		{
			ID:            BizCustomQuery,
			Name:          ResourceTypeIDMap[BizCustomQuery],
			NameEn:        "Dynamic Grouping",
			Description:   "根据条件查询主机信息",
			DescriptionEn: "custom query the host instances",
			Parents:       []Parent{businessParent},
			ProviderConfig: ResourceConfig{
				Path: "/auth/v3/find/resource",
			},
			Version: 1,
		},
		{
			ID:            BizCustomField,
			Name:          ResourceTypeIDMap[BizCustomField],
			NameEn:        "Custom Field",
			Description:   "模型在业务下的自定义字段",
			DescriptionEn: "model's custom field under a business",
			Parents:       []Parent{businessParent},
			ProviderConfig: ResourceConfig{
				Path: "/auth/v3/find/resource",
			},
			Version: 1,
		},
		{
			ID:            BizProcessServiceInstance,
			Name:          ResourceTypeIDMap[BizProcessServiceInstance],
			NameEn:        "Service Instance",
			Description:   "服务实例",
			DescriptionEn: "service instance",
			Parents:       []Parent{businessParent},
			ProviderConfig: ResourceConfig{
				Path: "/auth/v3/find/resource",
			},
			Version: 1,
		},
		{
			ID:            BizProcessServiceCategory,
			Name:          ResourceTypeIDMap[BizProcessServiceCategory],
			NameEn:        "Service Category",
			Description:   "服务分类用于分类服务实例",
			DescriptionEn: "service category is to classify service instances",
			Parents:       []Parent{businessParent},
			ProviderConfig: ResourceConfig{
				Path: "/auth/v3/find/resource",
			},
			Version: 1,
		},
		{
			ID:            BizSetTemplate,
			Name:          ResourceTypeIDMap[BizSetTemplate],
			NameEn:        "Set Template",
			Description:   "集群模板用于实例化集群",
			DescriptionEn: "set template is used to instantiate a set",
			Parents:       []Parent{businessParent},
			ProviderConfig: ResourceConfig{
				Path: "/auth/v3/find/resource",
			},
			Version: 1,
		},
		{
			ID:            BizTopology,
			Name:          ResourceTypeIDMap[BizTopology],
			NameEn:        "Business Topology",
			Description:   "业务拓扑包含了业务拓扑树中所有的相关元素",
			DescriptionEn: "business topology contains all elements related to the business topology tree",
			Parents:       []Parent{businessParent},
			ProviderConfig: ResourceConfig{
				Path: "/auth/v3/find/resource",
			},
			Version: 1,
		},
		{
			ID:            BizProcessServiceTemplate,
			Name:          ResourceTypeIDMap[BizProcessServiceTemplate],
			NameEn:        "Service Template",
			Description:   "服务模板用于实例化服务实例",
			DescriptionEn: "service template is used to instantiate a service instance ",
			Parents:       []Parent{businessParent},
			ProviderConfig: ResourceConfig{
				Path: "/auth/v3/find/resource",
			},
			Version: 1,
		},
		// only for host topology usage, not related to actions
		//{
		//	ID:            Set,
		//	Name:          ResourceTypeIDMap[Set],
		//	NameEn:        "Set",
		//	Description:   "集群列表",
		//	DescriptionEn: "all the sets in blueking cmdb.",
		//	Parents:       []Parent{businessParent},
		//	ProviderConfig: ResourceConfig{
		//		Path: "/auth/v3/find/resource",
		//	},
		//	Version: 1,
		//},
		//{
		//	ID:            Module,
		//	Name:          ResourceTypeIDMap[Module],
		//	NameEn:        "Module",
		//	Description:   "模块列表",
		//	DescriptionEn: "all the modules in blueking cmdb.",
		//	Parents: []Parent{{
		//		SystemID:   SystemIDCMDB,
		//		ResourceID: Set,
		//	}},
		//	ProviderConfig: ResourceConfig{
		//		Path: "/auth/v3/find/resource",
		//	},
		//	Version: 1,
		//},
	}
}

func genPublicResources() []ResourceType {
	return []ResourceType{
		{
			ID:            Business,
			Name:          ResourceTypeIDMap[Business],
			NameEn:        "Business",
			Description:   "业务列表",
			DescriptionEn: "all the business in blueking cmdb.",
			Parents:       nil,
			ProviderConfig: ResourceConfig{
				Path: "/auth/v3/find/resource",
			},
			Version: 1,
		},
		{
			ID:            BusinessForHostTrans,
			Name:          ResourceTypeIDMap[BusinessForHostTrans],
			NameEn:        "Host In Business",
			Description:   "业务主机",
			DescriptionEn: "host in business",
			Parents:       nil,
			ProviderConfig: ResourceConfig{
				Path: "/auth/v3/find/resource",
			},
			Version: 1,
		},
		{
			ID:            SysCloudArea,
			Name:          ResourceTypeIDMap[SysCloudArea],
			NameEn:        "Cloud Area",
			Description:   "云区域",
			DescriptionEn: "cloud area",
			Parents:       nil,
			ProviderConfig: ResourceConfig{
				Path: "/auth/v3/find/resource",
			},
			Version: 1,
		},
		{
			ID:            SysResourcePoolDirectory,
			Name:          ResourceTypeIDMap[SysResourcePoolDirectory],
			NameEn:        "Host Pool Directory",
			Description:   "主机池目录",
			DescriptionEn: "host pool directory",
			Parents:       nil,
			ProviderConfig: ResourceConfig{
				Path: "/auth/v3/find/resource",
			},
			Version: 1,
		},
		{
			ID:            SysHostRscPoolDirectory,
			Name:          ResourceTypeIDMap[SysHostRscPoolDirectory],
			NameEn:        "Host In Host Pool Directory",
			Description:   "主机池主机",
			DescriptionEn: "host in host pool directory",
			Parents:       nil,
			ProviderConfig: ResourceConfig{
				Path: "/auth/v3/find/resource",
			},
			Version: 1,
		},
		{
			ID:            SysModelGroup,
			Name:          ResourceTypeIDMap[SysModelGroup],
			NameEn:        "Model Group",
			Description:   "模型分组用于对模型进行归类",
			DescriptionEn: "group models by model group",
			Parents:       nil,
			ProviderConfig: ResourceConfig{
				Path: "/auth/v3/find/resource",
			},
			Version: 1,
		},
		{
			ID:            SysInstanceModel,
			Name:          ResourceTypeIDMap[SysInstanceModel],
			NameEn:        "InstanceModel",
			Description:   "实例模型",
			DescriptionEn: "instance model",
			Parents:       nil,
			ProviderConfig: ResourceConfig{
				Path: "/auth/v3/find/resource",
			},
			Version: 1,
		},
		{
			ID:            SysModel,
			Name:          ResourceTypeIDMap[SysModel],
			NameEn:        "Model",
			Description:   "模型",
			DescriptionEn: "model",
			Parents:       nil,
			ProviderConfig: ResourceConfig{
				Path: "/auth/v3/find/resource",
			},
			Version: 1,
		},
		{
			ID:            SysInstance,
			Name:          ResourceTypeIDMap[SysInstance],
			NameEn:        "Instance",
			Description:   "模型实例",
			DescriptionEn: "model instance",
			Parents: []Parent{{
				SystemID:   SystemIDCMDB,
				ResourceID: SysModel,
			}},
			ProviderConfig: ResourceConfig{
				Path: "/auth/v3/find/resource",
			},
			Version: 1,
		},
		{
			ID:            SysAssociationType,
			Name:          ResourceTypeIDMap[SysAssociationType],
			NameEn:        "Association Type",
			Description:   "关联类型是模型关联关系的分类",
			DescriptionEn: "association type is the classification of model association",
			Parents:       nil,
			ProviderConfig: ResourceConfig{
				Path: "/auth/v3/find/resource",
			},
			Version: 1,
		},
		{
			ID:            SysOperationStatistic,
			Name:          ResourceTypeIDMap[SysOperationStatistic],
			NameEn:        "Operational Statistics",
			Description:   "运营统计",
			DescriptionEn: "operational statistics",
			Parents:       nil,
			ProviderConfig: ResourceConfig{
				Path: "/auth/v3/find/resource",
			},
			Version: 1,
		},
		{
			ID:            SysAuditLog,
			Name:          ResourceTypeIDMap[SysAuditLog],
			NameEn:        "Operation Audit",
			Description:   "操作审计",
			DescriptionEn: "audit log",
			Parents:       nil,
			ProviderConfig: ResourceConfig{
				Path: "/auth/v3/find/resource",
			},
			Version: 1,
		},
		{
			ID:            SysCloudAccount,
			Name:          ResourceTypeIDMap[SysCloudAccount],
			NameEn:        "Cloud Account",
			Description:   "云账户",
			DescriptionEn: "cloud account",
			Parents:       nil,
			ProviderConfig: ResourceConfig{
				Path: "/auth/v3/find/resource",
			},
			Version: 1,
		},
		{
			ID:            SysCloudResourceTask,
			Name:          ResourceTypeIDMap[SysCloudResourceTask],
			NameEn:        "Cloud Resource Task",
			Description:   "云资源任务",
			DescriptionEn: "cloud resource task",
			Parents:       nil,
			ProviderConfig: ResourceConfig{
				Path: "/auth/v3/find/resource",
			},
			Version: 1,
		},
		{
			ID:            SysEventWatch,
			Name:          ResourceTypeIDMap[SysEventWatch],
			NameEn:        "Event Listen",
			Description:   "事件监听",
			DescriptionEn: "event watch",
			Parents:       nil,
			ProviderConfig: ResourceConfig{
				Path: "/auth/v3/find/resource",
			},
			Version: 1,
		},
		{
			ID:            SysModelEvent,
			Name:          ResourceTypeIDMap[SysModelEvent],
			NameEn:        "Model List",
			Description:   "模型列表",
			DescriptionEn: "model list",
			Parents:       nil,
			ProviderConfig: ResourceConfig{
				Path: "/auth/v3/find/resource",
			},
			Version: 1,
		},
		{
			ID:            MainlineModelEvent,
			Name:          ResourceTypeIDMap[MainlineModelEvent],
			NameEn:        "Resource Event",
			Description:   "资源事件",
			DescriptionEn: "resource event",
			Parents:       nil,
			ProviderConfig: ResourceConfig{
				Path: "/auth/v3/find/resource",
			},
			Version: 1,
		},
	}
}
