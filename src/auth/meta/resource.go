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

package meta

type ResourceType string

func (r ResourceType) String() string {
	return string(r)
}

const (
	Business                 ResourceType = "business"
	Model                    ResourceType = "model"
	ModelModule              ResourceType = "modelModule"
	ModelSet                 ResourceType = "modelSet"
	MainlineModel            ResourceType = "mainlineObject"
	MainlineModelTopology    ResourceType = "mainlineObjectTopology"
	MainlineInstanceTopology ResourceType = "mainlineInstanceTopology"
	AssociationType          ResourceType = "associationType"
	ModelAssociation         ResourceType = "modelAssociation"
	ModelInstanceAssociation ResourceType = "modelInstanceAssociation"
	ModelInstance            ResourceType = "modelInstance"
	ModelInstanceTopology    ResourceType = "modelInstanceTopology"
	ModelTopology            ResourceType = "modelTopology"
	ModelClassification      ResourceType = "modelClassification"
	ModelAttributeGroup      ResourceType = "modelAttributeGroup"
	ModelAttribute           ResourceType = "modelAttribute"
	ModelUnique              ResourceType = "modelUnique"
	HostUserCustom           ResourceType = "hostUserCustom"
	HostFavorite             ResourceType = "hostFavorite"
	Process                  ResourceType = "process"
	NetDataCollector         ResourceType = "netDataCollector"
	DynamicGrouping          ResourceType = "dynamicGrouping"
)

const (
	Host                         = "host"
	ProcessConfigTemplate        = "processConfigTemplate"
	ProcessConfigTemplateVersion = "processConfigTemplateVersion"
	ProcessBoundConfig           = "processBoundConfig"
	EventPushing                 = "eventPushing"
	SystemFunctionality          = "systemFunctionality"

	NetCollector = "netCollector"
	NetDevice    = "netDevice"
	NetProperty  = "netProperty"
	NetReport    = "netReport"
)

type ResourceDescribe struct {
	Type    ResourceType
	Actions []Action
}

var (
	BusinessDescribe = ResourceDescribe{
		Type:    Business,
		Actions: []Action{Create, Update, Delete, FindMany},
	}

	ModelDescribe = ResourceDescribe{
		Type:    Model,
		Actions: []Action{Create, Update, Delete, FindMany},
	}

	ModelModuleDescribe = ResourceDescribe{
		Type:    ModelModule,
		Actions: []Action{Create, Update, Delete, FindMany},
	}

	ModelSetDescribe = ResourceDescribe{
		Type:    ModelSet,
		Actions: []Action{Create, Update, Delete, FindMany, DeleteMany},
	}

	MainlineModelDescribe = ResourceDescribe{
		Type:    MainlineModel,
		Actions: []Action{Create, Delete, Find},
	}

	MainlineModelTopologyDescribe = ResourceDescribe{
		Type:    MainlineModelTopology,
		Actions: []Action{Find},
	}

	MainlineInstanceTopologyDescribe = ResourceDescribe{
		Type:    MainlineInstanceTopology,
		Actions: []Action{Find},
	}

	AssociationTypeDescribe = ResourceDescribe{
		Type:    AssociationType,
		Actions: []Action{FindMany, Create, Update, Delete},
	}

	ModelAssociationDescribe = ResourceDescribe{
		Type:    ModelAssociation,
		Actions: []Action{FindMany, Create, Update, Delete},
	}

	ModelInstanceAssociationDescribe = ResourceDescribe{
		Type:    ModelInstanceAssociation,
		Actions: []Action{FindMany, Create, Delete},
	}

	ModelInstanceDescribe = ResourceDescribe{
		Type: ModelInstance,
		Actions: []Action{
			DeleteMany,
			FindMany,
			UpdateMany,
			Create,
			Find,
			Update,
			DeleteMany,
			Delete,
			// the following actions is the host actions for only.
			MoveResPoolHostToBizIdleModule,
			MoveHostToBizFaultModule,
			MoveHostToBizIdleModule,
			MoveHostFromModuleToResPool,
			MoveHostToAnotherBizModule,
			CleanHostInSetOrModule,
			MoveHostsToBusinessOrModule,
			AddHostToResourcePool,
			MoveHostToModule,
		},
	}

	ModelInstanceTopologyDescribe = ResourceDescribe{
		Type:    ModelInstanceTopology,
		Actions: []Action{Find, FindMany},
	}

	ModelTopologyDescribe = ResourceDescribe{
		Type:    ModelTopology,
		Actions: []Action{Find, Update},
	}

	ModelClassificationDescribe = ResourceDescribe{
		Type:    ModelClassification,
		Actions: []Action{FindMany, Create, Update, Delete},
	}

	ModelAttributeGroupDescribe = ResourceDescribe{
		Type:    ModelAttributeGroup,
		Actions: []Action{Find, Create, Delete},
	}

	ModelAttributeDescribe = ResourceDescribe{
		Type:    ModelAttribute,
		Actions: []Action{Find, Create, Update, Delete},
	}

	ModelUniqueDescribe = ResourceDescribe{
		Type:    ModelUnique,
		Actions: []Action{FindMany, Create, Update, Delete},
	}

	HostUserCustomDescribe = ResourceDescribe{
		Type:    HostUserCustom,
		Actions: []Action{Find, FindMany, Create, Update, Delete},
	}

	HostFavoriteDescribe = ResourceDescribe{
		Type:    HostFavorite,
		Actions: []Action{FindMany, Create, Update, Delete, DeleteMany},
	}

	ProcessDescribe = ResourceDescribe{
		Type:    Process,
		Actions: []Action{Create, Find, FindMany, Delete, DeleteMany, Update, UpdateMany, Create},
	}

	NetDataCollectorDescribe = ResourceDescribe{
		Type:    NetDataCollector,
		Actions: []Action{Find, FindMany, Update, UpdateMany, DeleteMany, Create, DeleteMany},
	}
)
