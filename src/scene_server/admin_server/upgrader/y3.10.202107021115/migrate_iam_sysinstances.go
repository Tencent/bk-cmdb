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

package y3_10_202107021115

import (
	"context"
	"errors"
	"net/http"
	"strconv"

	iamtype "configcenter/src/ac/iam"
	"configcenter/src/common"
	"configcenter/src/common/auth"
	"configcenter/src/common/blog"
	"configcenter/src/common/mapstr"
	"configcenter/src/common/metadata"
	"configcenter/src/common/resource/esb"
	commonutil "configcenter/src/common/util"
	"configcenter/src/scene_server/admin_server/upgrader"
	"configcenter/src/storage/dal"
	"configcenter/src/storage/driver/mongodb/instancemapping"
)

// migrateIAMSysInstances migrate iam system instances
func migrateIAMSysInstances(ctx context.Context, db dal.RDB, iam *iamtype.IAM, conf *upgrader.Config) error {
	if !auth.EnableAuthorize() {
		return nil
	}

	// get all custom objects(without mainline objects)
	objects := []metadata.Object{}
	condition := map[string]interface{}{
		common.BKIsPre: false,
		common.BKClassificationIDField: map[string]interface{}{
			common.BKDBNE: "bk_biz_topo",
		},
	}
	err := db.Table(common.BKTableNameObjDes).Find(condition).All(ctx, &objects)
	if err != nil {
		blog.Errorf("get all custom objects failed, err: %v", err)
		return err
	}

	param := &iamtype.DeleteCMDBResourceParam{
		ActionIDs: []iamtype.ActionID{
			"create_sys_instance",
			"edit_sys_instance",
			"delete_sys_instance",
			"create_event_subscription",
			"edit_event_subscription",
			"delete_event_subscription",
			"find_event_subscription",
			"watch_set_template_event",
		},
		InstanceSelectionIDs: []iamtype.InstanceSelectionID{"sys_instance", "sys_event_pushing"},
		TypeIDs:              []iamtype.TypeID{"sys_instance", "sys_event_pushing"},
	}

	// add new system instances
	if err := iam.SyncIAMSysInstances(ctx, objects); err != nil {
		blog.Errorf("sync iam system instances failed, err: %v", err)
		return err
	}

	// migrate instance auth policies
	instanceActions := []iamtype.ActionID{"create_sys_instance", "edit_sys_instance", "delete_sys_instance"}
	for _, action := range instanceActions {
		if err := migrateModelInstancePermission(ctx, action, db, iam); err != nil {
			blog.Errorf("[upgrade y3.10.202106301057] migrate model instance authorization failed, error: %v", err)
			return err
		}
	}

	// delete the old cmdb resource
	return iam.DeleteCMDBResource(ctx, param, objects)
}

func migrateModelInstancePermission(ctx context.Context, action iamtype.ActionID, db dal.DB, iam *iamtype.IAM) error {
	var timestamp, count, pageSize int64 = 0, -1, 500
	var objectIDs []int64
	hasQueriedObjectIDs := false

	for page := int64(1); page < count/pageSize || count == -1; page++ {
		listPoliciesParam := &iamtype.ListPoliciesParams{
			ActionID:  action,
			Page:      page,
			PageSize:  pageSize,
			Timestamp: timestamp,
		}

		listPoliciesResp, err := iam.Client.ListPolicies(ctx, listPoliciesParam)
		if err != nil {
			blog.Errorf("list %s policies failed, page: %d, timestamp: %d, error: %v", action, page, timestamp, err)
			return err
		}

		if listPoliciesResp.Metadata.System != iamtype.SystemIDCMDB {
			blog.Errorf("list %s policies, but system id %s does not match", action, listPoliciesResp.Metadata.System)
			return errors.New("system id does not match")
		}

		count = listPoliciesResp.Count
		timestamp = listPoliciesResp.Metadata.Timestamp

		for _, policyRes := range listPoliciesResp.Results {
			parsedPolicy, err := parseInstancePolicy(policyRes.Expression)
			if err != nil {
				blog.ErrorJSON("parse %s policies %s failed, error: %s", action, policyRes.Expression, err)
				return err
			}

			// if user has permission to any instances, then migrate it to all current objects' instance action
			if parsedPolicy.isAny {
				if !hasQueriedObjectIDs {
					// inner objects and mainline objects do not authorize by common instance permission
					associations := make([]metadata.Association, 0)
					filter := mapstr.MapStr{
						common.AssociationKindIDField: common.AssociationKindMainline,
					}

					if err := db.Table(common.BKTableNameObjAsst).Find(filter).Fields(common.BKObjIDField).
						All(ctx, &associations); err != nil {
						blog.Errorf("get mainline associations failed, err: %v", err)
						return err
					}

					excludeObjIDs := []string{
						common.BKInnerObjIDApp, common.BKInnerObjIDSet, common.BKInnerObjIDModule,
						common.BKInnerObjIDHost, common.BKInnerObjIDProc, common.BKInnerObjIDPlat,
					}
					for _, association := range associations {
						if !metadata.IsCommon(association.ObjectID) {
							excludeObjIDs = append(excludeObjIDs, association.ObjectID)
						}
					}

					objFilter := map[string]interface{}{
						common.BKObjIDField: map[string]interface{}{
							common.BKDBNIN: excludeObjIDs,
						},
					}

					rawObjectIDs, err := db.Table(common.BKTableNameObjDes).Distinct(ctx, common.BKFieldID, objFilter)
					if err != nil {
						blog.Errorf("get all object ids failed, error: %v", err)
						return err
					}

					objectIDs, err = commonutil.SliceInterfaceToInt64(rawObjectIDs)
					if err != nil {
						blog.Errorf("parse all object ids failed, error: %v", err)
						return err
					}
					hasQueriedObjectIDs = true
				}

				for _, id := range objectIDs {
					err := batchGrantInstanceAuth(ctx, iam, policyRes.Subject, policyRes.ExpiredAt, action, nil, id)
					if err != nil {
						blog.ErrorJSON("batch grant instance auth failed, err: %s, policy: %s, id: %s", err, policyRes, id)
						return err
					}
				}
				continue
			}

			for _, id := range parsedPolicy.objectIDs {
				err := batchGrantInstanceAuth(ctx, iam, policyRes.Subject, policyRes.ExpiredAt, action, nil, id)
				if err != nil {
					blog.ErrorJSON("batch grant instance auth failed, err: %s, policy: %s, id: %s", err, policyRes, id)
					return err
				}
			}

			for objID, instanceIDs := range parsedPolicy.objInstIDMap {
				// get objectID by instance IDs
				instIDObjMappings, err := instancemapping.GetInstanceObjectMapping(instanceIDs)
				if err != nil {
					blog.Errorf("get instance object mapping from instance ids(%+v) failed, err: %v", instanceIDs, err)
					return err
				}

				objIDs := make([]string, 0)
				objInstMap := make(map[string][]int64)
				for _, objectInfo := range instIDObjMappings {
					objIDs = append(objIDs, objectInfo.ObjectID)
					objInstMap[objectInfo.ObjectID] = append(objInstMap[objectInfo.ObjectID], objectInfo.ID)
				}

				// get object IDs by objID since iam uses object ID for authorization
				objFilter := map[string]interface{}{
					common.BKObjIDField: map[string]interface{}{common.BKDBIN: objIDs},
				}
				objects := make([]metadata.Object, 0)
				if err := db.Table(common.BKTableNameObjDes).Find(objFilter).Fields(common.BKObjIDField,
					common.BKFieldID).All(ctx, &objects); err != nil {
					blog.Errorf("get objects failed, error: %v, objIDs: %+v", err, objIDs)
					return err
				}

				var objInstIDs []int64
				var tableName string
				for _, object := range objects {
					// only use the instances that belongs to the object
					if object.ID == objID {
						objInstIDs = objInstMap[object.ObjectID]
						tableName = common.GetObjectInstTableName(object.ObjectID, object.OwnerID)
						break
					}
				}

				if len(objInstIDs) == 0 {
					continue
				}

				instanceFilter := map[string]interface{}{
					common.BKInstIDField: map[string]interface{}{common.BKDBIN: objInstIDs},
				}

				// get instance names by ids
				instances := make([]iamtype.SimplifiedInstance, 0)
				if err := db.Table(tableName).Find(instanceFilter).Fields(common.BKInstIDField,
					common.BKInstNameField).All(ctx, &instances); err != nil {
					blog.Errorf("get instances failed, error: %v, instIDs: %+v", err, instanceIDs)
					return err
				}

				instanceMap := make(map[int64]string)
				for _, instance := range instances {
					instanceMap[instance.InstanceID] = instance.InstanceName
				}

				instIDs := make([]int64, 0)
				for index, instID := range objInstIDs {
					instIDs = append(instIDs, instID)

					// iam only allows granting 20 permissions at a time
					if len(instIDs) < 20 && index != len(instIDObjMappings)-1 {
						continue
					}

					iamInsts := make([]metadata.IamInstance, 0)
					for _, instanceID := range instIDs {
						iamInsts = append(iamInsts, metadata.IamInstance{
							ID:   strconv.FormatInt(instanceID, 10),
							Name: instanceMap[instanceID],
						})
					}

					err = batchGrantInstanceAuth(ctx, iam, policyRes.Subject, policyRes.ExpiredAt, action, iamInsts, objID)
					if err != nil {
						blog.ErrorJSON("batch grant instance auth failed, err: %s, policy: %s, instIDs: %s, objID: %s",
							err, policyRes, instIDs, objID)
						return err
					}
					instIDs = make([]int64, 0)
				}
			}
		}
	}
	return nil
}

func batchGrantInstanceAuth(ctx context.Context, iam *iamtype.IAM, subject iamtype.PolicySubject, expiredAt int64,
	actionID iamtype.ActionID, instances []metadata.IamInstance, objectID int64) error {

	header := http.Header{}
	header.Add(common.BKHTTPHeaderUser, common.CCSystemOperatorUserName)
	req := &metadata.IamBatchOperateInstanceAuthReq{
		Asynchronous: false,
		Operate:      metadata.IamGrantOperation,
		System:       iamtype.SystemIDCMDB,
		Actions:      []metadata.ActionWithID{{ID: convertOldInstanceAction(actionID, objectID)}},
		Subject: metadata.IamSubject{
			Type: subject.Type,
			Id:   subject.ID,
		},
		ExpiredAt: expiredAt,
	}

	// create model instance is related to no resources
	if actionID == "create_sys_instance" {
		req.Resources = make([]metadata.IamInstances, 0)
		_, err := esb.EsbClient().IamSrv().BatchOperateInstanceAuth(ctx, header, req)
		if err != nil {
			blog.ErrorJSON("batch register instance auth failed, err: %s, input: %s", err, req)
			return err
		}
		return nil
	}

	req.Resources = []metadata.IamInstances{{
		System:    iamtype.SystemIDCMDB,
		Type:      string(iamtype.GenIAMDynamicResTypeID(objectID)),
		Instances: make([]metadata.IamInstance, 0),
	}}

	// if specified no instances, then grant permissions to all instances in the object
	if len(instances) > 0 {
		req.Resources[0].Instances = instances
	}

	_, err := esb.EsbClient().IamSrv().BatchOperateInstanceAuth(ctx, header, req)
	if err != nil {
		blog.ErrorJSON("batch register instance auth failed, err: %s, input: %s", err, req)
		return err
	}
	return nil
}

// convertOldInstanceAction convert old form of iam instance action to the new form
func convertOldInstanceAction(actionID iamtype.ActionID, id int64) string {
	switch actionID {
	case "create_sys_instance":
		return string(iamtype.GenDynamicActionID(iamtype.Create, id))
	case "edit_sys_instance":
		return string(iamtype.GenDynamicActionID(iamtype.Edit, id))
	case "delete_sys_instance":
		return string(iamtype.GenDynamicActionID(iamtype.Delete, id))
	}
	return ""
}
