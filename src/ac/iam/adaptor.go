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
	"fmt"
	"strings"

	"configcenter/src/ac/meta"
	"configcenter/src/common/metadata"
	"configcenter/src/scene_server/auth_server/sdk/types"
)

var NotEnoughLayer = fmt.Errorf("not enough layer")

func AdaptAuthOptions(a *meta.ResourceAttribute) (ActionID, []types.Resource, error) {

	var action ActionID

	action, err := ConvertResourceAction(a.Type, a.Action, a.BusinessID)
	if err != nil {
		return "", nil, err
	}

	// convert different cmdb resource's to resource's registered to iam
	rscType, err := ConvertResourceType(a.Type, a.BusinessID)
	if err != nil {
		return "", nil, err
	}

	resource, err := GenIamResource(action, *rscType, a)
	if err != nil {
		return "", nil, err
	}

	return action, resource, nil
}

func ConvertResourceType(resourceType meta.ResourceType, businessID int64) (*TypeID, error) {
	var iamResourceType TypeID
	switch resourceType {
	case meta.Business:
		iamResourceType = Business
	case meta.Model,
		meta.ModelUnique,
		meta.ModelAttributeGroup:
		iamResourceType = SysModel
	case meta.ModelAttribute:
		if businessID > 0 {
			iamResourceType = BizCustomField
		} else {
			iamResourceType = SysModel
		}
	case meta.ModelModule, meta.ModelSet, meta.MainlineInstance, meta.MainlineInstanceTopology:
		iamResourceType = BizTopology
	case meta.MainlineModel, meta.ModelTopology:
	case meta.ModelClassification:
		iamResourceType = SysModelGroup
	case meta.AssociationType:
		iamResourceType = SysAssociationType
	case meta.ModelAssociation:
		iamResourceType = SysModel
	case meta.ModelInstanceAssociation:
		iamResourceType = SysInstance
	case meta.MainlineModelTopology:
	case meta.ModelInstance:
		iamResourceType = SysInstance
	case meta.ModelInstanceTopology:
		iamResourceType = SkipType
	case meta.CloudAreaInstance:
		iamResourceType = SysCloudArea
	case meta.HostInstance:
		iamResourceType = Host
	case meta.HostFavorite:
		iamResourceType = SkipType
	case meta.Process:
		iamResourceType = BizProcessServiceInstance
	case meta.DynamicGrouping:
		iamResourceType = BizCustomQuery
	case meta.AuditLog:
		iamResourceType = SysAuditLog
	case meta.SystemBase:
	case meta.UserCustom:
		iamResourceType = UserCustom
	case meta.NetDataCollector:
		return nil, fmt.Errorf("unsupported resource type: %s", resourceType)
	case meta.ProcessServiceTemplate:
		iamResourceType = BizProcessServiceTemplate
	case meta.ProcessServiceCategory:
		iamResourceType = BizProcessServiceCategory
	case meta.ProcessServiceInstance:
		iamResourceType = BizProcessServiceInstance
	case meta.BizTopology:
		iamResourceType = BizTopology
	case meta.SetTemplate:
		iamResourceType = BizSetTemplate
	case meta.OperationStatistic:
		iamResourceType = SysOperationStatistic
	case meta.HostApply:
		iamResourceType = BizHostApply
	case meta.ResourcePoolDirectory:
		iamResourceType = SysResourcePoolDirectory
	case meta.CloudAccount:
		iamResourceType = SysCloudAccount
	case meta.CloudResourceTask:
		iamResourceType = SysCloudResourceTask
	case meta.EventWatch:
		iamResourceType = SysEventWatch
	case meta.ConfigAdmin:
	case meta.SystemConfig:
	default:
		return nil, fmt.Errorf("unsupported resource type: %s", resourceType)
	}

	return &iamResourceType, nil
}

func ConvertResourceAction(resourceType meta.ResourceType, action meta.Action, businessID int64) (ActionID, error) {
	if action == meta.SkipAction {
		return Skip, nil
	}

	convertAction := action
	switch action {
	case meta.CreateMany:
		convertAction = meta.Create
	case meta.FindMany:
		convertAction = meta.Find
	case meta.DeleteMany:
		convertAction = meta.Delete
	case meta.UpdateMany:
		convertAction = meta.Update
	}

	if resourceType == meta.ModelAttribute || resourceType == meta.ModelAttributeGroup {
		if convertAction == meta.Delete || convertAction == meta.Update || convertAction == meta.Create {
			if businessID > 0 {
				return EditBusinessCustomField, nil
			} else {
				return EditSysModel, nil
			}
		}
	}

	if resourceType == meta.HostInstance && convertAction == meta.Update {
		if businessID > 0 {
			return EditBusinessHost, nil
		} else {
			return EditResourcePoolHost, nil
		}
	}

	if _, exist := resourceActionMap[resourceType]; exist {
		actionID, ok := resourceActionMap[resourceType][convertAction]
		if ok && actionID != Unsupported {
			return actionID, nil
		}
	}
	return Unsupported, fmt.Errorf("unsupported type %s action: %s", resourceType, action)
}

var resourceActionMap = map[meta.ResourceType]map[meta.Action]ActionID{
	meta.ModelInstance: {
		meta.Delete:   DeleteSysInstance,
		meta.Update:   EditSysInstance,
		meta.Create:   CreateSysInstance,
		meta.Find:     Skip,
		meta.FindMany: Skip,
	},
	meta.ModelAttributeGroup: {
		meta.Delete:   EditSysModel,
		meta.Update:   EditSysModel,
		meta.Create:   EditSysModel,
		meta.Find:     Skip,
		meta.FindMany: Skip,
	},
	meta.ModelUnique: {
		meta.Delete:   EditSysModel,
		meta.Update:   EditSysModel,
		meta.Create:   EditSysModel,
		meta.Find:     Skip,
		meta.FindMany: Skip,
	},
	meta.Business: {
		meta.Archive:              ArchiveBusiness,
		meta.Create:               CreateBusiness,
		meta.Update:               EditBusiness,
		meta.Find:                 FindBusiness,
		meta.ViewBusinessResource: ViewBusinessResource,
	},
	meta.DynamicGrouping: {
		meta.Delete:   DeleteBusinessCustomQuery,
		meta.Update:   EditBusinessCustomQuery,
		meta.Create:   CreateBusinessCustomQuery,
		meta.Find:     ViewBusinessResource,
		meta.FindMany: ViewBusinessResource,
		meta.Execute:  ViewBusinessResource,
	},
	meta.MainlineModel: {
		meta.Find:   Skip,
		meta.Create: EditBusinessLayer,
		meta.Delete: EditBusinessLayer,
	},
	meta.ModelTopology: {
		meta.Find:   EditModelTopologyView,
		meta.Update: EditModelTopologyView,
	},
	meta.MainlineModelTopology: {
		meta.Find: Skip,
	},
	meta.Process: {
		meta.Find:   Skip,
		meta.Create: EditBusinessServiceInstance,
		meta.Delete: EditBusinessServiceInstance,
		meta.Update: EditBusinessServiceInstance,
	},
	meta.HostInstance: {
		meta.MoveResPoolHostToBizIdleModule: ResourcePoolHostTransferToBusiness,
		meta.MoveResPoolHostToDirectory:     ResourcePoolHostTransferToDirectory,
		meta.MoveBizHostFromModuleToResPool: BusinessHostTransferToResourcePool,
		meta.AddHostToResourcePool:          CreateResourcePoolHost,
		meta.Create:                         CreateResourcePoolHost,
		meta.Delete:                         DeleteResourcePoolHost,
		meta.MoveHostToAnotherBizModule:     HostTransferAcrossBusiness,
		meta.Find:                           Skip,
		meta.FindMany:                       Skip,
	},
	meta.ProcessServiceCategory: {
		meta.Delete: DeleteBusinessServiceCategory,
		meta.Update: EditBusinessServiceCategory,
		meta.Create: CreateBusinessServiceCategory,
		meta.Find:   Skip,
	},
	meta.ProcessServiceInstance: {
		meta.Delete:   DeleteBusinessServiceInstance,
		meta.Update:   EditBusinessServiceInstance,
		meta.Create:   CreateBusinessServiceInstance,
		meta.Find:     Skip,
		meta.FindMany: Skip,
	},
	meta.ProcessServiceTemplate: {
		meta.Delete:   DeleteBusinessServiceTemplate,
		meta.Update:   EditBusinessServiceTemplate,
		meta.Create:   CreateBusinessServiceTemplate,
		meta.Find:     Skip,
		meta.FindMany: Skip,
	},
	meta.SetTemplate: {
		meta.Delete:   DeleteBusinessSetTemplate,
		meta.Update:   EditBusinessSetTemplate,
		meta.Create:   CreateBusinessSetTemplate,
		meta.Find:     Skip,
		meta.FindMany: Skip,
	},
	meta.ModelModule: {
		meta.Delete:   DeleteBusinessTopology,
		meta.Update:   EditBusinessTopology,
		meta.Create:   CreateBusinessTopology,
		meta.Find:     Skip,
		meta.FindMany: Skip,
	},
	meta.ModelSet: {
		meta.Delete:   DeleteBusinessTopology,
		meta.Update:   EditBusinessTopology,
		meta.Create:   CreateBusinessTopology,
		meta.Find:     Skip,
		meta.FindMany: Skip,
	},
	meta.MainlineInstance: {
		meta.Delete:   DeleteBusinessTopology,
		meta.Update:   EditBusinessTopology,
		meta.Create:   CreateBusinessTopology,
		meta.Find:     Skip,
		meta.FindMany: Skip,
	},
	meta.MainlineInstanceTopology: {
		meta.Delete: Skip,
		meta.Update: Skip,
		meta.Create: Skip,
		meta.Find:   Skip,
	},
	meta.HostApply: {
		meta.Create: EditBusinessHostApply,
		meta.Update: EditBusinessHostApply,
		meta.Delete: EditBusinessHostApply,
		meta.Find:   Skip,
	},
	meta.ResourcePoolDirectory: {
		meta.Delete:                DeleteResourcePoolDirectory,
		meta.Update:                EditResourcePoolDirectory,
		meta.Create:                CreateResourcePoolDirectory,
		meta.AddHostToResourcePool: CreateResourcePoolHost,
		meta.Find:                  Skip,
	},
	meta.CloudAreaInstance: {
		meta.Delete:   DeleteCloudArea,
		meta.Update:   EditCloudArea,
		meta.Create:   CreateCloudArea,
		meta.Find:     Skip,
		meta.FindMany: Skip,
	},
	meta.CloudAccount: {
		meta.Delete:   DeleteCloudAccount,
		meta.Update:   EditCloudAccount,
		meta.Create:   CreateCloudAccount,
		meta.Find:     FindCloudAccount,
		meta.FindMany: FindCloudAccount,
	},
	meta.CloudResourceTask: {
		meta.Delete: DeleteCloudResourceTask,
		meta.Update: EditCloudResourceTask,
		meta.Create: CreateCloudResourceTask,
		meta.Find:   FindCloudResourceTask,
	},
	meta.Model: {
		meta.Delete:   DeleteSysModel,
		meta.Update:   EditSysModel,
		meta.Create:   CreateSysModel,
		meta.Find:     Skip,
		meta.FindMany: Skip,
	},
	meta.AssociationType: {
		meta.Delete:   DeleteAssociationType,
		meta.Update:   EditAssociationType,
		meta.Create:   CreateAssociationType,
		meta.Find:     Skip,
		meta.FindMany: Skip,
	},
	meta.ModelClassification: {
		meta.Delete:   DeleteModelGroup,
		meta.Update:   EditModelGroup,
		meta.Create:   CreateModelGroup,
		meta.Find:     Skip,
		meta.FindMany: Skip,
	},
	meta.OperationStatistic: {
		meta.Create:   EditOperationStatistic,
		meta.Delete:   EditOperationStatistic,
		meta.Update:   EditOperationStatistic,
		meta.Find:     FindOperationStatistic,
		meta.FindMany: FindOperationStatistic,
	},
	meta.AuditLog: {
		meta.Find:     FindAuditLog,
		meta.FindMany: FindAuditLog,
	},
	meta.SystemBase: {
		meta.ModelTopologyView:      EditModelTopologyView,
		meta.ModelTopologyOperation: EditBusinessLayer,
	},
	meta.EventWatch: {
		meta.WatchHost:             WatchHostEvent,
		meta.WatchHostRelation:     WatchHostRelationEvent,
		meta.WatchBiz:              WatchBizEvent,
		meta.WatchSet:              WatchSetEvent,
		meta.WatchModule:           WatchModuleEvent,
		meta.WatchProcess:          WatchProcessEvent,
		meta.WatchCommonInstance:   WatchCommonInstanceEvent,
		meta.WatchMainlineInstance: WatchMainlineInstanceEvent,
	},
	meta.UserCustom: {
		meta.Find:   Skip,
		meta.Update: Skip,
		meta.Delete: Skip,
		meta.Create: Skip,
	},
	meta.ModelInstanceAssociation: {
		meta.Find:     Skip,
		meta.FindMany: Skip,
		meta.Update:   EditSysInstance,
		meta.Delete:   EditSysInstance,
		meta.Create:   EditSysInstance,
	},
	meta.ModelAssociation: {
		meta.Find:     Skip,
		meta.FindMany: Skip,
		meta.Update:   EditSysModel,
		meta.Delete:   EditSysModel,
		meta.Create:   EditSysModel,
	},
	meta.ModelInstanceTopology: {
		meta.Find:   Skip,
		meta.Update: Skip,
		meta.Delete: Skip,
		meta.Create: Skip,
	},
	meta.ModelAttribute: {
		meta.Find:   Skip,
		meta.Update: EditSysModel,
		meta.Delete: DeleteSysModel,
		meta.Create: CreateSysModel,
	},
	meta.HostFavorite: {
		meta.Find:   Skip,
		meta.Update: Skip,
		meta.Delete: Skip,
		meta.Create: Skip,
	},

	meta.ProcessTemplate: {
		meta.Find:   Skip,
		meta.Delete: DeleteBusinessServiceTemplate,
		meta.Update: EditBusinessServiceTemplate,
		meta.Create: CreateBusinessServiceTemplate,
	},
	meta.BizTopology: {
		meta.Find:   Skip,
		meta.Update: EditBusinessTopology,
		meta.Delete: DeleteBusinessTopology,
		meta.Create: CreateBusinessTopology,
	},
	// unsupported resource actions for now
	meta.NetDataCollector: {
		meta.Find:   Unsupported,
		meta.Update: Unsupported,
		meta.Delete: Unsupported,
		meta.Create: Unsupported,
	},
	meta.InstallBK: {
		meta.Update: Skip,
	},
	// TODO: confirm this
	meta.SystemConfig: {
		meta.FindMany: Skip,
		meta.Find:     Skip,
		meta.Update:   Skip,
		meta.Delete:   Skip,
		meta.Create:   Skip,
	},
	meta.ConfigAdmin: {
		meta.Find:   Skip,
		meta.Update: GlobalSettings,
		// unsupported action
		meta.Delete: Unsupported,
		// unsupported action
		meta.Create: Unsupported,
	},
}

func ParseIamPathToAncestors(iamPath []string) ([]metadata.IamResourceInstance, error) {
	instances := make([]metadata.IamResourceInstance, 0)
	for _, path := range iamPath {
		pathItemArr := strings.Split(strings.Trim(path, "/"), "/")
		for _, pathItem := range pathItemArr {
			typeAndID := strings.Split(pathItem, ",")
			if len(typeAndID) != 2 {
				return nil, fmt.Errorf("pathItem %s invalid", pathItem)
			}
			id := typeAndID[1]
			if id == "*" {
				continue
			}
			instances = append(instances, metadata.IamResourceInstance{
				Type:     typeAndID[0],
				TypeName: ResourceTypeIDMap[TypeID(typeAndID[0])],
				ID:       id,
			})
		}
	}
	return instances, nil
}
