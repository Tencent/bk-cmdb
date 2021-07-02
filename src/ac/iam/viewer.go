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
	"configcenter/src/common/auth"
	"configcenter/src/common/blog"
	"configcenter/src/common/metadata"
	"configcenter/src/common/util"
)

type viewer struct {
	client apimachinery.ClientSetInterface
	iam    *IAM
}

func NewViewer(client apimachinery.ClientSetInterface, iam *IAM) *viewer {
	return &viewer{
		client: client,
		iam:    iam,
	}
}

// CreateView create iam view for a object
func (v *viewer) CreateView(ctx context.Context, header http.Header, objects []metadata.Object) error {
	if !auth.EnableAuthorize() {
		return nil
	}

	// register order: 1.ResourceType 2.InstanceSelection 3.Action 4.ActionGroup
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
	if !auth.EnableAuthorize() {
		return nil
	}

	// unregister order: 1.Action 2.InstanceSelection 3.ResourceType 4.ActionGroup
	if err := v.unregisterModelActions(ctx, objects); err != nil {
		return err
	}

	if err := v.unregisterModelInstanceSelections(ctx, objects); err != nil {
		return err
	}

	if err := v.unregisterModelResourceTypes(ctx, objects); err != nil {
		return err
	}

	if err := v.updateModelActionGroups(ctx, header); err != nil {
		return err
	}

	return nil
}

// registerModelResourceTypes register resource types for models
func (v *viewer) registerModelResourceTypes(ctx context.Context, objects []metadata.Object) error {
	rid := util.ExtractRequestIDFromContext(ctx)
	resourceTypes := genDynamicResourceTypes(objects)
	if err := v.iam.Client.RegisterResourcesTypes(ctx, resourceTypes); err != nil {
		blog.ErrorJSON("register resourceTypes failed, error: %s, objects: %s, resourceTypes: %s， rid:%s",
			err.Error(), objects, resourceTypes, rid)
		return err
	}

	return nil
}

// unregisterModelResourceTypes unregister resourceTypes for models
func (v *viewer) unregisterModelResourceTypes(ctx context.Context, objects []metadata.Object) error {
	rid := util.ExtractRequestIDFromContext(ctx)
	typeIDs := []TypeID{}
	resourceTypes := genDynamicResourceTypes(objects)
	for _, resourceType := range resourceTypes {
		typeIDs = append(typeIDs, resourceType.ID)
	}
	if err := v.iam.Client.DeleteResourcesTypes(ctx, typeIDs); err != nil {
		blog.ErrorJSON("unregister resourceTypes failed, error: %s, objects: %s, resourceTypes: %s, rid:%s",
			err.Error(), objects, resourceTypes, rid)
		return err
	}

	return nil
}

// registerModelInstanceSelections register instanceSelections for models
func (v *viewer) registerModelInstanceSelections(ctx context.Context, objects []metadata.Object) error {
	rid := util.ExtractRequestIDFromContext(ctx)
	instanceSelections := genDynamicInstanceSelections(objects)
	if err := v.iam.Client.RegisterInstanceSelections(ctx, instanceSelections); err != nil {
		blog.ErrorJSON("register instanceSelections failed, error: %s, objects: %s, instanceSelections: %s, rid:%s",
			err.Error(), objects, instanceSelections, rid)
		return err
	}

	return nil
}

// unregisterModelInstanceSelections unregister instanceSelections for models
func (v *viewer) unregisterModelInstanceSelections(ctx context.Context, objects []metadata.Object) error {
	rid := util.ExtractRequestIDFromContext(ctx)
	instanceSelectionIDs := []InstanceSelectionID{}
	instanceSelections := genDynamicInstanceSelections(objects)
	for _, instanceSelection := range instanceSelections {
		instanceSelectionIDs = append(instanceSelectionIDs, instanceSelection.ID)
	}
	if err := v.iam.Client.DeleteInstanceSelections(ctx, instanceSelectionIDs); err != nil {
		blog.ErrorJSON("unregister instanceSelections failed, error: %s, objects: %s, instanceSelections: %s, rid:%s",
			err.Error(), objects, instanceSelections, rid)
		return err
	}

	return nil
}

// registerModelActions register actions for models
func (v *viewer) registerModelActions(ctx context.Context, objects []metadata.Object) error {
	rid := util.ExtractRequestIDFromContext(ctx)
	actions := genDynamicActions(objects)
	if err := v.iam.Client.RegisterActions(ctx, actions); err != nil {
		blog.ErrorJSON("register actions failed, error: %s, objects: %s, actions: %s, rid:%s",
			err.Error(), objects, actions, rid)
		return err
	}

	return nil
}

// unregisterModelActions unregister actions for models
func (v *viewer) unregisterModelActions(ctx context.Context, objects []metadata.Object) error {
	rid := util.ExtractRequestIDFromContext(ctx)
	actionIDs := []ActionID{}
	for _, obj := range objects {
		actionIDs = append(actionIDs, genDynamicActionIDs(obj)...)
	}

	// before deleting action, the dependent action polices must be deleted
	for _, actionID := range actionIDs {
		if err := v.iam.Client.DeleteActionPolicies(ctx, actionID); err != nil {
			blog.Errorf("delete action %s policies failed, err: %s, rid: %s", actionID, err, rid)
			return err
		}
	}

	if err := v.iam.Client.DeleteActions(ctx, actionIDs); err != nil {
		blog.ErrorJSON("unregister actions failed, error: %s, objects: %s, actionIDs: %s, rid:%s",
			err.Error(), objects, actionIDs, rid)
		return err
	}

	return nil
}

// updateModelActionGroups update actionGroups for models
// for now, the update api can only support full update, not incremental update
func (v *viewer) updateModelActionGroups(ctx context.Context, header http.Header) error {
	rid := util.ExtractRequestIDFromContext(ctx)
	objects, err := v.getCustomObjects(ctx, header)
	if err != nil {
		blog.Errorf("get custom objects failed, err: %s, rid: %s", err.Error(), rid)
		return err
	}
	actionGroups := GenerateActionGroups(objects)

	if err := v.iam.Client.UpdateActionGroups(ctx, actionGroups); err != nil {
		blog.ErrorJSON("update actionGroups failed, error: %s, actionGroups: %s, rid:%s",
			err.Error(), actionGroups, rid)
		return err
	}

	return nil
}

// getCustomObjects get objects which are custom
func (v *viewer) getCustomObjects(ctx context.Context, header http.Header) ([]metadata.Object, error) {
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
		return nil, fmt.Errorf("no custom objects were found")
	}

	objects := make([]metadata.Object, 0)
	for _, item := range resp.Data.Info {
		objects = append(objects, item.Spec)
	}

	return objects, nil
}
