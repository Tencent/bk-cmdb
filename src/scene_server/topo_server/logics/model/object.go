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

package model

import (
	"fmt"

	"configcenter/src/ac/extensions"
	"configcenter/src/apimachinery"
	"configcenter/src/common"
	"configcenter/src/common/auditlog"
	"configcenter/src/common/blog"
	"configcenter/src/common/http/rest"
	"configcenter/src/common/language"
	"configcenter/src/common/mapstr"
	"configcenter/src/common/mapstruct"
	"configcenter/src/common/metadata"
	"configcenter/src/common/util"
)

type rowInfo struct {
	Row  int64  `json:"row"`
	Info string `json:"info"`
	// value can empty, eg:parse error
	PropertyID string `json:"bk_property_id"`
}

// ObjectOperationInterface object operation methods
type ObjectOperationInterface interface {
	CreateObject(kit *rest.Kit, isMainline bool, data mapstr.MapStr) (*metadata.Object, error)
	DeleteObject(kit *rest.Kit, id int64, needCheckInst bool) error
	FindObject(kit *rest.Kit, cond mapstr.MapStr) ([]metadata.Object, error)
	FindObjectTopo(kit *rest.Kit, cond mapstr.MapStr) ([]metadata.ObjectTopo, error)
	FindSingleObject(kit *rest.Kit, objectID string) (*metadata.Object, error)
	UpdateObject(kit *rest.Kit, data mapstr.MapStr, id int64) error
	IsValidObject(kit *rest.Kit, objID string) error
}

// NewObjectOperation create a new object operation instance
func NewObjectOperation(client apimachinery.ClientSetInterface,
	authManager *extensions.AuthManager) ObjectOperationInterface {
	return &object{
		clientSet:   client,
		authManager: authManager,
	}
}

type object struct {
	clientSet   apimachinery.ClientSetInterface
	authManager *extensions.AuthManager
	lang        language.DefaultCCLanguageIf
}

// IsValidObject check whether objID is a real model's bk_obj_id field in backend
func (o *object) IsValidObject(kit *rest.Kit, objID string) error {

	checkObjCond := mapstr.MapStr{
		common.BKObjIDField: objID,
	}

	objItems, err := o.FindObject(kit, checkObjCond)
	if err != nil {
		blog.Errorf("failed to check the object repeated, err: %s, rid: %s", err.Error(), kit.Rid)
		return kit.CCError.CCErrorf(common.CCErrCommParamsIsInvalid, err.Error())
	}

	if len(objItems) == 0 {
		return kit.CCError.CCErrorf(common.CCErrCommParamsIsInvalid, common.BKObjIDField)
	}

	return nil
}

func (o *object) FindSingleObject(kit *rest.Kit, objectID string) (*metadata.Object, error) {

	cond := mapstr.MapStr{
		common.BKObjIDField: objectID,
	}

	objs, err := o.FindObject(kit, cond)
	if err != nil {
		blog.Errorf("get model failed, failed to get model by supplier account(%s) objects(%s), err: %s, rid: %s",
			kit.SupplierAccount, objectID, err.Error(), kit.Rid)
		return nil, err
	}

	if len(objs) == 0 {
		blog.Errorf("get model failed, get model by supplier account(%s) objects(%s) not found, result: %+v, "+
			"rid: %s", kit.SupplierAccount, objectID, objs, kit.Rid)
		return nil, kit.CCError.New(common.CCErrTopoObjectSelectFailed,
			kit.CCError.Error(common.CCErrCommNotFound).Error())
	}

	if len(objs) > 1 {
		blog.Errorf("get model failed, get model by supplier account(%s) objects(%s) get multiple, result: %+v, "+
			"rid: %s", kit.SupplierAccount, objectID, objs, kit.Rid)
		return nil, kit.CCError.New(common.CCErrTopoObjectSelectFailed,
			kit.CCError.Error(common.CCErrCommGetMultipleObject).Error())
	}

	return &objs[0], nil
}

func (o *object) CreateObject(kit *rest.Kit, isMainline bool, data mapstr.MapStr) (*metadata.Object, error) {

	obj, err := o.IsValid(kit, false, data)
	if err != nil {
		blog.Errorf("valid data(%#v) failed, err: %s, rid: %s", data, err.Error(), kit.Rid)
		return nil, err
	}

	objCls, err := o.clientSet.CoreService().Model().ReadModelClassification(kit.Ctx, kit.Header,
		&metadata.QueryCondition{Condition: mapstr.MapStr{
			common.BKClassificationIDField: obj.ObjCls,
		},
		})
	if err != nil {
		blog.Errorf("get object classification by params failed, err: %s, rid: %s", err.Error(), kit.Rid)
		return nil, err
	}

	if len(objCls.Data.Info) == 0 {
		blog.Errorf("can't find classification by params, classification: %s is not exist, rid: %s",
			obj.ObjCls, kit.Rid)
		return nil, kit.CCError.CCErrorf(common.CCErrCommParamsIsInvalid, common.BKClassificationIDField)
	}

	if len(obj.ObjIcon) == 0 {
		return nil, kit.CCError.CCErrorf(common.CCErrCommParamsNeedSet, common.BKObjIconField)
	}

	obj.OwnerID = kit.SupplierAccount

	objRsp, err := o.clientSet.CoreService().Model().CreateModel(kit.Ctx, kit.Header, &metadata.CreateModel{Spec: *obj})
	if err != nil {
		blog.Errorf("failed to request the object controller, error info is %s, rid: %s", err.Error(), kit.Rid)
		return nil, kit.CCError.CCError(common.CCErrCommHTTPDoRequestFailed)
	}

	if !objRsp.Result {
		blog.Errorf("failed to search the object(%s), error info is %s, rid: %s",
			obj.ObjectID, objRsp.ErrMsg, kit.Rid)
		return nil, kit.CCError.New(objRsp.Code, objRsp.ErrMsg)
	}

	obj.ID = int64(objRsp.Data.Created.ID)

	// create the default group
	groupData := metadata.Group{
		IsDefault:  true,
		GroupIndex: -1,
		GroupName:  "Default",
		GroupID:    "default",
		ObjectID:   obj.ObjectID,
		OwnerID:    obj.OwnerID,
	}

	rspGrp, err := o.clientSet.CoreService().Model().CreateAttributeGroup(kit.Ctx, kit.Header,
		obj.ObjectID, metadata.CreateModelAttributeGroup{Data: groupData})
	if err != nil {
		blog.Errorf("create attribute group failed, err: %s, rid: %s", err.Error(), kit.Rid)
		return nil, kit.CCError.Error(common.CCErrCommHTTPDoRequestFailed)
	}

	if !rspGrp.Result {
		blog.Errorf("create attribute group[%s] failed, err: is %s, rid: %s",
			groupData.GroupID, rspGrp.ErrMsg, kit.Rid)
		return nil, kit.CCError.Error(common.CCErrTopoObjectGroupCreateFailed)
	}

	keys := make([]metadata.UniqueKey, 0)
	// create the default inst name
	attr := metadata.Attribute{
		ObjectID:          obj.ObjectID,
		IsOnly:            true,
		IsPre:             true,
		Creator:           "user",
		IsEditable:        true,
		PropertyIndex:     -1,
		PropertyGroup:     groupData.GroupID,
		PropertyGroupName: groupData.GroupName,
		IsRequired:        true,
		PropertyType:      common.FieldTypeSingleChar,
		PropertyID:        obj.GetInstNameFieldName(),
		PropertyName:      obj.GetDefaultInstPropertyName(),
		OwnerID:           kit.SupplierAccount,
	}

	// create a new record
	rspAttr, err := o.clientSet.CoreService().Model().CreateModelAttrs(kit.Ctx, kit.Header,
		attr.ObjectID, &metadata.CreateModelAttributes{Attributes: []metadata.Attribute{attr}})
	if err != nil {
		blog.Errorf("failed to request coreService to create model attrs, the err: %s, ObjectID: %s, input: %s, "+
			"rid: %s", err.Error(), attr.ObjectID, attr, kit.Rid)
		return nil, kit.CCError.CCError(common.CCErrCommHTTPDoRequestFailed)
	}

	if !rspAttr.Result {
		blog.Errorf("create model attrs failed, ObjectID: %s, input: %s, rid: %s", attr.ObjectID, attr, kit.Rid)
		return nil, rspAttr.CCError()
	}

	for _, exception := range rspAttr.Data.Exceptions {
		return nil, kit.CCError.New(int(exception.Code), exception.Message)
	}

	if len(rspAttr.Data.Repeated) > 0 {
		blog.Errorf("create model attrs failed, the attr is duplicated, ObjectID: %s, input: %s, rid: %s",
			attr.ObjectID, attr, kit.Rid)
		return nil, kit.CCError.CCError(common.CCErrorAttributeNameDuplicated)
	}

	if len(rspAttr.Data.Created) != 1 {
		blog.Errorf("create model attrs created amount error, ObjectID: %s, input: %s, rid: %s",
			attr.ObjectID, attr, kit.Rid)
		return nil, kit.CCError.CCError(common.CCErrTopoObjectAttributeCreateFailed)
	}

	attr.ID = int64(rspAttr.Data.Created[0].ID)

	keys = append(keys, metadata.UniqueKey{Kind: metadata.UniqueKeyKindProperty, ID: uint64(attr.ID)})

	if isMainline {
		pAttr := metadata.Attribute{
			ObjectID:          obj.ObjectID,
			IsOnly:            true,
			IsPre:             true,
			Creator:           "system",
			IsEditable:        true,
			PropertyIndex:     -1,
			PropertyGroup:     groupData.GroupID,
			PropertyGroupName: groupData.GroupName,
			IsRequired:        true,
			PropertyType:      common.FieldTypeInt,
			PropertyID:        common.BKInstParentStr,
			PropertyName:      common.BKInstParentStr,
			IsSystem:          true,
			OwnerID:           kit.SupplierAccount,
		}

		rsppAttr, err := o.clientSet.CoreService().Model().CreateModelAttrs(kit.Ctx, kit.Header,
			pAttr.ObjectID, &metadata.CreateModelAttributes{Attributes: []metadata.Attribute{pAttr}})
		if err != nil {
			blog.Errorf("failed to request coreService to create model attrs, the err: %s, "+
				"ObjectID: %s, input: %s, rid: %s", err.Error(), pAttr.ObjectID, pAttr, kit.Rid)
			return nil, kit.CCError.CCError(common.CCErrCommHTTPDoRequestFailed)
		}

		if !rsppAttr.Result {
			blog.Errorf("create model attrs failed, ObjectID: %s, input: %s, rid: %s", pAttr.ObjectID, pAttr, kit.Rid)
			return nil, rsppAttr.CCError()
		}

		for _, exception := range rsppAttr.Data.Exceptions {
			return nil, kit.CCError.New(int(exception.Code), exception.Message)
		}

		if len(rsppAttr.Data.Repeated) > 0 {
			blog.Errorf("create model attrs failed, the attr is duplicated, ObjectID: %s, input: %s, rid: %s",
				pAttr.ObjectID, pAttr, kit.Rid)
			return nil, kit.CCError.CCError(common.CCErrorAttributeNameDuplicated)
		}

		if len(rsppAttr.Data.Created) != 1 {
			blog.Errorf("create model attrs created amount error, ObjectID: %s, input: %s, rid: %s",
				pAttr.ObjectID, pAttr, kit.Rid)
			return nil, kit.CCError.CCError(common.CCErrTopoObjectAttributeCreateFailed)
		}
		pAttr.ID = int64(rsppAttr.Data.Created[0].ID)

		keys = append(keys, metadata.UniqueKey{Kind: metadata.UniqueKeyKindProperty, ID: uint64(pAttr.ID)})
	}

	uni := metadata.ObjectUnique{
		ObjID:   obj.ObjectID,
		OwnerID: kit.SupplierAccount,
		Keys:    keys,
		Ispre:   false,
	}
	// NOTICE: 2021年03月29日  唯一索引与index.MainLineInstanceUniqueIndex,index.InstanceUniqueIndex定义强依赖
	// 原因：建立模型之前要将表和表中的索引提前建立，mongodb 4.2.6(4.4之前)事务中不能建表，事务操作表中数据操作和建表，建立索引为互斥操作。
	resp, err := o.clientSet.CoreService().Model().CreateModelAttrUnique(kit.Ctx, kit.Header,
		uni.ObjID, metadata.CreateModelAttrUnique{Data: uni})
	if err != nil {
		blog.Errorf("create unique for %s failed, err: %s, rid: %s", uni.ObjID, err.Error(), kit.Rid)
		return nil, kit.CCError.Error(common.CCErrTopoObjectUniqueCreateFailed)
	}
	if !resp.Result {
		return nil, kit.CCError.New(resp.Code, resp.ErrMsg)
	}

	// generate audit log of object attribute group.
	audit := auditlog.NewObjectAuditLog(o.clientSet.CoreService())
	generateAuditParameter := auditlog.NewGenerateAuditCommonParameter(kit, metadata.AuditCreate)
	auditLog, err := audit.GenerateAuditLog(generateAuditParameter, obj.ID, nil)
	if err != nil {
		blog.Errorf("create object %s success, but generate audit log failed, err: %v, rid: %s",
			obj.ObjectName, err, kit.Rid)
		return nil, err
	}

	// save audit log.
	if err := audit.SaveAuditLog(kit, *auditLog); err != nil {
		blog.Errorf("create object %s success, but save audit log failed, err: %v, rid: %s",
			obj.ObjectName, err, kit.Rid)
		return nil, err
	}

	return obj, nil
}

// DeleteObject delete model by id
func (o *object) DeleteObject(kit *rest.Kit, id int64, needCheckInst bool) error {
	if id <= 0 {
		return kit.CCError.CCErrorf(common.CCErrCommParamsInvalid, common.BKFieldID)
	}

	// get model by id
	cond := mapstr.MapStr{
		metadata.ModelFieldID: id,
	}

	objs, err := o.FindObject(kit, cond)
	if err != nil {
		blog.Errorf("failed to find objects, the condition is (%v) err: %s, rid: %s", cond, err.Error(), kit.Rid)
		return err
	}
	// shouldn't return 404 error here, legacy implements just ignore not found error
	if len(objs) == 0 {
		blog.V(3).Infof("object not found, condition: %v, rid: %s", cond, kit.Rid)
		return nil
	}
	if len(objs) > 1 {
		return kit.CCError.CCError(common.CCErrCommGetMultipleObject)
	}
	obj := objs[0]

	// check whether it can be deleted
	if needCheckInst {
		if err = o.CanDelete(kit, obj); err != nil {
			return err
		}
	}

	// generate audit log of object.
	audit := auditlog.NewObjectAuditLog(o.clientSet.CoreService())
	generateAuditParameter := auditlog.NewGenerateAuditCommonParameter(kit, metadata.AuditDelete)
	auditLog, err := audit.GenerateAuditLog(generateAuditParameter, obj.ID, nil)
	if err != nil {
		blog.Errorf("generate audit log failed before delete object, objName: %s, err: %v, rid: %s",
			obj.ObjectName, err, kit.Rid)
		return err
	}

	// DeleteModelCascade 将会删除模型/模型属性/属性分组/唯一校验
	rsp, err := o.clientSet.CoreService().Model().DeleteModelCascade(kit.Ctx, kit.Header, id)
	if err != nil {
		blog.Errorf("failed to request the object controller, err: %s, rid: %s", err.Error(), kit.Rid)
		return kit.CCError.Error(common.CCErrCommHTTPDoRequestFailed)
	}
	if !rsp.Result {
		blog.Errorf("failed to delete the object by the id(%d), rid: %s", id, kit.Rid)
		return kit.CCError.New(rsp.Code, rsp.ErrMsg)
	}

	// save audit log.
	if err = audit.SaveAuditLog(kit, *auditLog); err != nil {
		blog.Errorf("delete object %s success, save audit log failed, err: %v, rid: %s", obj.ObjectName, err, kit.Rid)
		return err
	}

	return nil
}

// CanDelete return nil only when:
// 1. not inner model
// 2. model has no instances
// 3. model has no association with other model
func (o *object) CanDelete(kit *rest.Kit, targetObj metadata.Object) error {
	// step 1. ensure not inner model
	if common.IsInnerModel(targetObj.GetObjectID()) {
		return kit.CCError.Error(common.CCErrTopoForbiddenToDeleteModelFailed)
	}

	cond := mapstr.New()
	if targetObj.IsCommon() {
		cond.Set(common.BKObjIDField, targetObj.ObjectID)
	}

	// step 2. ensure model has no instances
	input := &metadata.QueryCondition{Condition: cond}
	findInstResponse, err := o.clientSet.CoreService().Instance().ReadInstance(kit.Ctx, kit.Header,
		targetObj.ObjectID, input)

	if err != nil {
		blog.Errorf("[operation-obj] failed to check if it (%s) has some insts, err: %s, rid: %s",
			targetObj.ObjectID, err.Error(), kit.Rid)
		return err
	}
	if len(findInstResponse.Data.Info) != 0 {
		blog.Errorf("the object [%s] has been instantiated and cannot be deleted, rid: %s",
			targetObj.ObjectID, kit.Rid)
		return kit.CCError.Errorf(common.CCErrTopoObjectHasSomeInstsForbiddenToDelete, targetObj.ObjectID)
	}

	// step 3. ensure model has no association with other model
	or := make([]interface{}, 0)
	or = append(or, mapstr.MapStr{common.BKObjIDField: targetObj.ObjectID})
	or = append(or, mapstr.MapStr{common.AssociatedObjectIDField: targetObj.ObjectID})

	assocResult, err := o.clientSet.CoreService().Association().ReadModelAssociation(kit.Ctx, kit.Header,
		&metadata.QueryCondition{Condition: map[string]interface{}{common.BKDBOR: or}})
	if err != nil {
		blog.Errorf("check object[%s] can be deleted, but get object associate info failed, err: %v, rid: %s",
			targetObj.ObjectID, err, kit.Rid)
		return kit.CCError.Error(common.CCErrCommHTTPDoRequestFailed)
	}

	if !assocResult.Result {
		blog.Errorf("check if object[%s] can be deleted, but get object associate info failed, err: %v, rid: %s",
			targetObj.ObjectID, err, kit.Rid)
		return kit.CCError.Error(assocResult.Code)
	}

	if len(assocResult.Data.Info) != 0 {
		blog.Errorf("check if object[%s] can be deleted, but object has already associate to another one., rid: %s",
			targetObj.ObjectID, kit.Rid)
		return kit.CCError.Error(common.CCErrorTopoObjectHasAlreadyAssociated)
	}

	return nil
}

func (o *object) isFrom(kit *rest.Kit, fromObjID, toObjID string) (bool, error) {

	asstItems, err := o.clientSet.CoreService().Association().ReadModelAssociation(kit.Ctx, kit.Header,
		&metadata.QueryCondition{Condition: mapstr.MapStr{common.BKObjIDField: fromObjID}})

	if err != nil {
		blog.Errorf("[operation-asst] failed to request object controller, err: %s, rid: %s", err.Error(), kit.Rid)
		return false, kit.CCError.New(common.CCErrCommHTTPDoRequestFailed, err.Error())
	}

	if !asstItems.Result {
		blog.Errorf("[operation-asst] failed to search the object(%s) association info , err: %s, rid: %s",
			fromObjID, asstItems.ErrMsg, kit.Rid)
		return false, kit.CCError.New(asstItems.Code, asstItems.ErrMsg)
	}

	for _, asst := range asstItems.Data.Info {
		if asst.AsstObjID == toObjID {
			return true, nil
		}
	}

	return false, nil
}

func (o *object) FindObjectTopo(kit *rest.Kit, cond mapstr.MapStr) ([]metadata.ObjectTopo, error) {
	objs, err := o.FindObject(kit, cond)
	if err != nil {
		blog.Errorf("[operation-obj] failed to find object, err: %s, rid: %s", err.Error(), kit.Rid)
		return nil, err
	}

	results := make([]metadata.ObjectTopo, 0)
	for _, obj := range objs {
		asstItems, err := o.clientSet.CoreService().Association().ReadModelAssociation(kit.Ctx, kit.Header,
			&metadata.QueryCondition{Condition: mapstr.MapStr{common.BKObjIDField: obj}})

		if err != nil {
			blog.Errorf("[operation-asst] failed to request object controller, err: %s, rid: %s", err.Error(), kit.Rid)
			return nil, kit.CCError.New(common.CCErrCommHTTPDoRequestFailed, err.Error())
		}

		if !asstItems.Result {
			blog.Errorf("[operation-asst] failed to search the object(%s) association info , err: %s, rid: %s",
				obj, asstItems.ErrMsg, kit.Rid)
			return nil, kit.CCError.New(asstItems.Code, asstItems.ErrMsg)
		}

		for _, asst := range asstItems.Data.Info {

			// find association kind with association kind id.
			resp, err := o.clientSet.CoreService().Association().ReadAssociationType(kit.Ctx, kit.Header,
				&metadata.QueryCondition{
					Condition: mapstr.MapStr{common.AssociationKindIDField: asst.AsstKindID},
				})
			if err != nil {
				blog.Errorf("find object topo failed, because get association kind[%s] failed, err: %v, rid: %s",
					asst.AsstKindID, err, kit.Rid)
				return nil, kit.CCError.Errorf(common.CCErrTopoGetAssociationKindFailed, asst.AsstKindID)
			}

			if !resp.Result {
				blog.Errorf("find object topo failed, because get association kind[%s] failed, err: %v, rid: %s",
					asst.AsstKindID, resp.ErrMsg, kit.Rid)
				return nil, kit.CCError.Errorf(common.CCErrTopoGetAssociationKindFailed, asst.AsstKindID)
			}

			// should only be one association kind.
			if len(resp.Data.Info) == 0 {
				blog.Errorf("find object topo failed, because get association kind[%s] failed, "+
					"err: can not find this association kind., rid: %s", asst.AsstKindID, kit.Rid)
				return nil, kit.CCError.Errorf(common.CCErrTopoGetAssociationKindFailed, asst.AsstKindID)
			}

			asstObjs, err := o.FindObject(kit, mapstr.MapStr{common.BKObjIDField: asst.AsstObjID})
			if err != nil {
				blog.Errorf("[operation-obj] failed to find object, err: %s, rid: %s", err.Error(), kit.Rid)
				return nil, err
			}

			for _, asstObj := range asstObjs {
				tmp := metadata.ObjectTopo{}
				tmp.Label = resp.Data.Info[0].AssociationKindName
				tmp.LabelName = resp.Data.Info[0].AssociationKindName
				tmp.From.ObjID = obj.ObjectID
				tmp.From.ClassificationID = obj.ObjCls
				tmp.From.Position = obj.Position
				tmp.From.OwnerID = obj.OwnerID
				tmp.From.ObjName = obj.ObjectName
				tmp.To.OwnerID = asstObj.OwnerID
				tmp.To.ObjID = asstObj.ObjectID
				tmp.To.ClassificationID = asstObj.ObjCls
				tmp.To.Position = asstObj.Position
				tmp.To.ObjName = asstObj.ObjectName
				ok, err := o.isFrom(kit, asstObj.ObjectID, obj.ObjectID)
				if err != nil {
					return nil, err
				}

				if ok {
					tmp.Arrows = "to,from"
				} else {
					tmp.Arrows = "to"
				}

				results = append(results, tmp)
			}
		}

	}

	return results, nil
}

func (o *object) FindObject(kit *rest.Kit, cond mapstr.MapStr) ([]metadata.Object, error) {
	rsp, err := o.clientSet.CoreService().Model().ReadModel(kit.Ctx, kit.Header,
		&metadata.QueryCondition{Condition: cond},
	)
	if err != nil {
		blog.Errorf("[operation-obj] find object failed, cond: %+v, err: %s, rid: %s", cond, err.Error(), kit.Rid)
		return nil, kit.CCError.Error(common.CCErrCommHTTPDoRequestFailed)
	}

	if !rsp.Result {
		blog.Errorf("[operation-obj] failed to search the objects by the condition(%#v) , error info is %s, rid: %s",
			cond, rsp.ErrMsg, kit.Rid)
		return nil, kit.CCError.New(rsp.Code, rsp.ErrMsg)
	}

	return rsp.Data.Info, nil
}

func (o *object) UpdateObject(kit *rest.Kit, data mapstr.MapStr, id int64) error {

	obj, err := o.IsValid(kit, true, data)
	if err != nil {
		blog.Errorf("valid data failed, data: %v, err: %s, rid: %s", data, err.Error(), kit.Rid)
		return err
	}

	obj.ID = id

	// remove unchangeable fields.
	data.Remove(metadata.ModelFieldObjectID)
	data.Remove(metadata.ModelFieldID)
	data.Remove(metadata.ModelFieldObjCls)

	// generate audit log of object attribute group.
	audit := auditlog.NewObjectAuditLog(o.clientSet.CoreService())
	generateAuditParameter := auditlog.NewGenerateAuditCommonParameter(kit, metadata.AuditUpdate).WithUpdateFields(data)
	auditLog, err := audit.GenerateAuditLog(generateAuditParameter, obj.ID, nil)
	if err != nil {
		blog.Errorf("generate audit log failed before update object, objName: %s, err: %v, rid: %s",
			obj.ObjectName, err, kit.Rid)
		return err
	}

	cond := mapstr.New()
	if len(obj.ObjectID) != 0 {
		cond.Set(common.BKObjIDField, obj.ObjectID)
	} else {
		cond.Set(metadata.ModelFieldID, obj.ID)
	}

	objs, err := o.clientSet.CoreService().Model().ReadModel(kit.Ctx, kit.Header,
		&metadata.QueryCondition{Condition: cond},
	)

	if err != nil {
		blog.Errorf("get object by id[%d] failed, err: %s, rid: %s", id, err.Error(), kit.Rid)
		return err
	}

	if !objs.Result {
		blog.Errorf("get object by id[%d] failed, err: %s, rid: %s", id, objs.ErrMsg, kit.Rid)
		return kit.CCError.New(objs.Code, objs.ErrMsg)
	}

	if len(objs.Data.Info) != 1 {
		blog.Errorf("the object(%#v) is not unique, rid: %s", data, kit.Rid)
		return kit.CCError.Errorf(common.CCErrCommDuplicateItem, obj.ObjectName)
	}

	input := metadata.UpdateOption{
		Condition: mapstr.MapStr{common.BKFieldID: objs.Data.Info[0].ID},
		Data:      data,
	}

	rsp, err := o.clientSet.CoreService().Model().UpdateModel(kit.Ctx, kit.Header, &input)
	if err != nil {
		blog.Errorf("failed to request the object controller, error info is %s, rid: %s", err.Error(), kit.Rid)
		return kit.CCError.Error(common.CCErrCommHTTPDoRequestFailed)
	}

	if !rsp.Result {
		blog.Errorf("failed to search the object(%s), error info is %s, rid: %s", obj.ObjectID, rsp.ErrMsg, kit.Rid)
		return kit.CCError.New(rsp.Code, rsp.ErrMsg)
	}

	// save audit log.
	if err := audit.SaveAuditLog(kit, *auditLog); err != nil {
		blog.Errorf("update object %s success, but save audit log failed, err: %v, rid: %s",
			obj.ObjectName, err, kit.Rid)
		return err
	}

	return nil
}

func (o *object) IsValid(kit *rest.Kit, isUpdate bool, data mapstr.MapStr) (*metadata.Object, error) {

	obj := new(metadata.Object)
	if err := mapstruct.Decode2Struct(data, obj); err != nil {
		blog.Errorf("parse object failed, err: %v, input: %#v, rid: %s", err, data, kit.Rid)
		return nil, kit.CCError.CCErrorf(common.CCErrCommJSONUnmarshalFailed)
	}

	if !isUpdate || data.Exists(metadata.ModelFieldObjectID) {

		if err := util.ValidModelIDField(data[metadata.ModelFieldObjectID],
			metadata.ModelFieldObjectID, kit.CCError); err != nil {
			blog.Errorf("failed to valid the object id(%s), rid: %s", metadata.ModelFieldObjectID, kit.Rid)
			return nil, err
		}
	}

	if !isUpdate || data.Exists(metadata.ModelFieldObjectName) {
		if err := util.ValidModelNameField(data[metadata.ModelFieldObjectName],
			metadata.ModelFieldObjectName, kit.CCError); err != nil {
			blog.Errorf("failed to valid the object name(%s), rid: %s", metadata.ModelFieldObjectName, kit.Rid)
			return nil, kit.CCError.New(common.CCErrCommParamsIsInvalid, metadata.ModelFieldObjectName+" "+err.Error())
		}
	}

	if !isUpdate && !data.Exists(metadata.ModelFieldObjCls) {
		return nil, kit.CCError.CCErrorf(common.CCErrCommParamsIsInvalid, metadata.ModelFieldObjCls)
	}

	if !isUpdate && !metadata.IsCommon(obj.ObjectID) {
		return nil, kit.CCError.New(common.CCErrCommParamsIsInvalid,
			fmt.Sprintf("'%s' the built-in object id, please use a new one", obj.ObjectID))
	}

	return obj, nil
}
