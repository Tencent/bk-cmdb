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

package service

import (
	"fmt"

	"configcenter/src/ac/iam"
	"configcenter/src/common"
	"configcenter/src/common/blog"
	"configcenter/src/common/http/rest"
	"configcenter/src/common/metadata"
	"configcenter/src/scene_server/auth_server/types"
)

// genResourcePullMethod generate iam callback methods for input resource type, method not set means not related to this kind of instances
func (s *AuthService) genResourcePullMethod(kit *rest.Kit, resourceType iam.TypeID) (types.ResourcePullMethod, error) {
	switch resourceType {
	case iam.Host:
		return types.ResourcePullMethod{
			ListAttr:             s.lgc.ListAttr,
			ListAttrValue:        s.lgc.ListAttrValue,
			ListInstance:         s.lgc.ListHostInstance,
			FetchInstanceInfo:    s.lgc.FetchHostInfo,
			ListInstanceByPolicy: s.lgc.ListHostByPolicy,
		}, nil

	case iam.Business, iam.BusinessForHostTrans:

		// business instances should not include resource pool business
		extraCond := map[string]interface{}{
			common.BKDefaultField: map[string]interface{}{
				common.BKDBNE: common.DefaultAppFlag,
			},
		}

		return types.ResourcePullMethod{
			ListAttr:      s.lgc.ListAttr,
			ListAttrValue: s.lgc.ListAttrValue,
			ListInstance: func(kit *rest.Kit, resourceType iam.TypeID, filter *types.ListInstanceFilter, page types.Page) (*types.ListInstanceResult, error) {
				return s.lgc.ListSystemInstance(kit, resourceType, filter, page, extraCond)
			},
			FetchInstanceInfo: func(kit *rest.Kit, resourceType iam.TypeID, filter *types.FetchInstanceInfoFilter) ([]map[string]interface{}, error) {
				return s.lgc.FetchInstanceInfo(kit, resourceType, filter, extraCond)
			},
			ListInstanceByPolicy: func(kit *rest.Kit, resourceType iam.TypeID, filter *types.ListInstanceByPolicyFilter, page types.Page) (result *types.ListInstanceResult, e error) {
				return s.lgc.ListInstanceByPolicy(kit, resourceType, filter, page, extraCond)
			},
		}, nil

	case iam.SysCloudArea:

		// cloud area instances should not include default cloud area, since it can't be operated
		extraCond := map[string]interface{}{
			common.BKCloudIDField: map[string]interface{}{
				common.BKDBNE: common.BKDefaultDirSubArea,
			},
		}

		return types.ResourcePullMethod{
			ListAttr:      s.lgc.ListAttr,
			ListAttrValue: s.lgc.ListAttrValue,
			ListInstance: func(kit *rest.Kit, resourceType iam.TypeID, filter *types.ListInstanceFilter, page types.Page) (*types.ListInstanceResult, error) {
				return s.lgc.ListSystemInstance(kit, resourceType, filter, page, extraCond)
			},
			FetchInstanceInfo: func(kit *rest.Kit, resourceType iam.TypeID, filter *types.FetchInstanceInfoFilter) ([]map[string]interface{}, error) {
				return s.lgc.FetchInstanceInfo(kit, resourceType, filter, extraCond)
			},
			ListInstanceByPolicy: func(kit *rest.Kit, resourceType iam.TypeID, filter *types.ListInstanceByPolicyFilter, page types.Page) (result *types.ListInstanceResult, e error) {
				return s.lgc.ListInstanceByPolicy(kit, resourceType, filter, page, extraCond)
			},
		}, nil

	case iam.SysInstance:
		return types.ResourcePullMethod{
			ListAttr:          s.lgc.ListAttr,
			ListAttrValue:     s.lgc.ListAttrValue,
			ListInstance:      s.lgc.ListModelInstance,
			FetchInstanceInfo: s.lgc.FetchObjInstInfo,
			ListInstanceByPolicy: func(kit *rest.Kit, resourceType iam.TypeID, filter *types.ListInstanceByPolicyFilter, page types.Page) (result *types.ListInstanceResult, e error) {
				return s.lgc.ListInstanceByPolicy(kit, resourceType, filter, page, nil)
			},
		}, nil

	case iam.BizCustomQuery, iam.BizProcessServiceTemplate, iam.BizSetTemplate:
		return types.ResourcePullMethod{
			ListInstance: s.lgc.ListBusinessInstance,
			FetchInstanceInfo: func(kit *rest.Kit, resourceType iam.TypeID, filter *types.FetchInstanceInfoFilter) ([]map[string]interface{}, error) {
				return s.lgc.FetchInstanceInfo(kit, resourceType, filter, nil)
			},
			ListInstanceByPolicy: func(kit *rest.Kit, resourceType iam.TypeID, filter *types.ListInstanceByPolicyFilter, page types.Page) (result *types.ListInstanceResult, e error) {
				return s.lgc.ListInstanceByPolicy(kit, resourceType, filter, page, nil)
			},
		}, nil

	case iam.SysModelGroup, iam.SysCloudAccount, iam.SysCloudResourceTask:
		return types.ResourcePullMethod{
			ListInstance: func(kit *rest.Kit, resourceType iam.TypeID, filter *types.ListInstanceFilter, page types.Page) (*types.ListInstanceResult, error) {
				return s.lgc.ListSystemInstance(kit, resourceType, filter, page, nil)
			},
			FetchInstanceInfo: func(kit *rest.Kit, resourceType iam.TypeID, filter *types.FetchInstanceInfoFilter) ([]map[string]interface{}, error) {
				return s.lgc.FetchInstanceInfo(kit, resourceType, filter, nil)
			},
			ListInstanceByPolicy: func(kit *rest.Kit, resourceType iam.TypeID, filter *types.ListInstanceByPolicyFilter, page types.Page) (result *types.ListInstanceResult, e error) {
				return s.lgc.ListInstanceByPolicy(kit, resourceType, filter, page, nil)
			},
		}, nil

	case iam.SysModel, iam.SysInstanceModel, iam.SysModelEvent, iam.MainlineModelEvent:

		// get mainline objects
		mainlineOpt := &metadata.QueryCondition{
			Condition: map[string]interface{}{common.AssociationKindIDField: common.AssociationKindMainline},
		}
		asstRes, err := s.lgc.CoreAPI.CoreService().Association().ReadModelAssociation(kit.Ctx, kit.Header, mainlineOpt)
		if err != nil {
			blog.Errorf("search mainline association failed, err: %v, rid: %s", err, kit.Rid)
			return types.ResourcePullMethod{}, err
		}

		if err := asstRes.CCError(); err != nil {
			blog.Errorf("search mainline association failed, err: %v, rid: %s", err, kit.Rid)
			return types.ResourcePullMethod{}, err
		}

		mainlineObjIDs := make([]string, 0)
		for _, asst := range asstRes.Data.Info {
			if metadata.IsCommon(asst.ObjectID) {
				mainlineObjIDs = append(mainlineObjIDs, asst.ObjectID)
			}
		}

		// process and cloud area are temporarily excluded TODO: remove this restriction when they are available for user
		// instance model is used as parent layer of instances, should exclude host model and mainline model as
		// they use separate operations
		excludedObjIDs := []string{common.BKInnerObjIDProc, common.BKInnerObjIDPlat}

		var extraCond map[string]interface{}
		switch resourceType {
		case iam.SysModelEvent, iam.SysInstanceModel:
			excludedObjIDs = append(excludedObjIDs, common.BKInnerObjIDHost, common.BKInnerObjIDApp,
				common.BKInnerObjIDSet, common.BKInnerObjIDModule)
			excludedObjIDs = append(excludedObjIDs, mainlineObjIDs...)
			extraCond = map[string]interface{}{
				common.BKObjIDField: map[string]interface{}{
					common.BKDBNIN: excludedObjIDs,
				},
			}
		case iam.MainlineModelEvent:
			extraCond = map[string]interface{}{
				common.BKObjIDField: map[string]interface{}{
					common.BKDBIN: mainlineObjIDs,
				},
			}
		default:
			extraCond = map[string]interface{}{
				common.BKObjIDField: map[string]interface{}{
					common.BKDBNIN: excludedObjIDs,
				},
			}
		}

		return types.ResourcePullMethod{
			ListInstance: func(kit *rest.Kit, resourceType iam.TypeID, filter *types.ListInstanceFilter,
				page types.Page) (*types.ListInstanceResult, error) {
				return s.lgc.ListSystemInstance(kit, resourceType, filter, page, extraCond)
			},
			FetchInstanceInfo: func(kit *rest.Kit, resourceType iam.TypeID, filter *types.FetchInstanceInfoFilter) (
				[]map[string]interface{}, error) {
				return s.lgc.FetchInstanceInfo(kit, resourceType, filter, extraCond)
			},
			ListInstanceByPolicy: func(kit *rest.Kit, resourceType iam.TypeID, filter *types.ListInstanceByPolicyFilter,
				page types.Page) (result *types.ListInstanceResult, e error) {
				return s.lgc.ListInstanceByPolicy(kit, resourceType, filter, page, extraCond)
			},
		}, nil

	case iam.SysAssociationType:

		// association types should not include preset ones, since they can't be operated
		extraCond := map[string]interface{}{
			common.BKIsPre: map[string]interface{}{
				common.BKDBNE: true,
			},
		}

		return types.ResourcePullMethod{
			ListInstance: func(kit *rest.Kit, resourceType iam.TypeID, filter *types.ListInstanceFilter, page types.Page) (*types.ListInstanceResult, error) {
				return s.lgc.ListSystemInstance(kit, resourceType, filter, page, extraCond)
			},
			FetchInstanceInfo: func(kit *rest.Kit, resourceType iam.TypeID, filter *types.FetchInstanceInfoFilter) ([]map[string]interface{}, error) {
				return s.lgc.FetchInstanceInfo(kit, resourceType, filter, extraCond)
			},
			ListInstanceByPolicy: func(kit *rest.Kit, resourceType iam.TypeID, filter *types.ListInstanceByPolicyFilter, page types.Page) (result *types.ListInstanceResult, e error) {
				return s.lgc.ListInstanceByPolicy(kit, resourceType, filter, page, extraCond)
			},
		}, nil

	case iam.SysResourcePoolDirectory, iam.SysHostRscPoolDirectory:
		resourcePoolBizID, err := s.lgc.GetResourcePoolBizID(kit)
		if err != nil {
			return types.ResourcePullMethod{}, err
		}

		// resource pool directory must be in the resource pool business
		extraCond := map[string]interface{}{
			common.BKAppIDField: resourcePoolBizID,
		}

		return types.ResourcePullMethod{
			ListInstance: func(kit *rest.Kit, resourceType iam.TypeID, filter *types.ListInstanceFilter, page types.Page) (*types.ListInstanceResult, error) {
				return s.lgc.ListSystemInstance(kit, resourceType, filter, page, extraCond)
			},
			FetchInstanceInfo: func(kit *rest.Kit, resourceType iam.TypeID, filter *types.FetchInstanceInfoFilter) ([]map[string]interface{}, error) {
				return s.lgc.FetchInstanceInfo(kit, resourceType, filter, extraCond)
			},
			ListInstanceByPolicy: func(kit *rest.Kit, resourceType iam.TypeID, filter *types.ListInstanceByPolicyFilter, page types.Page) (result *types.ListInstanceResult, e error) {
				return s.lgc.ListInstanceByPolicy(kit, resourceType, filter, page, extraCond)
			},
		}, nil

	case iam.SysOperationStatistic, iam.SysAuditLog, iam.BizCustomField, iam.BizHostApply,
		iam.BizTopology, iam.SysEventWatch, iam.BizProcessServiceCategory, iam.BizProcessServiceInstance:
		return types.ResourcePullMethod{}, nil

	default:
		return types.ResourcePullMethod{}, fmt.Errorf("gen method failed: unsupported resource type: %s", resourceType)
	}
}
