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

package authcenter

import (
	"fmt"

	"configcenter/src/auth/meta"
)

func GenerateResourceID(attribute *meta.ResourceAttribute) (string, error) {
	switch attribute.Basic.Type {
	case meta.Business:
		return businessResourceID(attribute)
	case meta.Model:
		return modelResourceID(attribute)
	case meta.ModelModule:
		return modelModuleResourceID(attribute)
	case meta.ModelSet:
		return modelSetResourceID(attribute)
	case meta.MainlineModel:
		return mainlineModelResourceID(attribute)
	case meta.MainlineModelTopology:
		return mainlineModelTopologyResourceID(attribute)
	case meta.MainlineInstanceTopology:
		return mainlineInstanceTopologyResourceID(attribute)
	case meta.AssociationType:
		return associationTypeResourceID(attribute)
	case meta.ModelAssociation:
		return modelAssociationResourceID(attribute)
	case meta.ModelInstanceAssociation:
		return modelInstanceAssociationResourceID(attribute)
	case meta.ModelInstance:
		return modelInstanceResourceID(attribute)
	case meta.ModelInstanceTopology:
		return modelInstanceTopologyResourceID(attribute)
	case meta.ModelTopology:
		return modelTopologyResourceID(attribute)
	case meta.ModelClassification:
		return modelClassificationResourceID(attribute)
	case meta.ModelAttributeGroup:
		return modelAttributeGroupResourceID(attribute)
	case meta.ModelAttribute:
		return modelAttributeResourceID(attribute)
	case meta.ModelUnique:
		return modelUniqueResourceID(attribute)
	case meta.HostUserCustom:
		return hostUserCustomResourceID(attribute)
	case meta.HostFavorite:
		return hostFavoriteResourceID(attribute)
	case meta.Process:
		return processResourceID(attribute)
	case meta.NetDataCollector:
		return netDataCollectorResourceID(attribute)
	default:
		return "", fmt.Errorf("unsupported resource type: %s", attribute.Type)
	}
}

// generate business related resource id.
func businessResourceID(attribute *meta.ResourceAttribute) (string, error) {

	return "", nil
}

// generate model's resource id, works for app model and model management
// resource type in auth center.
func modelResourceID(attribute *meta.ResourceAttribute) (string, error) {

	return "", nil
}

// generate module resource id.
func modelModuleResourceID(attribute *meta.ResourceAttribute) (string, error) {

	return "", nil
}

func modelSetResourceID(attribute *meta.ResourceAttribute) (string, error) {

	return "", nil
}

func mainlineModelResourceID(attribute *meta.ResourceAttribute) (string, error) {

	return "", nil
}

func mainlineModelTopologyResourceID(attribute *meta.ResourceAttribute) (string, error) {

	return "", nil
}

func mainlineInstanceTopologyResourceID(attribute *meta.ResourceAttribute) (string, error) {

	return "", nil
}

func modelAssociationResourceID(attribute *meta.ResourceAttribute) (string, error) {

	return "", nil
}

func associationTypeResourceID(attribute *meta.ResourceAttribute) (string, error) {

	return "", nil
}

func modelInstanceAssociationResourceID(attribute *meta.ResourceAttribute) (string, error) {

	return "", nil
}

func modelInstanceResourceID(attribute *meta.ResourceAttribute) (string, error) {

	return "", nil
}

func modelInstanceTopologyResourceID(attribute *meta.ResourceAttribute) (string, error) {

	return "", nil
}

func modelTopologyResourceID(attribute *meta.ResourceAttribute) (string, error) {

	return "", nil
}

func modelClassificationResourceID(attribute *meta.ResourceAttribute) (string, error) {

	return "", nil
}

func modelAttributeGroupResourceID(attribute *meta.ResourceAttribute) (string, error) {

	return "", nil
}

func modelAttributeResourceID(attribute *meta.ResourceAttribute) (string, error) {

	return "", nil
}

func modelUniqueResourceID(attribute *meta.ResourceAttribute) (string, error) {

	return "", nil
}

func hostUserCustomResourceID(attribute *meta.ResourceAttribute) (string, error) {

	return "", nil
}

func hostFavoriteResourceID(attribute *meta.ResourceAttribute) (string, error) {

	return "", nil
}

func processResourceID(attribute *meta.ResourceAttribute) (string, error) {

	return "", nil
}

func netDataCollectorResourceID(attribute *meta.ResourceAttribute) (string, error) {

	return "", nil
}
