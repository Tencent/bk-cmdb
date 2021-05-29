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
	"fmt"
	"net/http"

	"configcenter/src/apimachinery"
	"configcenter/src/common"
	"configcenter/src/common/blog"
	"configcenter/src/common/metadata"
)

type viewer struct {
	client apimachinery.ClientSetInterface
	iam *IAM
}

func NewViewer(client apimachinery.ClientSetInterface, iam *IAM) *viewer {
	return &viewer{
		client: client,
		iam: iam,
	}
}


// CreateView create iam view for a object
func (v *viewer) CreateView(ctx context.Context, header http.Header, objects []metadata.Object) error {
	// register order: 1.Action 2.InstanceSelection 3.ResourceType 4.ActionGroup
	if err := v.registerModelResourceTypes(ctx, objects); err != nil {
		return err
	}

	if err := v.registerModelInstanceSelections(ctx, objects); err != nil {
		return err
	}

	if err := v.registerModelActions(ctx, objects); err != nil {
		return err
	}

	if err := v.updateModelActionGroups(ctx, header); err != nil {
		return err
	}

	return nil
}

// DeleteView delete iam view for a object
func (v *viewer) DeleteView(ctx context.Context, header http.Header, objects []metadata.Object) error {
	// unregister order: 1.ResourceType 2.InstanceSelection 3.Action 4.ActionGroup
	if err := v.unregisterModelResourceTypes(ctx, objects); err != nil {
		return err
	}

	if err := v.unregisterModelInstanceSelections(ctx, objects); err != nil {
		return err
	}

	if err := v.unregisterModelActions(ctx, objects); err != nil {
		return err
	}

	if err := v.updateModelActionGroups(ctx, header); err != nil {
		return err
	}

	return nil
}

// registerModelResourceTypes register resource types for models
func (v *viewer) registerModelResourceTypes(ctx context.Context, objects []metadata.Object) error {
	resourceTypes := GenDynamicResourceTypes(objects)
	if err := v.iam.client.RegisterResourcesTypes(ctx, resourceTypes); err != nil {
		blog.ErrorJSON("register resourceTypes failed, error: %s, objects: %s, resourceTypes: %s",
			err.Error(), objects, resourceTypes)
		return err
	}

	return nil
}

// unregisterModelResourceTypes unregister resourceTypes for models
func (v *viewer) unregisterModelResourceTypes(ctx context.Context,	objects []metadata.Object) error {
	typeIDs := []TypeID{}
	resourceTypes := GenDynamicResourceTypes(objects)
	for _, resourceType := range resourceTypes {
		typeIDs = append(typeIDs, resourceType.ID)
	}
	if err := v.iam.client.DeleteResourcesTypes(ctx, typeIDs); err != nil {
		blog.ErrorJSON("unregister resourceTypes failed, error: %s, objects: %s, resourceTypes: %s",
			err.Error(), objects, resourceTypes)
		return err
	}

	return nil
}

// registerModelInstanceSelections register instanceSelections for models
func (v *viewer) registerModelInstanceSelections(ctx context.Context,	objects []metadata.Object) error {
	instanceSelections := GenDynamicInstanceSelections(objects)
	if err := v.iam.client.RegisterInstanceSelections(ctx, instanceSelections); err != nil {
		blog.ErrorJSON("register instanceSelections failed, error: %s, objects: %s, instanceSelections: %s",
			err.Error(), objects,
			instanceSelections)
		return err
	}

	return nil
}

// unregisterModelInstanceSelections unregister instanceSelections for models
func (v *viewer) unregisterModelInstanceSelections(ctx context.Context,	objects []metadata.Object) error {
	instanceSelectionIDs := []InstanceSelectionID{}
	instanceSelections := GenDynamicInstanceSelections(objects)
	for _, instanceSelection := range instanceSelections {
		instanceSelectionIDs = append(instanceSelectionIDs, instanceSelection.ID)
	}
	if err := v.iam.client.DeleteInstanceSelections(ctx, instanceSelectionIDs); err != nil {
		blog.ErrorJSON("unregister instanceSelections failed, error: %s, objects: %s, instanceSelections: %s",
			err.Error(), objects,
			instanceSelections)
		return err
	}

	return nil
}

// registerModelActions register actions for models
func (v *viewer) registerModelActions(ctx context.Context,	objects []metadata.Object) error {
	actions := GenDynamicActions(objects)
	if err := v.iam.client.RegisterActions(ctx, actions); err != nil {
		blog.ErrorJSON("register actions failed, error: %s, objects: %s, actions: %s", err.Error(), objects,
			actions)
		return err
	}

	return nil
}

// unregisterModelActions unregister actions for models
func (v *viewer) unregisterModelActions(ctx context.Context,	objects []metadata.Object) error {
	actionIDs := []ActionID{}
	for _, obj := range objects {
		actionIDs = append(actionIDs, GenDynamicActionIDs(obj)...)
	}
	if err := v.iam.client.DeleteActions(ctx, actionIDs); err != nil {
		blog.ErrorJSON("unregister actions failed, error: %s, objects: %s, actionIDs: %s", err.Error(),
			objects, actionIDs)
		return err
	}

	return nil
}

// updateModelActionGroups update actionGroups for models
func (v *viewer) updateModelActionGroups(ctx context.Context, header http.Header) error {
	// for now, the update api can only support full update, not incremental update

	objects, err := v.GetCustomObjects(ctx, header)
	if err != nil {
		blog.Errorf("get custom objects failed, err: %s:%s", err.Error())
		return err
	}
	actionGroups := GenerateActionGroups(objects)

	if err := v.iam.client.UpdateActionGroups(ctx, actionGroups); err != nil {
		blog.ErrorJSON("update actionGroups failed, error: %s, actionGroups: %s", err.Error(),
			actionGroups)
		return err
	}

	return nil
}

// GetCustomObjects get objects which are custom
func (v *viewer) GetCustomObjects(ctx context.Context, header http.Header) ([]metadata.Object, error) {
	resp, err := v.client.CoreService().Model().ReadModel(ctx, header, &metadata.QueryCondition{
		Fields: []string{common.BKObjIDField, common.BKObjNameField, common.BKFieldID},
		Page:   metadata.BasePage{Limit: common.BKNoLimit},
		Condition: map[string]interface{}{
			common.BKIsPre: false,
		},
	})
	if err != nil {
		return nil, fmt.Errorf("get custom objects failed, err: %+v", err)
	}
	if len(resp.Data.Info) == 0 {
		blog.Info("get custom objects failed, no custom objects were found")
	}

	objects := make([]metadata.Object, 0)
	for _, item := range resp.Data.Info {
		objects = append(objects, item.Spec)
	}

	return objects, nil
}
