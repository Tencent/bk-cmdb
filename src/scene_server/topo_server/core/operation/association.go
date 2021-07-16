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

package operation

import (
	"context"
	"fmt"

	"configcenter/src/ac/extensions"
	"configcenter/src/apimachinery"
	"configcenter/src/common"
	"configcenter/src/common/auditlog"
	"configcenter/src/common/blog"
	"configcenter/src/common/condition"
	"configcenter/src/common/errors"
	"configcenter/src/common/http/rest"
	"configcenter/src/common/language"
	"configcenter/src/common/mapstr"
	"configcenter/src/common/metadata"
	"configcenter/src/scene_server/topo_server/core/inst"
	"configcenter/src/scene_server/topo_server/core/model"
)

// AssociationOperationInterface association operation methods
type AssociationOperationInterface interface {
	CreateMainlineAssociation(kit *rest.Kit, data *metadata.Association, maxTopoLevel int) (model.Object, error)
	DeleteMainlineAssociation(kit *rest.Kit, objID string) error
	SearchMainlineAssociationTopo(kit *rest.Kit, targetObj model.Object) ([]*metadata.MainlineObjectTopo, error)
	SearchMainlineAssociationInstTopo(kit *rest.Kit, objID string, instID int64, withStatistics bool, withDefault bool) ([]*metadata.TopoInstRst, errors.CCError)
	SearchTopoNodeHostAndServiceInstCount(kit *rest.Kit, instID int64, input *metadata.SearchBizTopoNodeHostAndServiceInstCountOption) ([]*metadata.TopoInstNodeHostAndServiceInstCount, errors.CCError)
	IsMainlineObject(kit *rest.Kit, objID string) (bool, error)

	CreateCommonAssociation(kit *rest.Kit, data *metadata.Association) (*metadata.Association, error)
	DeleteAssociationWithPreCheck(kit *rest.Kit, associationID int64) error
	UpdateAssociation(kit *rest.Kit, data mapstr.MapStr, assoID int64) error
	SearchObjectAssociation(kit *rest.Kit, objID string) ([]metadata.Association, error)
	SearchObjectsAssociation(kit *rest.Kit, objIDs []string) ([]metadata.Association, error)

	DeleteAssociation(kit *rest.Kit, cond condition.Condition) error
	SearchInstAssociation(kit *rest.Kit, query *metadata.QueryInput) ([]metadata.InstAsst, error)
	SearchInstAssociationList(kit *rest.Kit, query *metadata.QueryCondition) ([]metadata.InstAsst, uint64, error)
	SearchInstAssociationUIList(kit *rest.Kit, objID string, query *metadata.QueryCondition) (result interface{}, asstCnt uint64, err error)
	SearchInstAssociationSingleObjectInstInfo(kit *rest.Kit, returnInstInfoObjID string, query *metadata.QueryCondition) (result []metadata.InstBaseInfo, cnt uint64, err error)
	CreateCommonInstAssociation(kit *rest.Kit, data *metadata.InstAsst) error
	DeleteInstAssociation(kit *rest.Kit, cond map[string]interface{}) error
	CheckAssociation(kit *rest.Kit, objectID string, instID int64) error
	CheckAssociations(kit *rest.Kit, objectID string, instIDs []int64) error

	// 关联关系改造后的接口
	SearchObjectAssocWithAssocKindList(kit *rest.Kit, asstKindIDs []string) (resp *metadata.AssociationList, err error)
	SearchType(kit *rest.Kit, request *metadata.SearchAssociationTypeRequest) (resp *metadata.SearchAssociationTypeResult, err error)
	CreateType(kit *rest.Kit, request *metadata.AssociationKind) (resp *metadata.CreateAssociationTypeResult, err error)
	UpdateType(kit *rest.Kit, asstTypeID int64, request *metadata.UpdateAssociationTypeRequest) (resp *metadata.UpdateAssociationTypeResult, err error)
	DeleteType(kit *rest.Kit, asstTypeID int64) (resp *metadata.DeleteAssociationTypeResult, err error)

	SearchObject(kit *rest.Kit, request *metadata.SearchAssociationObjectRequest) (resp *metadata.SearchAssociationObjectResult, err error)
	CreateObject(kit *rest.Kit, request *metadata.Association) (resp *metadata.CreateAssociationObjectResult, err error)
	UpdateObject(kit *rest.Kit, asstID int, request *metadata.UpdateAssociationObjectRequest) (resp *metadata.UpdateAssociationObjectResult, err error)
	DeleteObject(kit *rest.Kit, asstID int) (resp *metadata.DeleteAssociationObjectResult, err error)

	SearchInst(kit *rest.Kit, request *metadata.SearchAssociationInstRequest) (resp *metadata.SearchAssociationInstResult, err error)
	SearchAssociationRelatedInst(kit *rest.Kit, request *metadata.SearchAssociationRelatedInstRequest) (resp *metadata.SearchAssociationInstResult, err error)
	CreateInst(kit *rest.Kit, request *metadata.CreateAssociationInstRequest) (resp *metadata.CreateAssociationInstResult, err error)
	DeleteInst(kit *rest.Kit, assoIDList []int64, bkObjId string) (resp *metadata.DeleteAssociationInstResult, err error)

	ImportInstAssociation(ctx context.Context, kit *rest.Kit, objID string, importData map[int]metadata.ExcelAssociation, languageIf language.CCLanguageIf) (resp metadata.ResponeImportAssociationData, err error)

	SetProxy(cls ClassificationOperationInterface, obj ObjectOperationInterface, grp GroupOperationInterface, attr AttributeOperationInterface, inst InstOperationInterface, targetModel model.Factory, targetInst inst.Factory)
}

// NewAssociationOperation create a new association operation instance
func NewAssociationOperation(client apimachinery.ClientSetInterface, authManager *extensions.AuthManager) AssociationOperationInterface {
	return &association{
		clientSet:   client,
		authManager: authManager,
	}
}

type association struct {
	clientSet    apimachinery.ClientSetInterface
	authManager  *extensions.AuthManager
	cls          ClassificationOperationInterface
	obj          ObjectOperationInterface
	grp          GroupOperationInterface
	attr         AttributeOperationInterface
	inst         InstOperationInterface
	modelFactory model.Factory
	instFactory  inst.Factory
}

func (assoc *association) SetProxy(cls ClassificationOperationInterface, obj ObjectOperationInterface, grp GroupOperationInterface, attr AttributeOperationInterface, inst InstOperationInterface, targetModel model.Factory, targetInst inst.Factory) {
	assoc.cls = cls
	assoc.obj = obj
	assoc.attr = attr
	assoc.inst = inst
	assoc.grp = grp
	assoc.modelFactory = targetModel
	assoc.instFactory = targetInst
}

func (assoc *association) SearchObjectAssociation(kit *rest.Kit, objID string) ([]metadata.Association, error) {
	cond := condition.CreateCondition()
	if 0 != len(objID) {
		cond.Field(common.BKObjIDField).Eq(objID)
	}

	fCond := cond.ToMapStr()
	rsp, err := assoc.clientSet.CoreService().Association().ReadModelAssociation(kit.Ctx, kit.Header, &metadata.QueryCondition{Condition: fCond})
	if nil != err {
		blog.Errorf("[operation-asst] failed to request object controller, err: %s, rid: %s", err.Error(), kit.Rid)
		return nil, kit.CCError.New(common.CCErrCommHTTPDoRequestFailed, err.Error())
	}

	if !rsp.Result {
		blog.Errorf("[operation-asst] failed to search the object(%s) association info , err: %s, rid: %s", objID, rsp.ErrMsg, kit.Rid)
		return nil, kit.CCError.New(rsp.Code, rsp.ErrMsg)
	}

	return rsp.Data.Info, nil
}

func (assoc *association) SearchObjectsAssociation(kit *rest.Kit, objIDs []string) ([]metadata.Association, error) {
	cond := condition.CreateCondition()
	if 0 != len(objIDs) {
		cond.Field(common.BKObjIDField).In(objIDs)
	}

	fCond := cond.ToMapStr()
	rsp, err := assoc.clientSet.CoreService().Association().ReadModelAssociation(kit.Ctx, kit.Header, &metadata.QueryCondition{Condition: fCond})
	if nil != err {
		blog.Errorf("[operation-asst] failed to request object controller, err: %s, rid: %s", err.Error(), kit.Rid)
		return nil, kit.CCError.New(common.CCErrCommHTTPDoRequestFailed, err.Error())
	}

	if !rsp.Result {
		blog.Errorf("[operation-asst] failed to search the object(%s) association info , err: %s, rid: %s", objIDs, rsp.ErrMsg, kit.Rid)
		return nil, kit.CCError.New(rsp.Code, rsp.ErrMsg)
	}

	return rsp.Data.Info, nil
}

func (assoc *association) SearchInstAssociation(kit *rest.Kit, query *metadata.QueryInput) ([]metadata.InstAsst, error) {
	intput, err := mapstr.NewFromInterface(query.Condition)
	rsp, err := assoc.clientSet.CoreService().Association().ReadInstAssociation(kit.Ctx, kit.Header, &metadata.QueryCondition{Condition: intput})
	if nil != err {
		blog.Errorf("[operation-asst] failed to request object controller, err: %s, rid: %s", err.Error(), kit.Rid)
		return nil, kit.CCError.New(common.CCErrCommHTTPDoRequestFailed, err.Error())
	}

	if !rsp.Result {
		blog.Errorf("[operation-asst] failed to search the association info, query: %#v, err: %s, rid: %s", query, rsp.ErrMsg, kit.Rid)
		return nil, kit.CCError.New(rsp.Code, rsp.ErrMsg)
	}

	return rsp.Data.Info, nil
}

// CreateCommonAssociation create a common association, in topo model scene, which doesn't include bk_mainline association type
func (assoc *association) CreateCommonAssociation(kit *rest.Kit, data *metadata.Association) (*metadata.Association, error) {
	if data.AsstKindID == common.AssociationKindMainline {
		return nil, kit.CCError.Error(common.CCErrorTopoAssociationKindMainlineUnavailable)
	}
	if len(data.AsstKindID) == 0 || len(data.AsstObjID) == 0 || len(data.ObjectID) == 0 {
		blog.Errorf("[operation-asst] failed to create the association , association kind id associate/object id is required, rid: %s", kit.Rid)
		return nil, kit.CCError.Error(common.CCErrorTopoAssociationMissingParameters)
	}

	// if the on delete action is empty, set none as default.
	if len(data.OnDelete) == 0 {
		data.OnDelete = metadata.NoAction
	}

	// check if this association has already exist,
	// if yes, it's not allowed to create this association

	//  check the association
	cond := condition.CreateCondition()
	cond.Field(common.AssociatedObjectIDField).Eq(data.AsstObjID)
	cond.Field(common.BKObjIDField).Eq(data.ObjectID)
	cond.Field(common.AssociationKindIDField).Eq(data.AsstKindID)

	rsp, err := assoc.clientSet.CoreService().Association().ReadModelAssociation(kit.Ctx, kit.Header, &metadata.QueryCondition{Condition: cond.ToMapStr()})
	if nil != err {
		blog.Errorf("[operation-asst] failed to request object controller, err: %s, rid: %s", err.Error(), kit.Rid)
		return nil, kit.CCError.New(common.CCErrCommHTTPDoRequestFailed, err.Error())
	}

	if !rsp.Result {
		blog.Errorf("[operation-asst] failed to create the association (%#v) , err: %s, rid: %s", cond.ToMapStr(), rsp.ErrMsg, kit.Rid)
		return nil, kit.CCError.New(rsp.Code, rsp.ErrMsg)
	}
	if len(rsp.Data.Info) > 0 {
		blog.Errorf("[operation-asst] failed to create the association (%#v) , the associations %s->%s already exist , rid: %s", kit.Rid,
			cond.ToMapStr(), data.ObjectID, data.AsstObjID)
		return nil, kit.CCError.Errorf(common.CCErrTopoAssociationAlreadyExist, data.ObjectID, data.AsstObjID)
	}

	// check source object exists
	if err := assoc.obj.IsValidObject(kit, data.ObjectID); nil != err {
		blog.Errorf("[operation-asst] the object(%s) is invalid, err: %s, rid: %s", data.ObjectID, err.Error(), kit.Rid)
		return nil, err
	}

	if err := assoc.obj.IsValidObject(kit, data.AsstObjID); nil != err {
		blog.Errorf("[operation-asst] the object(%s) is invalid, err: %s, rid: %s", data.AsstObjID, err.Error(), kit.Rid)
		return nil, err
	}

	// create a new
	rspAsst, err := assoc.clientSet.CoreService().Association().CreateModelAssociation(kit.Ctx, kit.Header, &metadata.CreateModelAssociation{Spec: *data})
	if nil != err {
		blog.Errorf("[operation-asst] failed to request object controller, err: %s, rid: %s", err.Error(), kit.Rid)
		return nil, kit.CCError.New(common.CCErrCommHTTPDoRequestFailed, err.Error())
	}

	if !rspAsst.Result {
		blog.Errorf("[operation-asst] failed to create the association (%#v) , err: %s, rid: %s", data, rspAsst.ErrMsg, kit.Rid)
		return nil, kit.CCError.New(rspAsst.Code, rspAsst.ErrMsg)
	}

	data.ID = int64(rspAsst.Data.Created.ID)
	return data, nil
}

func (assoc *association) DeleteInstAssociation(kit *rest.Kit, cond map[string]interface{}) error {

	rsp, err := assoc.clientSet.CoreService().Association().DeleteInstAssociation(kit.Ctx, kit.Header,
		&metadata.DeleteOption{Condition: cond})
	if nil != err {
		blog.Errorf("[operation-asst] failed to request object controller, err: %s, rid: %s", err.Error(), kit.Rid)
		return kit.CCError.New(common.CCErrCommHTTPDoRequestFailed, err.Error())
	}

	if !rsp.Result {
		blog.Errorf("[operation-asst] failed to delete the inst association info , err: %s, rid: %s", rsp.ErrMsg, kit.Rid)
		return kit.CCError.New(rsp.Code, rsp.ErrMsg)
	}

	return nil
}

func (assoc *association) CreateCommonInstAssociation(kit *rest.Kit, data *metadata.InstAsst) error {
	// create a new
	rspAsst, err := assoc.clientSet.CoreService().Association().CreateInstAssociation(kit.Ctx, kit.Header, &metadata.CreateOneInstanceAssociation{Data: *data})
	if nil != err {
		blog.Errorf("[operation-asst] failed to request object controller, err: %s, rid: %s", err.Error(), kit.Rid)
		return kit.CCError.New(common.CCErrCommHTTPDoRequestFailed, err.Error())
	}

	if !rspAsst.Result {
		blog.Errorf("[operation-asst] failed to create the association (%#v) , err: %s, rid: %s", data, rspAsst.ErrMsg, kit.Rid)
		return kit.CCError.New(rspAsst.Code, rspAsst.ErrMsg)
	}

	return nil
}

func (assoc *association) IsMainlineObject(kit *rest.Kit, objID string) (bool, error) {
	cond := mapstr.MapStr{common.AssociationKindIDField: common.AssociationKindMainline}
	asst, err := assoc.clientSet.CoreService().Association().ReadModelAssociation(kit.Ctx, kit.Header,
		&metadata.QueryCondition{Condition: cond})
	if err != nil {
		return false, err
	}

	if !asst.Result {
		return false, errors.New(asst.Code, asst.ErrMsg)
	}

	if len(asst.Data.Info) <= 0 {
		return false, fmt.Errorf("model association [%+v] not found", cond)
	}

	for _, mainline := range asst.Data.Info {
		if mainline.ObjectID == objID || mainline.AsstObjID == objID {
			return true, nil
		}
	}

	return false, nil
}

func (assoc *association) DeleteAssociationWithPreCheck(kit *rest.Kit, associationID int64) error {
	// if this association has already been instantiated, then this association should not be deleted.
	// get the association with id at first.
	cond := condition.CreateCondition()
	cond.Field(metadata.AssociationFieldAssociationId).Eq(associationID)
	result, err := assoc.clientSet.CoreService().Association().ReadModelAssociation(kit.Ctx, kit.Header, &metadata.QueryCondition{Condition: cond.ToMapStr()})
	if err != nil {
		blog.Errorf("[operation-asst] delete association with id[%d], but get this association for pre check failed, err: %v, rid: %s", associationID, err, kit.Rid)
		return kit.CCError.New(common.CCErrCommHTTPDoRequestFailed, err.Error())
	}

	if !result.Result {
		blog.Errorf("[operation-asst] delete association with id[%d], but get this association for pre check failed, err: %s, rid: %s", associationID, result.ErrMsg, kit.Rid)
		return kit.CCError.New(result.Code, result.ErrMsg)
	}

	if len(result.Data.Info) == 0 {
		blog.Errorf("[operation-asst] delete association with id[%d], but can not find this association, return now., rid: %s", associationID, kit.Rid)
		return nil
	}

	if len(result.Data.Info) > 1 {
		blog.Errorf("[operation-asst] delete association with id[%d], but got multiple association, rid: %s", associationID, kit.Rid)
		return kit.CCError.Error(common.CCErrTopoGotMultipleAssociationInstance)
	}

	if result.Data.Info[0].AsstKindID == common.AssociationKindMainline {
		return kit.CCError.Error(common.CCErrorTopoAssociationKindMainlineUnavailable)
	}

	// find instance(s) belongs to this association
	cond = condition.CreateCondition()
	cond.Field(common.AssociationObjAsstIDField).Eq(result.Data.Info[0].AssociationName)
	query := metadata.QueryInput{Condition: cond.ToMapStr()}
	insts, err := assoc.SearchInstAssociation(kit, &query)
	if err != nil {
		blog.Errorf("[operation-asst] delete association with id[%d], but association instance(s) failed, err: %v, rid: %s", associationID, err, kit.Rid)
		return kit.CCError.New(common.CCErrCommHTTPDoRequestFailed, err.Error())
	}

	if len(insts) != 0 {
		// object association has already been instantiated, association can not be deleted.
		blog.Errorf("[operation-asst] delete association with id[%d], but has multiple instances, can not be deleted., rid: %s", associationID, kit.Rid)
		return kit.CCError.Error(common.CCErrTopoAssociationHasAlreadyBeenInstantiated)
	}

	// TODO: check association on_delete action before really delete this association.
	// all the pre check has finished, delete the association now.
	cond = condition.CreateCondition()
	cond.Field(metadata.AssociationFieldAssociationId).Eq(associationID)
	return assoc.DeleteAssociation(kit, cond)
}

func (assoc *association) DeleteAssociation(kit *rest.Kit, cond condition.Condition) error {
	rsp, err := assoc.clientSet.CoreService().Association().ReadModelAssociation(kit.Ctx, kit.Header, &metadata.QueryCondition{Condition: cond.ToMapStr()})
	if nil != err {
		blog.Errorf("delete object association, but get association with cond[%v] failed, err: %v, rid: %s", cond.ToMapStr(), err, kit.Rid)
		return kit.CCError.Error(common.CCErrCommHTTPDoRequestFailed)
	}

	if !rsp.Result {
		blog.Errorf("delete object association, but get association with cond[%v] failed, err: %s, rid: %s", cond.ToMapStr(), rsp.ErrMsg, kit.Rid)
		return kit.CCError.New(rsp.Code, rsp.ErrMsg)
	}

	if len(rsp.Data.Info) < 1 {
		// we assume this association has already been deleted.
		blog.Warnf("delete object association, but can not get association with cond[%v] , rid: %s", cond.ToMapStr(), kit.Rid)
		return nil
	}

	// a pre-defined association can not be updated.
	if nil != rsp.Data.Info[0].IsPre && *rsp.Data.Info[0].IsPre {
		blog.Errorf("delete object association with cond[%v], but it's a pre-defined association, can not be deleted., rid: %s", cond.ToMapStr(), kit.Rid)
		return kit.CCError.Error(common.CCErrorTopoDeletePredefinedAssociation)
	}

	// delete the object association
	result, err := assoc.clientSet.CoreService().Association().DeleteModelAssociation(kit.Ctx, kit.Header, &metadata.DeleteOption{Condition: cond.ToMapStr()})
	if nil != err {
		blog.Errorf("[operation-asst] failed to request object controller, err: %s, rid: %s", err.Error(), kit.Rid)
		return kit.CCError.Error(common.CCErrCommHTTPDoRequestFailed)
	}

	if !result.Result {
		blog.Errorf("[operation-asst] failed to create the association (%#v) , err: %s, rid: %s", cond.ToMapStr(), result.ErrMsg, kit.Rid)
		return kit.CCError.Error(result.Code)
	}

	return nil
}

func (assoc *association) UpdateAssociation(kit *rest.Kit, data mapstr.MapStr, assoID int64) error {
	asst := &metadata.Association{}
	err := data.MarshalJSONInto(asst)
	if err != nil {
		blog.Errorf("[operation-asst] update association with  %s, rid: %s", err.Error(), kit.Rid)
		return kit.CCError.Error(common.CCErrCommJSONUnmarshalFailed)
	}

	if field, can := asst.CanUpdate(); !can {
		blog.Warnf("update association[%d], but request to update a forbidden update field[%s]., rid: %s", assoID, field, kit.Rid)
		return kit.CCError.Error(common.CCErrorTopoObjectAssociationUpdateForbiddenFields)
	}

	cond := condition.CreateCondition()
	cond.Field(metadata.AssociationFieldAssociationId).Eq(assoID)

	rsp, err := assoc.clientSet.CoreService().Association().ReadModelAssociation(kit.Ctx, kit.Header, &metadata.QueryCondition{Condition: cond.ToMapStr()})
	if nil != err {
		blog.Errorf("[operation-asst] failed to request object controller, err: %s, rid: %s", err.Error(), kit.Rid)
		return kit.CCError.Error(common.CCErrCommHTTPDoRequestFailed)
	}

	if !rsp.Result {
		blog.Errorf("[operation-asst] failed to update the association (%#v) , err: %s, rid: %s", cond.ToMapStr(), rsp.ErrMsg, kit.Rid)
		return kit.CCError.New(rsp.Code, rsp.ErrMsg)
	}

	if len(rsp.Data.Info) < 1 {
		blog.Errorf("[operation-asst] failed to update the object association , id %d not found, rid: %s", assoID, kit.Rid)
		return kit.CCError.Error(common.CCErrorTopoObjectAssociationNotExist)
	}

	// a pre-defined association can not be updated.
	if nil != rsp.Data.Info[0].IsPre && *rsp.Data.Info[0].IsPre {
		blog.Errorf("update object association[%d], but it's a pre-defined association, can not be updated., rid: %s", assoID, kit.Rid)
		return kit.CCError.Error(common.CCErrorTopoUpdatePredefinedAssociation)
	}

	// check object exists
	if err := assoc.obj.IsValidObject(kit, rsp.Data.Info[0].ObjectID); nil != err {
		blog.Errorf("[operation-asst] the object(%s) is invalid, error info is %s, rid: %s", rsp.Data.Info[0].ObjectID, err.Error(), kit.Rid)
		return err
	}

	if err := assoc.obj.IsValidObject(kit, rsp.Data.Info[0].AsstObjID); nil != err {
		blog.Errorf("[operation-asst] the object(%s) is invalid, error info is %s, rid: %s", rsp.Data.Info[0].AsstObjID, err.Error(), kit.Rid)
		return err
	}

	updateopt := metadata.UpdateOption{
		Condition: condition.CreateCondition().Field(common.BKFieldID).Eq(assoID).ToMapStr(),
		Data:      data,
	}
	rspAsst, err := assoc.clientSet.CoreService().Association().UpdateModelAssociation(kit.Ctx, kit.Header, &updateopt)
	if nil != err {
		blog.Errorf("[operation-asst] failed to request object controller, err: %s, rid: %s", err.Error(), kit.Rid)
		return kit.CCError.Error(common.CCErrCommHTTPDoRequestFailed)
	}

	if !rspAsst.Result {
		blog.Errorf("[operation-asst] failed to create the association (%#v) , err: %s, rid: %s", data, rspAsst.ErrMsg, kit.Rid)
		return kit.CCError.Error(rspAsst.Code)
	}

	return nil
}

// CheckAssociation and return error if the instance exist association
func (assoc *association) CheckAssociation(kit *rest.Kit, objectID string, instID int64) error {
	cond := condition.CreateCondition()
	or := cond.NewOR()
	or.Item(mapstr.MapStr{common.BKObjIDField: objectID, common.BKInstIDField: instID})
	or.Item(mapstr.MapStr{common.BKAsstObjIDField: objectID, common.BKAsstInstIDField: instID})
	asst, err := assoc.SearchInstAssociation(kit, &metadata.QueryInput{Condition: cond.ToMapStr()})
	if nil != err {
		return err
	}
	if len(asst) == 0 {
		return nil
	}
	for _, asst := range asst {
		var errCheck error
		isInstExist := false
		if asst.ObjectID == objectID && asst.InstID == instID {
			isInstExist, errCheck = assoc.CheckAssociationInstExist(kit, asst.AsstObjectID, asst.AsstInstID)
		} else if asst.AsstObjectID == objectID && asst.AsstInstID == instID {
			isInstExist, errCheck = assoc.CheckAssociationInstExist(kit, asst.ObjectID, asst.InstID)
		} else {
			return kit.CCError.New(common.CCErrCommDBSelectFailed, "instance is not associated in selected association")
		}
		if errCheck != nil {
			return errCheck
		}
		if isInstExist {
			return kit.CCError.CCErrorf(common.CCErrTopoInstHasBeenAssociation, instID)
		}
	}

	return nil
}

func (assoc *association) CheckAssociationInstExist(kit *rest.Kit, objectID string, instID int64) (bool, error) {
	instIDField := common.GetInstIDField(objectID)
	instRsp, err := assoc.clientSet.CoreService().Instance().ReadInstance(kit.Ctx, kit.Header, objectID,
		&metadata.QueryCondition{Condition: mapstr.MapStr{instIDField: instID}})
	if err != nil {
		return false, kit.CCError.Error(common.CCErrObjectSelectInstFailed)
	}
	if !instRsp.Result {
		return false, kit.CCError.New(instRsp.Code, instRsp.ErrMsg)
	}
	if len(instRsp.Data.Info) > 0 {
		return true, nil
	}
	// 实例不存在，删除实例的关联关系
	if err := assoc.DeleteAssociationDirtyData(kit, objectID, instID); err != nil {
		return false, err
	}
	return false, nil
}

func (assoc *association) DeleteAssociationDirtyData(kit *rest.Kit, objectID string, instID int64) error {
	cond := condition.CreateCondition()
	or := cond.NewOR()
	or.Item(mapstr.MapStr{common.BKObjIDField: objectID, common.BKInstIDField: instID})
	or.Item(mapstr.MapStr{common.BKAsstObjIDField: objectID, common.BKAsstInstIDField: instID})
	if delErr := assoc.DeleteInstAssociation(kit, cond.ToMapStr()); delErr != nil {
		return delErr
	}

	return nil
}

// CheckAssociations returns error if the instances has associations with exist instances, clear dirty associations
func (assoc *association) CheckAssociations(kit *rest.Kit, objectID string, instIDs []int64) error {
	if len(instIDs) == 0 {
		return nil
	}

	// get all associations for the instances
	cond := map[string]interface{}{
		common.BKDBOR: []map[string]interface{}{
			{common.BKObjIDField: objectID, common.BKInstIDField: map[string]interface{}{common.BKDBIN: instIDs}},
			{common.BKAsstObjIDField: objectID, common.BKAsstInstIDField: map[string]interface{}{common.BKDBIN: instIDs}},
		},
	}

	associations, err := assoc.SearchInstAssociation(kit, &metadata.QueryInput{Condition: cond, Limit: common.BKNoLimit})
	if err != nil {
		blog.ErrorJSON("search instance associations failed, err: %s, cond: %s, rid: %s", err, cond, kit.Rid)
		return err
	}

	if len(associations) == 0 {
		return nil
	}

	instIDExistsMap := make(map[int64]bool)
	for _, instID := range instIDs {
		instIDExistsMap[instID] = true
	}

	// get all associated inst IDs grouped by object ID, then check if any inst exists, clear not exist one's assts
	asstObjInstIDsMap := make(map[string][]int64)
	for _, asst := range associations {
		if asst.ObjectID == objectID && instIDExistsMap[asst.InstID] {
			asstObjInstIDsMap[asst.AsstObjectID] = append(asstObjInstIDsMap[asst.AsstObjectID], asst.AsstInstID)
		} else if asst.AsstObjectID == objectID && instIDExistsMap[asst.AsstInstID] {
			asstObjInstIDsMap[asst.ObjectID] = append(asstObjInstIDsMap[asst.ObjectID], asst.InstID)
		}
	}

	for asstObjID, asstInstIDs := range asstObjInstIDsMap {
		query := &metadata.QueryCondition{
			Condition: map[string]interface{}{
				common.GetInstIDField(asstObjID): map[string]interface{}{common.BKDBIN: asstInstIDs},
			},
			Page: metadata.BasePage{Limit: 1},
		}

		asstInstRsp, err := assoc.clientSet.CoreService().Instance().ReadInstance(kit.Ctx, kit.Header, asstObjID, query)
		if err != nil {
			blog.ErrorJSON("check instance existence failed, err: %s, query: %s, rid: %s", err, query, kit.Rid)
			return kit.CCError.Error(common.CCErrObjectSelectInstFailed)
		}
		if err := asstInstRsp.CCError(); err != nil {
			blog.ErrorJSON("check instance existence failed, err: %s, query: %s, rid: %s", err, query, kit.Rid)
			return err
		}

		if len(asstInstRsp.Data.Info) > 0 {
			return kit.CCError.CCError(common.CCErrorInstHasAsst)
		}

		deleteAsstCond := map[string]interface{}{
			common.BKDBOR: []map[string]interface{}{
				{common.BKObjIDField: asstObjID, common.BKInstIDField: map[string]interface{}{common.BKDBIN: asstInstIDs}},
				{common.BKAsstObjIDField: asstObjID, common.BKAsstInstIDField: map[string]interface{}{common.BKDBIN: asstInstIDs}},
			},
		}

		if err := assoc.DeleteInstAssociation(kit, deleteAsstCond); err != nil {
			blog.ErrorJSON("delete dirty assts failed, err: %s, cond: %s, rid: %s", err, deleteAsstCond, kit.Rid)
			return err
		}
	}
	return nil
}

// 关联关系改造后的接口
func (assoc *association) SearchObjectAssocWithAssocKindList(kit *rest.Kit, asstKindIDs []string) (resp *metadata.AssociationList, err error) {
	if len(asstKindIDs) == 0 {
		return &metadata.AssociationList{Associations: make([]metadata.AssociationDetail, 0)}, nil
	}

	asso := make([]metadata.AssociationDetail, 0)
	for _, id := range asstKindIDs {
		cond := condition.CreateCondition()
		cond.Field(common.AssociationKindIDField).Eq(id)

		r, err := assoc.clientSet.CoreService().Association().ReadModelAssociation(kit.Ctx, kit.Header, &metadata.QueryCondition{Condition: cond.ToMapStr()})
		if err != nil {
			blog.Errorf("get object association list with association kind[%s] failed, err: %v, rid: %s", id, err, kit.Rid)
			return nil, kit.CCError.Error(common.CCErrCommHTTPDoRequestFailed)
		}

		if !r.Result {
			blog.Errorf("get object association list with association kind[%s] failed, err: %v, rid: %s", id, r.ErrMsg, kit.Rid)
			return nil, kit.CCError.Errorf(r.Code, r.ErrMsg)
		}

		asso = append(asso, metadata.AssociationDetail{AssociationKindID: id, Associations: r.Data.Info})
	}

	return &metadata.AssociationList{Associations: asso}, nil
}

func (assoc *association) SearchType(kit *rest.Kit, request *metadata.SearchAssociationTypeRequest) (resp *metadata.SearchAssociationTypeResult, err error) {
	input := metadata.QueryCondition{
		Condition: request.Condition,
		Page:      metadata.BasePage{Limit: request.Limit, Start: request.Start, Sort: request.Sort},
	}

	return assoc.clientSet.CoreService().Association().ReadAssociationType(kit.Ctx, kit.Header, &input)
}

func (assoc *association) CreateType(kit *rest.Kit, request *metadata.AssociationKind) (resp *metadata.CreateAssociationTypeResult, err error) {

	rsp, err := assoc.clientSet.CoreService().Association().CreateAssociationType(kit.Ctx, kit.Header, &metadata.CreateAssociationKind{Data: *request})
	if err != nil {
		blog.Errorf("create association type failed, kind id: %s, err: %v, rid: %s", request.AssociationKindID, err, kit.Rid)
		return nil, kit.CCError.New(common.CCErrTopoCreateAssocKindFailed, err.Error())
	}
	if rsp.Result == false || rsp.Code != 0 {
		blog.ErrorJSON("create association type failed, request: %s, response: %s, rid: %s", request, rsp, kit.Rid)
		return nil, errors.NewCCError(rsp.Code, rsp.ErrMsg)
	}
	resp = &metadata.CreateAssociationTypeResult{BaseResp: rsp.BaseResp}
	resp.Data.ID = int64(rsp.Data.Created.ID)
	request.ID = resp.Data.ID

	return resp, nil

}

func (assoc *association) UpdateType(kit *rest.Kit, asstTypeID int64, request *metadata.UpdateAssociationTypeRequest) (resp *metadata.UpdateAssociationTypeResult, err error) {

	input := metadata.UpdateOption{
		Condition: condition.CreateCondition().Field(common.BKFieldID).Eq(asstTypeID).ToMapStr(),
		Data:      mapstr.NewFromStruct(request, "json"),
	}

	rsp, err := assoc.clientSet.CoreService().Association().UpdateAssociationType(kit.Ctx, kit.Header, &input)
	if err != nil {
		blog.Errorf("update association type failed, kind id: %d, err: %v, rid: %s", asstTypeID, err, kit.Rid)
		return nil, kit.CCError.New(common.CCErrTopoCreateAssocKindFailed, err.Error())
	}
	resp = &metadata.UpdateAssociationTypeResult{BaseResp: rsp.BaseResp}
	return resp, nil
}

func (assoc *association) DeleteType(kit *rest.Kit, asstTypeID int64) (resp *metadata.DeleteAssociationTypeResult, err error) {
	cond := condition.CreateCondition()
	cond.Field("id").Eq(asstTypeID)
	query := &metadata.SearchAssociationTypeRequest{
		Condition: cond.ToMapStr(),
	}

	result, err := assoc.SearchType(kit, query)
	if err != nil {
		blog.Errorf("delete association kind[%d], but get detailed info failed, err: %v, rid: %s", asstTypeID, err, kit.Rid)
		return nil, kit.CCError.Error(common.CCErrCommHTTPDoRequestFailed)
	}

	if !result.Result {
		blog.Errorf("delete association kind[%d], but get detailed info failed, err: %s, rid: %s", asstTypeID, result.ErrMsg, kit.Rid)
		return nil, kit.CCError.Error(common.CCErrCommHTTPDoRequestFailed)
	}

	if len(result.Data.Info) > 1 {
		blog.Errorf("delete association kind[%d], but get multiple instance, rid: %s", asstTypeID, kit.Rid)
		return nil, kit.CCError.Error(common.CCErrorTopoGetMultipleAssocKindInstWithOneID)
	}

	if len(result.Data.Info) == 0 {
		return &metadata.DeleteAssociationTypeResult{BaseResp: metadata.SuccessBaseResp, Data: common.CCSuccessStr}, nil
	}

	if result.Data.Info[0].IsPre != nil && *result.Data.Info[0].IsPre {
		blog.Errorf("delete association kind[%d], but this is a pre-defined association kind, can not be deleted., rid: %s", asstTypeID, kit.Rid)
		return nil, kit.CCError.Error(common.CCErrorTopoDeletePredefinedAssociationKind)
	}

	// a already used association kind can not be deleted.
	cond = condition.CreateCondition()
	cond.Field(common.AssociationKindIDField).Eq(result.Data.Info[0].AssociationKindID)
	asso, err := assoc.clientSet.CoreService().Association().ReadModelAssociation(kit.Ctx, kit.Header, &metadata.QueryCondition{Condition: cond.ToMapStr()})
	if err != nil {
		blog.Errorf("delete association kind[%d], but get objects that used this asso kind failed, err: %v, rid: %s", asstTypeID, err, kit.Rid)
		return nil, kit.CCError.Error(common.CCErrCommHTTPDoRequestFailed)
	}

	if !result.Result {
		blog.Errorf("delete association kind[%d], but get objects that used this asso kind failed, err: %s, rid: %s", asstTypeID, result.ErrMsg, kit.Rid)
		return nil, kit.CCError.Error(common.CCErrCommHTTPDoRequestFailed)
	}

	if len(asso.Data.Info) != 0 {
		blog.Warnf("delete association kind[%d], but it has already been used, can not be deleted., rid: %s", asstTypeID, kit.Rid)
		return nil, kit.CCError.Error(common.CCErrorTopoAssociationKindHasBeenUsed)
	}

	rsp, err := assoc.clientSet.CoreService().Association().DeleteAssociationType(
		kit.Ctx, kit.Header, &metadata.DeleteOption{
			Condition: condition.CreateCondition().Field(common.BKFieldID).Eq(asstTypeID).ToMapStr(),
		},
	)
	if err != nil {
		blog.Errorf("delete association type failed, kind id: %d, err: %v, rid: %s", asstTypeID, err, kit.Rid)
		return nil, kit.CCError.New(common.CCErrTopoCreateAssocKindFailed, err.Error())
	}

	return &metadata.DeleteAssociationTypeResult{BaseResp: rsp.BaseResp}, nil
}

func (assoc *association) SearchObject(kit *rest.Kit, request *metadata.SearchAssociationObjectRequest) (resp *metadata.SearchAssociationObjectResult, err error) {
	rsp, err := assoc.clientSet.CoreService().Association().ReadModelAssociation(kit.Ctx, kit.Header, &metadata.QueryCondition{Condition: request.Condition})
	if err != nil {
		return nil, err
	}

	resp = &metadata.SearchAssociationObjectResult{BaseResp: rsp.BaseResp, Data: []*metadata.Association{}}
	for index := range rsp.Data.Info {
		resp.Data = append(resp.Data, &rsp.Data.Info[index])
	}

	return resp, nil
}

func (assoc *association) CreateObject(kit *rest.Kit, request *metadata.Association) (resp *metadata.CreateAssociationObjectResult, err error) {
	rsp, err := assoc.clientSet.CoreService().Association().CreateModelAssociation(kit.Ctx, kit.Header, &metadata.CreateModelAssociation{Spec: *request})
	if err != nil {
		return nil, err
	}

	resp = &metadata.CreateAssociationObjectResult{
		BaseResp: rsp.BaseResp,
	}
	resp.Data.ID = int64(rsp.Data.Created.ID)
	return resp, nil
}

func (assoc *association) UpdateObject(kit *rest.Kit, asstID int, request *metadata.UpdateAssociationObjectRequest) (resp *metadata.UpdateAssociationObjectResult, err error) {
	input := metadata.UpdateOption{
		Condition: condition.CreateCondition().Field(common.BKFieldID).Eq(asstID).ToMapStr(),
		Data:      mapstr.NewFromStruct(request, "json"),
	}

	rsp, err := assoc.clientSet.CoreService().Association().UpdateModelAssociation(kit.Ctx, kit.Header, &input)
	if err != nil {
		return nil, err
	}

	resp = &metadata.UpdateAssociationObjectResult{
		BaseResp: rsp.BaseResp,
	}
	return resp, nil
}

func (assoc *association) DeleteObject(kit *rest.Kit, asstID int) (resp *metadata.DeleteAssociationObjectResult, err error) {
	input := metadata.DeleteOption{
		Condition: condition.CreateCondition().Field(common.BKFieldID).Eq(asstID).ToMapStr(),
	}

	rsp, err := assoc.clientSet.CoreService().Association().DeleteModelAssociation(kit.Ctx, kit.Header, &input)
	if err != nil {
		return nil, err
	}

	return &metadata.DeleteAssociationObjectResult{BaseResp: rsp.BaseResp}, nil

}

func (assoc *association) SearchInst(kit *rest.Kit, request *metadata.SearchAssociationInstRequest) (resp *metadata.SearchAssociationInstResult, err error) {
	rsp, err := assoc.clientSet.CoreService().Association().ReadInstAssociation(kit.Ctx, kit.Header, &metadata.QueryCondition{Condition: request.Condition})
	if err != nil {
		return nil, err
	}

	resp = &metadata.SearchAssociationInstResult{BaseResp: rsp.BaseResp, Data: []*metadata.InstAsst{}}
	for index := range rsp.Data.Info {
		resp.Data = append(resp.Data, &rsp.Data.Info[index])
	}

	return resp, nil
}

func (assoc *association) SearchAssociationRelatedInst(kit *rest.Kit, request *metadata.SearchAssociationRelatedInstRequest) (resp *metadata.SearchAssociationInstResult, err error) {
	cond := &metadata.QueryCondition{
		Fields: request.Fields,
		Page:   request.Page,
	}
	cond.Condition = mapstr.MapStr{
		condition.BKDBOR: []mapstr.MapStr{
			{
				common.BKObjIDField:  request.Condition.ObjectID,
				common.BKInstIDField: request.Condition.InstID,
			},
			{
				common.BKAsstObjIDField:  request.Condition.ObjectID,
				common.BKAsstInstIDField: request.Condition.InstID,
			},
		},
	}

	rsp, err := assoc.clientSet.CoreService().Association().ReadInstAssociation(kit.Ctx, kit.Header, cond)

	resp = &metadata.SearchAssociationInstResult{BaseResp: rsp.BaseResp, Data: []*metadata.InstAsst{}}
	for index := range rsp.Data.Info {
		resp.Data = append(resp.Data, &rsp.Data.Info[index])
	}

	return resp, err
}

func (assoc *association) CreateInst(kit *rest.Kit, request *metadata.CreateAssociationInstRequest) (resp *metadata.CreateAssociationInstResult, err error) {
	cond := condition.CreateCondition()
	cond.Field(common.AssociationObjAsstIDField).Eq(request.ObjectAsstID)
	result, err := assoc.SearchObject(kit, &metadata.SearchAssociationObjectRequest{Condition: cond.ToMapStr()})
	if err != nil {
		blog.Errorf("create association instance, but search object association with cond[%v] failed, err: %v, rid: %s", cond, err, kit.Rid)
		return nil, kit.CCError.Error(common.CCErrCommHTTPDoRequestFailed)
	}

	if !result.Result {
		blog.Errorf("create association instance, but search object association with cond[%v] failed, err: %s, rid: %s", cond, result.ErrMsg, kit.Rid)
		return nil, kit.CCError.New(result.Code, result.ErrMsg)
	}

	if len(result.Data) == 0 {
		blog.Errorf("create instance association, but can not find object association[%s]. rid: %s", request.ObjectAsstID, kit.Rid)
		return nil, kit.CCError.Error(common.CCErrorTopoObjectAssociationNotExist)
	}

	objectAsst := result.Data[0]

	objID := objectAsst.ObjectID
	asstObjID := objectAsst.AsstObjID

	switch result.Data[0].Mapping {
	case metadata.OneToOneMapping:
		// search instances belongs to this association.
		cond := condition.CreateCondition()
		cond.Field(common.AssociationObjAsstIDField).Eq(request.ObjectAsstID)
		cond.Field(common.BKInstIDField).Eq(request.InstID)
		instance, err := assoc.SearchInst(kit, &metadata.SearchAssociationInstRequest{Condition: cond.ToMapStr()})
		if err != nil {
			blog.Errorf("create association instance, but check instance with cond[%v] failed, err: %v, rid: %s", cond, err, kit.Rid)
			return nil, kit.CCError.Error(common.CCErrCommHTTPDoRequestFailed)
		}

		if !instance.Result {
			blog.Errorf("create association instance, but check instance with cond[%v] failed, err: %s, rid: %s", cond, instance.ErrMsg, kit.Rid)
			return nil, kit.CCError.New(instance.Code, instance.ErrMsg)
		}
		if len(instance.Data) >= 1 {
			return nil, kit.CCError.Error(common.CCErrorTopoCreateMultipleInstancesForOneToOneAssociation)
		}

		cond = condition.CreateCondition()
		cond.Field(common.AssociationObjAsstIDField).Eq(request.ObjectAsstID)
		cond.Field(common.BKAsstInstIDField).Eq(request.AsstInstID)

		instance, err = assoc.SearchInst(kit, &metadata.SearchAssociationInstRequest{Condition: cond.ToMapStr()})
		if err != nil {
			blog.Errorf("create association instance, but check instance with cond[%v] failed, err: %v, rid: %s", cond, err, kit.Rid)
			return nil, kit.CCError.Error(common.CCErrCommHTTPDoRequestFailed)
		}

		if !instance.Result {
			blog.Errorf("create association instance, but check instance with cond[%v] failed, err: %s, rid: %s", cond, instance.ErrMsg, kit.Rid)
			return nil, kit.CCError.New(instance.Code, instance.ErrMsg)
		}
		if len(instance.Data) >= 1 {
			return nil, kit.CCError.Error(common.CCErrorTopoCreateMultipleInstancesForOneToOneAssociation)
		}
	case metadata.OneToManyMapping:
		cond = condition.CreateCondition()
		cond.Field(common.AssociationObjAsstIDField).Eq(request.ObjectAsstID)
		cond.Field(common.BKAsstInstIDField).Eq(request.AsstInstID)

		instance, err := assoc.SearchInst(kit, &metadata.SearchAssociationInstRequest{Condition: cond.ToMapStr()})
		if err != nil {
			blog.Errorf("create association instance, but check instance with cond[%v] failed, err: %v, rid: %s", cond, err, kit.Rid)
			return nil, kit.CCError.Error(common.CCErrCommHTTPDoRequestFailed)
		}

		if !instance.Result {
			blog.Errorf("create association instance, but check instance with cond[%v] failed, err: %s, rid: %s", cond, instance.ErrMsg, kit.Rid)
			return nil, kit.CCError.New(instance.Code, instance.ErrMsg)
		}
		if len(instance.Data) >= 1 {
			return nil, kit.CCError.Error(common.CCErrorTopoCreateMultipleInstancesForOneToManyAssociation)
		}

	default:
		// after all the check, new association instance can be created.
	}

	input := metadata.CreateOneInstanceAssociation{
		Data: metadata.InstAsst{
			ObjectAsstID:      request.ObjectAsstID,
			InstID:            request.InstID,
			AsstInstID:        request.AsstInstID,
			ObjectID:          objID,
			AsstObjectID:      asstObjID,
			AssociationKindID: objectAsst.AsstKindID,
		},
	}
	createResult, err := assoc.clientSet.CoreService().Association().CreateInstAssociation(kit.Ctx, kit.Header, &input)
	if err != nil {
		blog.Errorf("create instance association failed, do coreservice create failed, err: %+v, rid: %s", err, kit.Rid)
		return nil, err
	}
	if err := createResult.CCError(); err != nil {
		blog.Errorf("create instance association failed, do coreservice create failed, err: %+v, rid: %s", err, kit.Rid)
		return nil, err
	}

	resp = &metadata.CreateAssociationInstResult{BaseResp: createResult.BaseResp}
	instanceAssociationID := int64(createResult.Data.Created.ID)
	resp.Data.ID = instanceAssociationID

	curData := mapstr.NewFromStruct(input.Data, "json")
	curData.Set("name", objectAsst.AssociationAliasName)

	// generate audit log.
	audit := auditlog.NewInstanceAssociationAudit(assoc.clientSet.CoreService())
	generateAuditParameter := auditlog.NewGenerateAuditCommonParameter(kit, metadata.AuditCreate)
	auditLog, err := audit.GenerateAuditLog(generateAuditParameter, instanceAssociationID, nil)
	if err != nil {
		blog.Errorf(" delete inst asst, generate audit log failed, err: %v, rid: %s", err, kit.Rid)
		return nil, err
	}

	// save audit log.
	err = audit.SaveAuditLog(kit, *auditLog)
	if err != nil {
		blog.Errorf("delete inst asst, save audit log failed, err: %v, rid: %s", err, kit.Rid)
		return nil, kit.CCError.Error(common.CCErrAuditSaveLogFailed)
	}

	return resp, err
}

func (assoc *association) DeleteInst(kit *rest.Kit, asstIDList []int64, bkObjID string) (resp *metadata.DeleteAssociationInstResult, err error) {

	cond := condition.CreateCondition().Field(common.BKFieldID).In(asstIDList)
	// The bkObjID should not be empty has been checked before, the logic here is to adapt to other scene.
	if bkObjID != "" {
		cond.Field(common.BKObjIDField).Eq(bkObjID)
	}

	// asstIDList check duplicate
	idMap := make(map[int64]struct{})
	for _, id := range asstIDList {
		if id <= 0 {
			blog.ErrorJSON("Delete instance association failed, input id list contains illegal id %d, kit: %+v, rid: %s", id, kit, kit.Rid)
			return nil, fmt.Errorf("input id list contains illegal id %d", id)
		}
		if _, exists := idMap[id]; exists {
			blog.ErrorJSON("Delete instance association failed, input id list contains duplicate id %d, kit: %+v, rid: %s", kit, kit.Rid)
			return nil, fmt.Errorf("input id list contains duplicate id %d", id)
		}
		idMap[id] = struct{}{}
	}

	searchCondition := metadata.QueryCondition{
		Condition:      cond.ToMapStr(),
		DisableCounter: true,
	}
	data, err := assoc.clientSet.CoreService().Association().ReadInstAssociation(kit.Ctx, kit.Header, &searchCondition)
	if err != nil {
		blog.Errorf("Delete instance association failed, get instance association failed, kit: %+v, err: %+v, rid: %s", kit, err, kit.Rid)
		return nil, err
	}
	if len(data.Data.Info) != len(asstIDList) {
		blog.Errorf("Delete instance association failed, got unexpected number of instance associations %d which should be %d, searchCondition: %+v, err: %+v, rid: %s", len(data.Data.Info), len(asstIDList), searchCondition, err, kit.Rid)
		return nil, kit.CCError.Error(common.CCErrCommNotFound)
	}

	instanceAssociation := data.Data.Info[0]

	searchCond := condition.CreateCondition()
	searchCond.Field(common.AssociationObjAsstIDField).Eq(instanceAssociation.ObjectAsstID)
	assInfoResult, err := assoc.SearchObject(kit, &metadata.SearchAssociationObjectRequest{Condition: searchCond.ToMapStr()})
	if err != nil {
		blog.Errorf("Delete instance association failed, search object association with cond[%v] failed, err: %v, rid: %s", cond, err, kit.Rid)
		return nil, kit.CCError.Error(common.CCErrCommHTTPDoRequestFailed)
	}
	if !assInfoResult.Result {
		blog.Errorf("Delete instance association failed, search object association with cond[%v] failed, err: %s, rid: %s", cond, assInfoResult.ErrMsg, kit.Rid)
		return nil, assInfoResult.CCError()
	}

	input := metadata.DeleteOption{
		Condition: cond.ToMapStr(),
	}
	rsp, err := assoc.clientSet.CoreService().Association().DeleteInstAssociation(kit.Ctx, kit.Header, &input)
	if err != nil {
		blog.ErrorJSON("Delete instance association failed, err: %s, input: %s, rid: %s", err, input, kit.Rid)
		return nil, kit.CCError.CCError(common.CCErrCommHTTPDoRequestFailed)
	}
	resp = &metadata.DeleteAssociationInstResult{
		BaseResp: rsp.BaseResp,
	}

	// generate audit log.
	audit := auditlog.NewInstanceAssociationAudit(assoc.clientSet.CoreService())
	generateAuditParameter := auditlog.NewGenerateAuditCommonParameter(kit, metadata.AuditDelete)

	auditList := []metadata.AuditLog{}
	for i, asstID := range asstIDList {
		auditLog, err := audit.GenerateAuditLog(generateAuditParameter, asstID, &data.Data.Info[i])
		if err != nil {
			blog.Errorf("Delete instance association failed, generate audit log failed, err: %v, rid: %s", err, kit.Rid)
			return nil, err
		}
		auditList = append(auditList, *auditLog)
	}
	// save audit log.
	err = audit.SaveAuditLog(kit, auditList...)
	if err != nil {
		blog.Errorf("Delete instance association failed, save audit log failed, err: %v, rid: %s", err, kit.Rid)
		return nil, kit.CCError.Error(common.CCErrAuditSaveLogFailed)
	}

	return resp, nil
}

// SearchInstAssociationList 与实例有关系的实例关系数据,以分页的方式返回
func (assoc *association) SearchInstAssociationList(kit *rest.Kit, query *metadata.QueryCondition) ([]metadata.InstAsst, uint64, error) {

	rsp, err := assoc.clientSet.CoreService().Association().ReadInstAssociation(kit.Ctx, kit.Header, query)
	if nil != err {
		blog.Errorf("ReadInstAssociation http do error, err: %s, rid: %s", err.Error(), kit.Rid)
		return nil, 0, kit.CCError.New(common.CCErrCommHTTPDoRequestFailed, err.Error())
	}

	if !rsp.Result {
		blog.ErrorJSON("ReadInstAssociation http response error, query: %s, response: %s, rid: %s", query, rsp, kit.Rid)
		return nil, 0, kit.CCError.New(rsp.Code, rsp.ErrMsg)
	}

	return rsp.Data.Info, rsp.Data.Count, nil
}

// SearchInstAssociationUIList 与实例有关系的实例关系数据,以分页的方式返回
func (assoc *association) SearchInstAssociationUIList(kit *rest.Kit, objID string, query *metadata.QueryCondition) (result interface{}, asstCnt uint64, err error) {

	rsp, err := assoc.clientSet.CoreService().Association().ReadInstAssociation(kit.Ctx, kit.Header, query)
	if nil != err {
		blog.Errorf("ReadInstAssociation http do error, err: %s, rid: %s", err.Error(), kit.Rid)
		return nil, 0, kit.CCError.New(common.CCErrCommHTTPDoRequestFailed, err.Error())
	}

	if !rsp.Result {
		blog.ErrorJSON("ReadInstAssociation http response error, query: %s, response: %s, rid: %s", query, rsp, kit.Rid)
		return nil, 0, kit.CCError.New(rsp.Code, rsp.ErrMsg)
	}

	objIDInstIDMap := make(map[string][]int64, 0)
	var objSrcAsstArr []metadata.InstAsst
	var objDstAsstArr []metadata.InstAsst
	for _, instAsst := range rsp.Data.Info {
		objIDInstIDMap[instAsst.ObjectID] = append(objIDInstIDMap[instAsst.ObjectID], instAsst.InstID)
		objIDInstIDMap[instAsst.AsstObjectID] = append(objIDInstIDMap[instAsst.AsstObjectID], instAsst.AsstInstID)
		if instAsst.ObjectID == objID {
			objSrcAsstArr = append(objSrcAsstArr, instAsst)
		} else {
			objDstAsstArr = append(objDstAsstArr, instAsst)

		}
	}

	instInfo := make(map[string][]mapstr.MapStr, 0)
	for instObjID, instIDArr := range objIDInstIDMap {
		idField := metadata.GetInstIDFieldByObjID(instObjID)
		cond := condition.CreateCondition()
		cond.Field(idField).In(instIDArr)
		input := &metadata.QueryCondition{
			Condition: cond.ToMapStr(),
			Page: metadata.BasePage{
				Start: 0,
				Limit: common.BKNoLimit,
			},
			Fields: []string{metadata.GetInstNameFieldName(instObjID), idField},
		}
		instResp, err := assoc.clientSet.CoreService().Instance().
			ReadInstance(kit.Ctx, kit.Header, instObjID, input)
		if err != nil {
			blog.Errorf("ReadInstance http do error, err: %s, input:%s, rid: %s", err.Error(), input, kit.Rid)
			return nil, 0, kit.CCError.New(common.CCErrCommHTTPDoRequestFailed, err.Error())
		}
		if !instResp.Result {
			blog.ErrorJSON("ReadInstance http response error, query: %s, response: %s, rid: %s", query, rsp, kit.Rid)
			return nil, 0, kit.CCError.New(rsp.Code, rsp.ErrMsg)
		}
		instInfo[instObjID] = instResp.Data.Info

	}
	instAsstMap := map[string][]metadata.InstAsst{
		"src": objSrcAsstArr,
		"dst": objDstAsstArr,
	}

	result = mapstr.MapStr{
		"association": instAsstMap,
		"instance":    instInfo,
	}

	return result, rsp.Data.Count, nil
}

// SearchInstAssociationUIList 与实例有关系的实例关系数据,以分页的方式返回
// returnInstInfoObjID 根据条件查询出来关联关系，需要返回实例信息（实例名，实例ID）的模型ID
func (assoc *association) SearchInstAssociationSingleObjectInstInfo(kit *rest.Kit, returnInstInfoObjID string, query *metadata.QueryCondition) (result []metadata.InstBaseInfo, cnt uint64, err error) {

	rsp, err := assoc.clientSet.CoreService().Association().ReadInstAssociation(kit.Ctx, kit.Header, query)
	if nil != err {
		blog.Errorf("ReadInstAssociation http do error, err: %s, rid: %s", err.Error(), kit.Rid)
		return nil, 0, kit.CCError.New(common.CCErrCommHTTPDoRequestFailed, err.Error())
	}

	if !rsp.Result {
		blog.ErrorJSON("ReadInstAssociation http response error, query: %s, response: %s, rid: %s", query, rsp, kit.Rid)
		return nil, 0, kit.CCError.New(rsp.Code, rsp.ErrMsg)
	}
	// association count
	cnt = rsp.Data.Count

	if cnt == 0 {
		return nil, 0, nil
	}

	var objIDInstIDArr []int64

	for _, instAsst := range rsp.Data.Info {
		if instAsst.ObjectID == returnInstInfoObjID {
			objIDInstIDArr = append(objIDInstIDArr, instAsst.InstID)
		} else if instAsst.AsstObjectID == returnInstInfoObjID {
			objIDInstIDArr = append(objIDInstIDArr, instAsst.AsstInstID)

		}
	}

	idField := metadata.GetInstIDFieldByObjID(returnInstInfoObjID)
	nameField := metadata.GetInstNameFieldName(returnInstInfoObjID)
	cond := condition.CreateCondition()
	cond.Field(idField).In(objIDInstIDArr)
	input := &metadata.QueryCondition{
		Condition: cond.ToMapStr(),
		Page: metadata.BasePage{
			Start: 0,
			Limit: common.BKNoLimit,
		},
		Fields: []string{nameField, idField},
	}
	instResp, err := assoc.clientSet.CoreService().Instance().
		ReadInstance(kit.Ctx, kit.Header, returnInstInfoObjID, input)
	if err != nil {
		blog.Errorf("ReadInstance http do error, err: %s, input:%s, rid: %s", err.Error(), input, kit.Rid)
		return nil, 0, kit.CCError.New(common.CCErrCommHTTPDoRequestFailed, err.Error())
	}
	if !instResp.Result {
		blog.ErrorJSON("ReadInstance http response error, query: %s, response: %s, rid: %s", query, rsp, kit.Rid)
		return nil, 0, kit.CCError.New(rsp.Code, rsp.ErrMsg)
	}

	result = make([]metadata.InstBaseInfo, 0)
	for _, row := range instResp.Data.Info {
		id, err := row.Int64(idField)
		if err != nil {
			blog.ErrorJSON("ReadInstance  convert field(%s) to int error. err:%s, inst:%s,rid:%s", idField, err.Error(), row, kit.Rid)
			// CCErrCommInstFieldConvertFail  convert %s  field %s to %s error %s
			return nil, 0, kit.CCError.Errorf(common.CCErrCommInstFieldConvertFail, returnInstInfoObjID, idField, "int", err.Error())
		}
		name, err := row.String(nameField)
		if err != nil {
			blog.ErrorJSON("ReadInstance  convert field(%s) to int error. err:%s, inst:%s,rid:%s", nameField, err.Error(), row, kit.Rid)
			// CCErrCommInstFieldConvertFail  convert %s  field %s to %s error %s
			return nil, 0, kit.CCError.Errorf(common.CCErrCommInstFieldConvertFail, returnInstInfoObjID, idField, "string", err.Error())
		}
		result = append(result, metadata.InstBaseInfo{
			ID:   id,
			Name: name,
		})
	}

	return result, cnt, nil
}
