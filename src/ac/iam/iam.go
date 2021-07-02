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
	"context"
	"net/http"

	"configcenter/src/ac/meta"
	"configcenter/src/apimachinery"
	"configcenter/src/apimachinery/authserver"
	"configcenter/src/apimachinery/flowctrl"
	"configcenter/src/apimachinery/rest"
	"configcenter/src/apimachinery/util"
	"configcenter/src/common/auth"
	"configcenter/src/common/blog"
	"configcenter/src/common/metadata"
	commonutil "configcenter/src/common/util"
	"configcenter/src/scene_server/auth_server/sdk/types"

	"github.com/prometheus/client_golang/prometheus"
)

type IAM struct {
	Client iamClientInterface
}

func NewIAM(tls *util.TLSClientConfig, cfg AuthConfig, reg prometheus.Registerer) (*IAM, error) {
	blog.V(5).Infof("new iam with parameters tls: %+v, cfg: %+v", tls, cfg)
	if !auth.EnableAuthorize() {
		return new(IAM), nil
	}
	client, err := util.NewClient(tls)
	if err != nil {
		return nil, err
	}
	c := &util.Capability{
		Client: client,
		Discover: &iamDiscovery{
			servers: cfg.Address,
		},
		Throttle: flowctrl.NewRateLimiter(5000, 5000),
		Mock: util.MockInfo{
			Mocked: false,
		},
		MetricOpts: util.MetricOption{Register: reg},
	}

	header := http.Header{}
	header.Set("Content-Type", "application/json")
	header.Set("Accept", "application/json")
	header.Set(iamAppCodeHeader, cfg.AppCode)
	header.Set(iamAppSecretHeader, cfg.AppSecret)

	return &IAM{
		Client: NewIAMClient(&IAMClientCfg{
			Config:      cfg,
			Client:      rest.NewRESTClient(c, ""),
			BasicHeader: header,
		}),
	}, nil
}

func (i IAM) RegisterSystem(ctx context.Context, host string, objects []metadata.Object) error {
	if !auth.EnableAuthorize() {
		return nil
	}

	systemResp, err := i.Client.GetSystemInfo(ctx, []SystemQueryField{})
	if err != nil && err != ErrNotFound {
		blog.Errorf("get system info failed, error: %s", err.Error())
		return err
	}
	if systemResp == nil {
		systemResp = new(SystemResp)
	}

	// if iam cmdb system has not been registered, register system
	if err == ErrNotFound {
		sys := System{
			ID:          SystemIDCMDB,
			Name:        SystemNameCMDB,
			EnglishName: SystemNameCMDBEn,
			Clients:     SystemIDCMDB,
			ProviderConfig: &SysConfig{
				Host: host,
				Auth: "basic",
			},
		}
		if err = i.Client.RegisterSystem(ctx, sys); err != nil {
			blog.ErrorJSON("register system %s failed, error: %s", sys, err.Error())
			return err
		}
		blog.V(5).Infof("register new system %+v succeed", sys)
	} else if systemResp.Data.BaseInfo.ProviderConfig == nil || systemResp.Data.BaseInfo.ProviderConfig.Host != host {
		// if iam registered cmdb system has no ProviderConfig
		// or registered host config is different with current host config, update system host config
		if err = i.Client.UpdateSystemConfig(ctx, &SysConfig{Host: host}); err != nil {
			blog.Errorf("update system host %s config failed, error: %s", host, err.Error())
			return err
		}
		if systemResp.Data.BaseInfo.ProviderConfig == nil {
			blog.V(5).Infof("update system host to %s succeed", systemResp.Data.BaseInfo.ProviderConfig.Host, host)
		} else {
			blog.V(5).Infof("update system host %s to %s succeed", systemResp.Data.BaseInfo.ProviderConfig.Host, host)
		}
	}

	existResourceTypeMap := make(map[TypeID]bool)
	removedResourceTypeMap := make(map[TypeID]struct{})
	newResourceTypes := make([]ResourceType, 0)
	for _, resourceType := range systemResp.Data.ResourceTypes {
		existResourceTypeMap[resourceType.ID] = true
		removedResourceTypeMap[resourceType.ID] = struct{}{}
	}
	for _, resourceType := range GenerateResourceTypes(objects) {
		// registered resource type exist in current resource types, should not be removed
		delete(removedResourceTypeMap, resourceType.ID)
		// if current resource type is registered, update it, or else register it
		if existResourceTypeMap[resourceType.ID] {
			if err = i.Client.UpdateResourcesTypes(ctx, resourceType); err != nil {
				blog.ErrorJSON("update resource type %s failed, error: %s, input resource type: %s",
					resourceType.ID, err.Error(), resourceType)
				return err
			}
		} else {
			newResourceTypes = append(newResourceTypes, resourceType)
		}
	}

	existInstanceSelectionMap := make(map[InstanceSelectionID]bool)
	removedInstanceSelectionMap := make(map[InstanceSelectionID]struct{})
	newInstanceSelections := make([]InstanceSelection, 0)
	for _, instanceSelection := range systemResp.Data.InstanceSelections {
		existInstanceSelectionMap[instanceSelection.ID] = true
		removedInstanceSelectionMap[instanceSelection.ID] = struct{}{}
	}

	for _, resourceType := range GenerateInstanceSelections(objects) {
		// registered instance selection exist in current instance selections, should not be removed
		delete(removedInstanceSelectionMap, resourceType.ID)
		// if current instance selection is registered, update it, or else register it
		if existInstanceSelectionMap[resourceType.ID] {
			if err = i.Client.UpdateInstanceSelection(ctx, resourceType); err != nil {
				blog.ErrorJSON("update instance selection %s failed, error: %s, input resource type: %s",
					resourceType.ID, err.Error(), resourceType)
				return err
			}
		} else {
			newInstanceSelections = append(newInstanceSelections, resourceType)
		}
	}

	existResourceActionMap := make(map[ActionID]bool)
	removedResourceActionMap := make(map[ActionID]struct{})
	newResourceActions := make([]ResourceAction, 0)
	for _, resourceAction := range systemResp.Data.Actions {
		existResourceActionMap[resourceAction.ID] = true
		removedResourceActionMap[resourceAction.ID] = struct{}{}
	}
	for _, resourceAction := range GenerateActions(objects) {
		// registered resource action exist in current resource actions, should not be removed
		delete(removedResourceActionMap, resourceAction.ID)
		// if current resource action is registered, update it, or else register it
		if existResourceActionMap[resourceAction.ID] {
			if err = i.Client.UpdateAction(ctx, resourceAction); err != nil {
				blog.ErrorJSON("update resource action %s failed, error: %s, input resource action: %s",
					resourceAction.ID, err.Error(), resourceAction)
				return err
			}
		} else {
			newResourceActions = append(newResourceActions, resourceAction)
		}
	}

	// 因为资源间的依赖关系，删除和更新的顺序为 1.Action 2.InstanceSelection 3.ResourceType
	// 因为资源间的依赖关系，新建的顺序则反过来为 1.ResourceType 2.InstanceSelection 3.Action
	// ActionGroup依赖于Action，该资源的增删操作始终放在最后
	// 先删除资源，再新增资源，因为实例视图的名称在系统中是唯一的，如果不先删，同样名称的实例视图将创建失败
	// remove redundant actions, redundant instance selections and resource types one by one
	// when dependencies are all deleted
	if len(removedResourceActionMap) > 0 {
		removedResourceActionIDs := make([]ActionID, len(removedResourceActionMap))
		idx := 0
		// before deleting action, the dependent action polices must be deleted
		for resourceActionID := range removedResourceActionMap {
			if err = i.Client.DeleteActionPolicies(ctx, resourceActionID); err != nil {
				blog.Errorf("delete action %s policies failed, err: %v", resourceActionID, err)
				return err
			}

			removedResourceActionIDs[idx] = resourceActionID
			idx++
		}
		if err = i.Client.DeleteActions(ctx, removedResourceActionIDs); err != nil {
			blog.ErrorJSON("delete resource actions failed, error: %s, resource actions: %s", err.Error(), removedResourceActionIDs)
			return err
		}
	}

	if len(removedInstanceSelectionMap) > 0 {
		removedInstanceSelectionIDs := make([]InstanceSelectionID, len(removedInstanceSelectionMap))
		idx := 0
		for resourceActionID := range removedInstanceSelectionMap {
			removedInstanceSelectionIDs[idx] = resourceActionID
			idx++
		}
		if err = i.Client.DeleteInstanceSelections(ctx, removedInstanceSelectionIDs); err != nil {
			blog.ErrorJSON("delete instance selections failed, error: %s, instance selections: %s",
				err.Error(), removedInstanceSelectionIDs)
			return err
		}
	}

	if len(removedResourceTypeMap) > 0 {
		removedResourceTypeIDs := make([]TypeID, len(removedResourceTypeMap))
		idx := 0
		for resourceType := range removedResourceTypeMap {
			removedResourceTypeIDs[idx] = resourceType
			idx++
		}
		if err = i.Client.DeleteResourcesTypes(ctx, removedResourceTypeIDs); err != nil {
			blog.ErrorJSON("delete resource types failed, error: %s, resource types: %s",
				err.Error(), removedResourceTypeIDs)
			return err
		}
	}

	if len(newResourceTypes) > 0 {
		if err = i.Client.RegisterResourcesTypes(ctx, newResourceTypes); err != nil {
			blog.ErrorJSON("register resource types failed, error: %s, resource types: %s",
				err.Error(), newResourceTypes)
			return err
		}
	}

	if len(newInstanceSelections) > 0 {
		if err = i.Client.RegisterInstanceSelections(ctx, newInstanceSelections); err != nil {
			blog.ErrorJSON("register instance selections failed, error: %s, resource types: %s",
				err.Error(), newInstanceSelections)
			return err
		}
	}

	if len(newResourceActions) > 0 {
		if err = i.Client.RegisterActions(ctx, newResourceActions); err != nil {
			blog.ErrorJSON("register resource actions failed, error: %s, resource actions: %s",
				err.Error(), newResourceActions)
			return err
		}
	}

	// register or update resource action groups
	actionGroups := GenerateActionGroups(objects)
	if len(systemResp.Data.ActionGroups) == 0 {
		if err = i.Client.RegisterActionGroups(ctx, actionGroups); err != nil {
			blog.ErrorJSON("register action groups failed, error: %s, action groups: %s", err.Error(), actionGroups)
			return err
		}
	} else {
		if err = i.Client.UpdateActionGroups(ctx, actionGroups); err != nil {
			blog.ErrorJSON("update action groups failed, error: %s, action groups: %s", err.Error(), actionGroups)
			return err
		}
	}

	// register or update resource creator actions
	resourceCreatorActions := GenerateResourceCreatorActions()
	if len(systemResp.Data.ResourceCreatorActions.Config) == 0 {
		if err = i.Client.RegisterResourceCreatorActions(ctx, resourceCreatorActions); err != nil {
			blog.ErrorJSON("register resource creator actions failed, error: %s, resource creator actions: %s",
				err.Error(), resourceCreatorActions)
			return err
		}
	} else {
		if err = i.Client.UpdateResourceCreatorActions(ctx, resourceCreatorActions); err != nil {
			blog.ErrorJSON("update resource creator actions failed, error: %s, resource creator actions: %s",
				err.Error(), resourceCreatorActions)
			return err
		}
	}

	//register or update common actions
	commonActions := GenerateCommonActions()
	if len(systemResp.Data.CommonActions) == 0 {
		if err = i.Client.RegisterCommonActions(ctx, commonActions); err != nil {
			blog.ErrorJSON("register common actions failed, error: %s, common actions: %s", err.Error(), commonActions)
			return err
		}
	} else {
		if err = i.Client.UpdateCommonActions(ctx, commonActions); err != nil {
			blog.ErrorJSON("update common actions failed, error: %s, common actions: %s", err.Error(), commonActions)
			return err
		}
	}

	return nil
}

// SyncIAMSysInstances sync system instances between CMDB and IAM
// it check the difference of system instances resource between CMDB and IAM
// if they have difference, sync and make them same
func (i IAM) SyncIAMSysInstances(ctx context.Context, objects []metadata.Object) error {
	rid := commonutil.ExtractRequestIDFromContext(ctx)

	fields := []SystemQueryField{FieldResourceTypes,
		FieldActions, FieldActionGroups, FieldInstanceSelections}
	iamResp, err := i.Client.GetSystemInfo(ctx, fields)
	if err != nil {
		blog.ErrorJSON("sync iam sysInstances failed, get system info error: %s, fields: %s, rid: %s",
			err, fields, rid)
		return err
	}

	// get the cmdb resources
	cmdbActions := genDynamicActions(objects)
	cmdbInstanceSelections := genDynamicInstanceSelections(objects)
	cmdbResourceTypes := genDynamicResourceTypes(objects)

	// compare resources between cmdb and iam
	addedActions, deletedActions := compareActions(cmdbActions, iamResp.Data.Actions)
	addedInstanceSelections, deletedInstanceSelections := compareInstanceSelections(cmdbInstanceSelections,
		iamResp.Data.InstanceSelections)
	addedResourceTypes, deletedResourceTypes := compareResourceTypes(cmdbResourceTypes, iamResp.Data.ResourceTypes)

	// 因为资源间的依赖关系，删除和更新的顺序为 1.Action 2.InstanceSelection 3.ResourceType
	// 因为资源间的依赖关系，新建的顺序则反过来为 1.ResourceType 2.InstanceSelection 3.Action
	// ActionGroup依赖于Action，该资源的增删操作始终放在最后
	// 先删除资源，再新增资源，因为实例视图的名称在系统中是唯一的，如果不先删，同样名称的实例视图将创建失败

	// delete unnecessary actions in iam
	if len(deletedActions) > 0 {
		blog.Infof("begin delete actions, count:%d, detail:%v, rid: %s", len(deletedActions), deletedActions, rid)

		// before deleting action, the dependent action polices must be deleted
		for _, actionID := range deletedActions {
			if err = i.Client.DeleteActionPolicies(ctx, actionID); err != nil {
				blog.Errorf("sync iam sysInstances failed, delete action %s policies err: %s, rid: %s",
					actionID, err, rid)
				return err
			}
		}

		if err := i.Client.DeleteActions(ctx, deletedActions); err != nil {
			blog.ErrorJSON("sync iam sysInstances failed, delete IAM actions error: %s, actions: %s, rid: %s",
				err, deletedActions, rid)
			return err
		}
	}

	// delete unnecessary InstanceSelections in iam
	if len(deletedInstanceSelections) > 0 {
		blog.Infof("begin delete instanceSelections, count:%d, detail:%v, rid: %s",
			len(deletedInstanceSelections), deletedInstanceSelections, rid)
		if err := i.Client.DeleteInstanceSelections(ctx, deletedInstanceSelections); err != nil {
			blog.ErrorJSON("sync iam sysInstances failed, delete instanceSelections error: %s, instanceSelections: %s,"+
				" rid: %s", err, deletedInstanceSelections, rid)
			return err
		}
	}

	// delete unnecessary ResourceTypes in iam
	if len(deletedResourceTypes) > 0 {
		blog.Infof("begin delete resourceTypes, count:%d, detail:%v, rid: %s",
			len(deletedResourceTypes), deletedResourceTypes, rid)
		if err := i.Client.DeleteResourcesTypes(ctx, deletedResourceTypes); err != nil {
			blog.ErrorJSON("sync iam sysInstances failed, delete resourceType error: %s, resourceType: %s, rid: %s",
				err, deletedResourceTypes, rid)
			return err
		}
	}

	// add cmdb ResourceTypes in iam
	if len(addedResourceTypes) > 0 {
		blog.Infof("begin add resourceTypes, count:%d, detail:%v, rid: %s",
			len(addedResourceTypes), addedResourceTypes, rid)
		if err := i.Client.RegisterResourcesTypes(ctx, addedResourceTypes); err != nil {
			blog.ErrorJSON("sync iam sysInstances failed, add resourceType error: %s, resourceType: %s, rid: %s",
				err, addedResourceTypes, rid)
			return err
		}
	}

	// add cmdb InstanceSelections in iam
	if len(addedInstanceSelections) > 0 {
		blog.Infof("begin add instanceSelections, count:%d, detail:%v, rid: %s",
			len(addedInstanceSelections), addedInstanceSelections, rid)
		if err := i.Client.RegisterInstanceSelections(ctx, addedInstanceSelections); err != nil {
			blog.ErrorJSON("sync iam sysInstances failed, add instanceSelections error: %s, instanceSelections: %s, "+
				"rid: %s", err, addedInstanceSelections, rid)
			return err
		}
	}

	// add cmdb actions in iam
	if len(addedActions) > 0 {
		blog.Infof("begin add actions, count:%d, detail:%v, rid: %s", len(addedActions), addedActions, rid)
		if err := i.Client.RegisterActions(ctx, addedActions); err != nil {
			blog.ErrorJSON("sync iam sysInstances failed, add IAM actions failed, error: %s, actions: %s, rid: %s",
				err, addedActions, rid)
			return err
		}
	}

	// update action_groups in iam
	if len(addedActions) > 0 || len(deletedActions) > 0 {
		cmdbActionGroups := GenerateActionGroups(objects)
		blog.Infof("begin update actionGroups")
		if err := i.Client.UpdateActionGroups(ctx, cmdbActionGroups); err != nil {
			blog.ErrorJSON("sync iam sysInstances failed, update actionGroups error: %s, actionGroups: %s, rid: %s",
				err, cmdbActionGroups, rid)
			return err
		}
	}

	return nil
}

// DeleteCMDBResource delete unnecessary CMDB resource from IAM
// it will  delete the resource if it exists on IAM
func (i IAM) DeleteCMDBResource(ctx context.Context, param *DeleteCMDBResourceParam, objects []metadata.Object) error {
	rid := commonutil.ExtractRequestIDFromContext(ctx)

	fields := []SystemQueryField{FieldResourceTypes,
		FieldActions, FieldActionGroups, FieldInstanceSelections}
	iamResp, err := i.Client.GetSystemInfo(ctx, fields)
	if err != nil {
		blog.ErrorJSON("sync iam sysInstances failed, get system info error: %s, fields: %s, rid: %s",
			err, fields, rid)
		return err
	}

	// get deleted actions
	deletedActions := getDeletedActions(param.ActionIDs, iamResp.Data.Actions)
	deletedInstanceSelections := getDeletedInstanceSelections(param.InstanceSelectionIDs,
		iamResp.Data.InstanceSelections)
	deletedResourceTypes := getDeletedResourceTypes(param.TypeIDs, iamResp.Data.ResourceTypes)

	// 因为资源间的依赖关系，删除的顺序为 1.Action 2.InstanceSelection 3.ResourceType
	// ActionGroup依赖于Action，该资源的增删操作始终放在最后

	// delete unnecessary actions in iam
	if len(deletedActions) > 0 {
		// before deleting action, the dependent action polices must be deleted
		for _, actionID := range deletedActions {
			if err = i.Client.DeleteActionPolicies(ctx, actionID); err != nil {
				blog.ErrorJSON("delete cmdb resource failed, delete action %s policies err: %s, rid: %s",
					actionID, err, rid)
				return err
			}
		}

		blog.Infof("begin delete actions, count:%d, detail:%v, rid: %s", len(deletedActions), deletedActions, rid)
		if err := i.Client.DeleteActions(ctx, deletedActions); err != nil {
			blog.ErrorJSON("delete cmdb resource failed, delete IAM actions error: %s, actions: %s, rid: %s",
				err, deletedActions, rid)
			return err
		}
	}

	// delete unnecessary InstanceSelections in iam
	if len(deletedInstanceSelections) > 0 {
		blog.Infof("begin delete instanceSelections, count:%d, detail:%v, rid: %s",
			len(deletedInstanceSelections), deletedInstanceSelections, rid)
		if err := i.Client.DeleteInstanceSelections(ctx, deletedInstanceSelections); err != nil {
			blog.ErrorJSON("delete cmdb resource failed, delete instanceSelections error: %s, instanceSelections: %s,"+
				"rid: %s", err, deletedInstanceSelections, rid)
			return err
		}
	}

	// delete unnecessary ResourceTypes in iam
	if len(deletedResourceTypes) > 0 {
		blog.Infof("begin delete resourceTypes, count:%d, detail:%v, rid: %s",
			len(deletedResourceTypes), deletedResourceTypes, rid)
		if err := i.Client.DeleteResourcesTypes(ctx, deletedResourceTypes); err != nil {
			blog.ErrorJSON("delete cmdb resource failed, delete resourceType error: %s, resourceType: %s, rid: %s",
				err, deletedResourceTypes, rid)
			return err
		}
	}

	// update action_groups in iam
	if len(deletedActions) > 0 {
		cmdbActionGroups := GenerateActionGroups(objects)
		blog.Infof("begin update actionGroups")
		if err := i.Client.UpdateActionGroups(ctx, cmdbActionGroups); err != nil {
			blog.ErrorJSON("delete cmdb resource failed, update actionGroups error: %s, actionGroups: %s, rid: %s",
				err, cmdbActionGroups, rid)
			return err
		}
	}

	return nil
}

// getDeletedActions get deleted actions
func getDeletedActions(cmdbActionIDs []ActionID, iamActions []ResourceAction) []ActionID {
	deletedActions := make([]ActionID, 0)
	iamActionMap := map[ActionID]struct{}{}
	for _, action := range iamActions {
		iamActionMap[action.ID] = struct{}{}
	}
	for _, actionID := range cmdbActionIDs {
		if _, ok := iamActionMap[actionID]; ok {
			deletedActions = append(deletedActions, actionID)
		}
	}

	return deletedActions
}

// getDeletedInstanceSelections get deleted instance selections
func getDeletedInstanceSelections(cmdbInstanceSelectionIDs []InstanceSelectionID,
	iamInstanceSelections []InstanceSelection) []InstanceSelectionID {
	deletedInstanceSelections := make([]InstanceSelectionID, 0)
	iamInstanceSelectionMap := map[InstanceSelectionID]struct{}{}
	for _, instanceSelection := range iamInstanceSelections {
		iamInstanceSelectionMap[instanceSelection.ID] = struct{}{}
	}
	for _, instanceSelectionID := range cmdbInstanceSelectionIDs {
		if _, ok := iamInstanceSelectionMap[instanceSelectionID]; ok {
			deletedInstanceSelections = append(deletedInstanceSelections, instanceSelectionID)
		}
	}

	return deletedInstanceSelections
}

// getDeletedResourceTypes get deleted resource types
func getDeletedResourceTypes(cmdbTypeIDs []TypeID, iamResourceTypes []ResourceType) []TypeID {
	deletedResourceTypes := make([]TypeID, 0)
	iamResourceTypeMap := map[TypeID]struct{}{}
	for _, resourceType := range iamResourceTypes {
		iamResourceTypeMap[resourceType.ID] = struct{}{}
	}
	for _, typeID := range cmdbTypeIDs {
		if _, ok := iamResourceTypeMap[typeID]; ok {
			deletedResourceTypes = append(deletedResourceTypes, typeID)
		}
	}

	return deletedResourceTypes
}

// compareActions compare actions between cmdb and iam
func compareActions(cmdbActions []ResourceAction, iamActions []ResourceAction) (
	addedActions []ResourceAction, deletedActionIDs []ActionID) {
	iamActionMap := map[ActionID]struct{}{}

	for _, action := range iamActions {
		if isIAMSysInstanceAction(action.ID) {
			iamActionMap[action.ID] = struct{}{}
		}
	}

	for _, action := range cmdbActions {
		if _, ok := iamActionMap[action.ID]; !ok {
			addedActions = append(addedActions, action)
		} else {
			delete(iamActionMap, action.ID)
		}
	}

	for actionID := range iamActionMap {
		deletedActionIDs = append(deletedActionIDs, actionID)
	}

	return addedActions, deletedActionIDs
}

// compareInstanceSelections compare instanceSelections between cmdb and iam
func compareInstanceSelections(cmdbInstanceSelections []InstanceSelection,
	iamInstanceSelections []InstanceSelection) (addInstanceSelection []InstanceSelection,
	deletedInstanceSelectionIDs []InstanceSelectionID) {
	iamInstanceSelectionMap := map[InstanceSelectionID]struct{}{}

	for _, instanceSelection := range iamInstanceSelections {
		if isIAMSysInstanceSelection(instanceSelection.ID) {
			iamInstanceSelectionMap[instanceSelection.ID] = struct{}{}
		}
	}

	for _, instanceSelection := range cmdbInstanceSelections {
		if _, ok := iamInstanceSelectionMap[instanceSelection.ID]; !ok {
			addInstanceSelection = append(addInstanceSelection, instanceSelection)
		} else {
			delete(iamInstanceSelectionMap, instanceSelection.ID)
		}
	}

	for instanceSelectionID := range iamInstanceSelectionMap {
		deletedInstanceSelectionIDs = append(deletedInstanceSelectionIDs, instanceSelectionID)
	}

	return addInstanceSelection, deletedInstanceSelectionIDs
}

// compareResourceTypes compare resourceTypes between cmdb and iam
func compareResourceTypes(cmdbResourceTypes []ResourceType, iamResourceTypes []ResourceType) (
	addedResourceTypes []ResourceType, deletedTypeIDs []TypeID) {
	iamResourceTypeMap := map[TypeID]struct{}{}

	for _, resourceType := range iamResourceTypes {
		if IsIAMSysInstance(resourceType.ID) {
			iamResourceTypeMap[resourceType.ID] = struct{}{}
		}
	}

	for _, resourceType := range cmdbResourceTypes {
		if _, ok := iamResourceTypeMap[resourceType.ID]; !ok {
			addedResourceTypes = append(addedResourceTypes, resourceType)
		} else {
			delete(iamResourceTypeMap, resourceType.ID)
		}
	}

	for typeID := range iamResourceTypeMap {
		deletedTypeIDs = append(deletedTypeIDs, typeID)
	}

	return addedResourceTypes, deletedTypeIDs
}

type authorizer struct {
	authClientSet authserver.AuthServerClientInterface
}

func NewAuthorizer(clientSet apimachinery.ClientSetInterface) *authorizer {
	return &authorizer{authClientSet: clientSet.AuthServer()}
}

func (a *authorizer) AuthorizeBatch(ctx context.Context, h http.Header, user meta.UserInfo,
	resources ...meta.ResourceAttribute) ([]types.Decision, error) {
	return a.authorizeBatch(ctx, h, true, user, resources...)
}

func (a *authorizer) AuthorizeAnyBatch(ctx context.Context, h http.Header, user meta.UserInfo,
	resources ...meta.ResourceAttribute) ([]types.Decision, error) {
	return a.authorizeBatch(ctx, h, false, user, resources...)
}

func (a *authorizer) authorizeBatch(ctx context.Context, h http.Header, exact bool, user meta.UserInfo,
	resources ...meta.ResourceAttribute) ([]types.Decision, error) {

	rid := commonutil.GetHTTPCCRequestID(h)

	opts, decisions, err := parseAttributesToBatchOptions(rid, user, resources...)
	if err != nil {
		return nil, err
	}

	// all resources are skipped
	if opts == nil {
		return decisions, nil
	}

	if blog.V(5) {
		blog.InfoJSON("auth options: %s, rid: %s", opts, rid)
	}

	var authDecisions []types.Decision
	if exact {
		authDecisions, err = a.authClientSet.AuthorizeBatch(ctx, h, opts)
		if err != nil {
			blog.ErrorJSON("authorize batch failed, err: %s, ops: %s, resources: %s, rid: %s",
				err, opts, resources,
				rid)
			return nil, err
		}
	} else {
		authDecisions, err = a.authClientSet.AuthorizeAnyBatch(ctx, h, opts)
		if err != nil {
			blog.ErrorJSON("authorize any batch failed, err: %s, ops: %s, resources: %s, rid: %s",
				err, opts, resources, rid)
			return nil, err
		}

	}

	index := 0
	for _, decision := range authDecisions {
		// skip resources' decisions are already set as authorized
		for decisions[index].Authorized {
			index++
		}
		decisions[index].Authorized = decision.Authorized
		index++
	}

	return decisions, nil
}

func parseAttributesToBatchOptions(rid string, user meta.UserInfo, resources ...meta.ResourceAttribute) (*types.AuthBatchOptions, []types.Decision, error) {
	if !auth.EnableAuthorize() {
		decisions := make([]types.Decision, len(resources))
		for i := range decisions {
			decisions[i].Authorized = true
		}
		return nil, decisions, nil
	}

	authBatchArr := make([]*types.AuthBatch, 0)
	decisions := make([]types.Decision, len(resources))
	for index, resource := range resources {

		// this resource should be skipped, do not need to verify in auth center.
		if resource.Action == meta.SkipAction {
			decisions[index].Authorized = true
			blog.V(5).Infof("skip authorization for resource: %+v, rid: %s", resource, rid)
			continue
		}

		action, resources, err := AdaptAuthOptions(&resource)
		if err != nil {
			blog.Errorf("adaptor cmdb resource to iam failed, err: %s, rid: %s", err, rid)
			return nil, nil, err
		}

		// this resource should be skipped, do not need to verify in auth center.
		if action == Skip {
			decisions[index].Authorized = true
			blog.V(5).Infof("skip authorization for resource: %+v, rid: %s", resource, rid)
			continue
		}

		authBatchArr = append(authBatchArr, &types.AuthBatch{
			Action:    types.Action{ID: string(action)},
			Resources: resources,
		})
	}

	// all resources are skipped
	if len(authBatchArr) == 0 {
		return nil, decisions, nil
	}

	ops := &types.AuthBatchOptions{
		System: SystemIDCMDB,
		Subject: types.Subject{
			Type: "user",
			ID:   user.UserName,
		},
		Batch: authBatchArr,
	}
	return ops, decisions, nil
}

func (a *authorizer) ListAuthorizedResources(ctx context.Context, h http.Header, input meta.ListAuthorizedResourcesParam) ([]string, error) {
	return a.authClientSet.ListAuthorizedResources(ctx, h, input)
}

func (a *authorizer) GetNoAuthSkipUrl(ctx context.Context, h http.Header, input *metadata.IamPermission) (string, error) {
	return a.authClientSet.GetNoAuthSkipUrl(ctx, h, input)
}

func (a *authorizer) GetPermissionToApply(ctx context.Context, h http.Header, input []meta.ResourceAttribute) (*metadata.IamPermission, error) {
	return a.authClientSet.GetPermissionToApply(ctx, h, input)
}

func (a *authorizer) RegisterResourceCreatorAction(ctx context.Context, h http.Header, input metadata.IamInstanceWithCreator) (
	[]metadata.IamCreatorActionPolicy, error) {

	return a.authClientSet.RegisterResourceCreatorAction(ctx, h, input)
}

func (a *authorizer) BatchRegisterResourceCreatorAction(ctx context.Context, h http.Header, input metadata.IamInstancesWithCreator) (
	[]metadata.IamCreatorActionPolicy, error) {

	return a.authClientSet.BatchRegisterResourceCreatorAction(ctx, h, input)
}
