/*
 * Tencent is pleased to support the open source community by making 蓝鲸 available.,
 * Copyright (C) 2017-2018 THL A29 Limited, a Tencent company. All rights reserved.
 * Licensed under the MIT License (the ",License",); you may not use this file except
 * in compliance with the License. You may obtain a copy of the License at
 * http://opensource.org/licenses/MIT
 * Unless required by applicable law or agreed to in writing, software distributed under
 * the License is distributed on an ",AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
 * either express or implied. See the License for the specific language governing permissions and
 * limitations under the License.
 */

package model

import (
	"fmt"
	"time"

	"configcenter/src/common"
	"configcenter/src/common/blog"
	"configcenter/src/common/errors"
	"configcenter/src/common/http/rest"
	"configcenter/src/common/language"
	"configcenter/src/common/lock"
	"configcenter/src/common/mapstr"
	"configcenter/src/common/metadata"
	"configcenter/src/common/universalsql/mongo"
	"configcenter/src/common/util"
	"configcenter/src/storage/driver/mongodb"
	"configcenter/src/storage/driver/redis"
)

type modelAttribute struct {
	model    *modelManager
	language language.CCLanguageIf
}

func (m *modelAttribute) CreateModelAttributes(kit *rest.Kit, objID string, inputParam metadata.CreateModelAttributes) (dataResult *metadata.CreateManyDataResult, err error) {

	dataResult = &metadata.CreateManyDataResult{
		CreateManyInfoResult: metadata.CreateManyInfoResult{
			Created:    []metadata.CreatedDataResult{},
			Repeated:   []metadata.RepeatedDataResult{},
			Exceptions: []metadata.ExceptionResult{},
		},
	}

	if err := m.model.isValid(kit, objID); nil != err {
		blog.Errorf("CreateModelAttributes failed, validate model(%s) failed, err: %s, rid: %s", objID, err.Error(), kit.Rid)
		return dataResult, err
	}

	addExceptionFunc := func(idx int64, err errors.CCErrorCoder, attr *metadata.Attribute) {
		dataResult.CreateManyInfoResult.Exceptions = append(dataResult.CreateManyInfoResult.Exceptions, metadata.ExceptionResult{
			OriginIndex: idx,
			Message:     err.Error(),
			Code:        int64(err.GetCode()),
			Data:        attr,
		})
	}

	for attrIdx, attr := range inputParam.Attributes {
		func() {
			// fmt.Sprintf("coreservice:create:model:%s:attr:%s", objID, attr.PropertyID)
			redisKey := lock.GetLockKey(lock.CreateModuleAttrFormat, objID, attr.PropertyID)

			locker := lock.NewLocker(redis.Client())
			looked, err := locker.Lock(redisKey, time.Second*35)
			defer locker.Unlock()
			if err != nil {
				blog.ErrorJSON("create model error. get create look error. err:%s, input:%s, rid:%s", err.Error(), inputParam, kit.Rid)
				addExceptionFunc(int64(attrIdx), kit.CCError.CCErrorf(common.CCErrCommRedisOPErr), &attr)
				return
			}
			if !looked {
				blog.ErrorJSON("create model have same task in progress. input:%s, rid:%s", inputParam, kit.Rid)
				addExceptionFunc(int64(attrIdx), kit.CCError.CCErrorf(common.CCErrCommOPInProgressErr, fmt.Sprintf("create object(%s) attribute(%s)", attr.ObjectID, attr.PropertyName)), &attr)
				return
			}
			if attr.IsPre {
				if attr.PropertyID == common.BKInstNameField {
					language := util.GetLanguage(kit.Header)
					lang := m.language.CreateDefaultCCLanguageIf(language)
					attr.PropertyName = util.FirstNotEmptyString(lang.Language("common_property_"+attr.PropertyID), attr.PropertyName, attr.PropertyID)
				}
			}

			attr.OwnerID = kit.SupplierAccount
			_, exists, err := m.isExists(kit, attr.ObjectID, attr.PropertyID, attr.BizID)
			blog.V(5).Infof("CreateModelAttributes isExists info. property id:%s, bizID:%#v, exit:%v, rid:%s", attr.PropertyID, attr.BizID, exists, kit.Rid)
			if nil != err {
				blog.Errorf("CreateModelAttributes failed, attribute field propertyID(%s) exists, err: %s, rid: %s", attr.PropertyID, err.Error(), kit.Rid)
				addExceptionFunc(int64(attrIdx), err.(errors.CCErrorCoder), &attr)
				return
			}

			if exists {
				dataResult.CreateManyInfoResult.Repeated = append(dataResult.CreateManyInfoResult.Repeated, metadata.RepeatedDataResult{
					OriginIndex: int64(attrIdx),
					Data:        mapstr.NewFromStruct(attr, "field"),
				})
				return
			}
			id, err := m.save(kit, attr)
			if nil != err {
				blog.Errorf("CreateModelAttributes failed, failed to save the attribute(%#v), err: %s, rid: %s", attr, err.Error(), kit.Rid)
				addExceptionFunc(int64(attrIdx), err.(errors.CCErrorCoder), &attr)
				return
			}

			dataResult.CreateManyInfoResult.Created = append(dataResult.CreateManyInfoResult.Created, metadata.CreatedDataResult{
				OriginIndex: int64(attrIdx),
				ID:          id,
			})
		} ()
	}

	return dataResult, nil
}

func (m *modelAttribute) SetModelAttributes(kit *rest.Kit, objID string, inputParam metadata.SetModelAttributes) (dataResult *metadata.SetDataResult, err error) {

	dataResult = &metadata.SetDataResult{
		Created:    []metadata.CreatedDataResult{},
		Updated:    []metadata.UpdatedDataResult{},
		Exceptions: []metadata.ExceptionResult{},
	}

	if err := m.model.isValid(kit, objID); nil != err {
		return dataResult, err
	}

	addExceptionFunc := func(idx int64, err errors.CCErrorCoder, attr *metadata.Attribute) {
		dataResult.Exceptions = append(dataResult.Exceptions, metadata.ExceptionResult{
			OriginIndex: idx,
			Message:     err.Error(),
			Code:        int64(err.GetCode()),
			Data:        attr,
		})
	}

	for attrIdx, attr := range inputParam.Attributes {

		existsAttr, exists, err := m.isExists(kit, attr.ObjectID, attr.PropertyID, attr.BizID)
		if nil != err {
			addExceptionFunc(int64(attrIdx), err.(errors.CCErrorCoder), &attr)
			continue
		}
		attr.OwnerID = kit.SupplierAccount
		if exists {
			cond := mongo.NewCondition()
			cond.Element(&mongo.Eq{Key: metadata.AttributeFieldSupplierAccount, Val: kit.SupplierAccount})
			cond.Element(&mongo.Eq{Key: metadata.AttributeFieldID, Val: existsAttr.ID})

			_, err := m.update(kit, mapstr.NewFromStruct(attr, "field"), cond)
			if nil != err {
				blog.Errorf("SetModelAttributes failed, failed to update the attribute(%#v) by the condition(%#v), err: %s, rid: %s", attr, cond.ToMapStr(), err.Error(), kit.Rid)
				addExceptionFunc(int64(attrIdx), err.(errors.CCErrorCoder), &attr)
				continue
			}
			dataResult.Updated = append(dataResult.Updated, metadata.UpdatedDataResult{
				OriginIndex: int64(attrIdx),
				ID:          uint64(existsAttr.ID),
			})
			continue
		}
		id, err := m.save(kit, attr)
		if nil != err {
			blog.Errorf("SetModelAttributes failed, failed to save the attribute(%#v), err: %s, rid: %s", attr, err.Error(), kit.Rid)
			addExceptionFunc(int64(attrIdx), err.(errors.CCErrorCoder), &attr)
			continue
		}

		dataResult.Created = append(dataResult.Created, metadata.CreatedDataResult{
			OriginIndex: int64(attrIdx),
			ID:          id,
		})

	}

	return dataResult, nil
}
func (m *modelAttribute) UpdateModelAttributes(kit *rest.Kit, objID string, inputParam metadata.UpdateOption) (*metadata.UpdatedCount, error) {

	if err := m.model.isValid(kit, objID); nil != err {
		blog.Errorf("UpdateModelAttributes failed, validate model(%s) failed, err: %s, rid: %s", objID, err.Error(), kit.Rid)
		return &metadata.UpdatedCount{}, err
	}

	cond, err := mongo.NewConditionFromMapStr(util.SetModOwner(inputParam.Condition.ToMapInterface(), kit.SupplierAccount))
	if nil != err {
		blog.Errorf("UpdateModelAttributes failed, failed to convert mapstr(%#v) into a condition object, err: %s, rid: %s", inputParam.Condition, err.Error(), kit.Rid)
		return &metadata.UpdatedCount{}, err
	}

	cnt, err := m.update(kit, inputParam.Data, cond)
	if nil != err {
		blog.ErrorJSON("UpdateModelAttributes failed, update attributes failed, model:%s, attributes:%s, condition: %s, err: %s, rid: %s", inputParam.Data, objID, cond, err.Error(), kit.Rid)
		return &metadata.UpdatedCount{}, err
	}

	return &metadata.UpdatedCount{Count: cnt}, nil
}

func (m *modelAttribute) UpdateModelAttributesIndex(kit *rest.Kit, objID string, inputParam metadata.UpdateOption) (result *metadata.UpdateAttrIndexData, err error) {

	// attributes exist check
	cond := inputParam.Condition
	cond = util.SetQueryOwner(cond, kit.SupplierAccount)
	exists, err := mongodb.Client().Table(common.BKTableNameObjAttDes).Find(cond).Count(kit.Ctx)
	if nil != err {
		blog.Errorf("UpdateModelAttributesIndex failed, request(%s): database operation is failed, condition: %v, err: %s", kit.Rid, inputParam.Condition, err.Error())
		return result, kit.CCError.Error(common.CCErrCommDBSelectFailed)
	}
	if exists <= 0 {
		blog.Errorf("UpdateModelAttributesIndex failed, attributes not exist, condition: %v", inputParam.Condition)
		return result, fmt.Errorf("UpdateModelAttributesIndex failed, attributes not exist, condition: %v", inputParam.Condition)
	}

	propertyGroupStr, err := inputParam.Data.String(common.BKPropertyGroupField)
	if err != nil {
		blog.ErrorJSON("UpdateModelAttributesIndex failed, request(%s): mapstr convert string failed, condition: %v, err: %s", kit.Rid, inputParam.Condition, err.Error())
		return result, err
	}
	// check if bk_property_index has been used, if not, use it directly
	condition := mapstr.MapStr{}
	condition = util.SetQueryOwner(condition, kit.SupplierAccount)
	condition[common.BKObjIDField] = objID
	condition[common.BKPropertyGroupField] = inputParam.Data[common.BKPropertyGroupField]
	condition[common.BKPropertyIndexField] = inputParam.Data[common.BKPropertyIndexField]
	count, err := mongodb.Client().Table(common.BKTableNameObjAttDes).Find(condition).Count(kit.Ctx)
	if nil != err {
		blog.Errorf("UpdateModelAttributesIndex failed, request(%s): database operation is failed, error info is %s", kit.Rid, err.Error())
		return result, kit.CCError.Error(common.CCErrCommDBSelectFailed)
	}
	if count <= 0 {
		data := mapstr.MapStr{
			common.BKPropertyIndexField: inputParam.Data[common.BKPropertyIndexField],
			common.BKPropertyGroupField: inputParam.Data[common.BKPropertyGroupField],
		}
		err = mongodb.Client().Table(common.BKTableNameObjAttDes).Update(kit.Ctx, cond, data)
		if nil != err {
			blog.Errorf("UpdateModelAttributesIndex failed, request(%s): database operation is failed, error info is %s", kit.Rid, err.Error())
			return result, kit.CCError.Error(common.CCErrCommDBSelectFailed)
		}

		result, err := m.buildUpdateAttrIndexReturn(kit, objID, propertyGroupStr)
		if err != nil {
			blog.Errorf("UpdateModelAttributesIndex, update index success, but build return data failed, rid: %s, err: %s", kit.Rid, err.Error())
			return result, err
		}

		return result, nil
	}

	// get all properties which bk_property_index is larger than the current bk_property_index , exclude self
	condition[common.BKPropertyIndexField] = mapstr.MapStr{"$gte": inputParam.Data[common.BKPropertyIndexField]}
	condition["id"] = mapstr.MapStr{"$ne": inputParam.Condition["id"]}
	resultAttrs := []metadata.Attribute{}
	err = mongodb.Client().Table(common.BKTableNameObjAttDes).Find(condition).All(kit.Ctx, &resultAttrs)
	if nil != err {
		blog.Errorf("UpdateModelAttributesIndex failed, request(%s): database operation is failed, error info is %s", kit.Rid, err.Error())
		return result, kit.CCError.Error(common.CCErrCommDBSelectFailed)
	}

	for _, attr := range resultAttrs {
		opt := mapstr.MapStr{}
		opt["id"] = attr.ID
		data := mapstr.MapStr{common.BKPropertyIndexField: attr.PropertyIndex + 1}
		err = mongodb.Client().Table(common.BKTableNameObjAttDes).Update(kit.Ctx, opt, data)
		if nil != err {
			blog.Errorf("UpdateModelAttributesIndex failed, request(%s): database operation is failed, error info is %s", kit.Rid, err.Error())
			return result, kit.CCError.Error(common.CCErrCommDBSelectFailed)
		}
	}

	// update bk_property_index now
	data := mapstr.MapStr{
		common.BKPropertyIndexField: inputParam.Data[common.BKPropertyIndexField],
		common.BKPropertyGroupField: inputParam.Data[common.BKPropertyGroupField],
	}
	err = mongodb.Client().Table(common.BKTableNameObjAttDes).Update(kit.Ctx, cond, data)
	if nil != err {
		blog.Errorf("UpdateModelAttributesIndex failed, request(%s): database operation is failed, error info is %s", kit.Rid, err.Error())
		return result, kit.CCError.Error(common.CCErrCommDBSelectFailed)
	}

	result, err = m.buildUpdateAttrIndexReturn(kit, objID, propertyGroupStr)
	if err != nil {
		blog.Errorf("UpdateModelAttributesIndex, update index success, but build return data failed, rid: %s, err: %s", kit.Rid, err.Error())
		return result, err
	}

	return result, nil
}

func (m *modelAttribute) UpdateModelAttributesByCondition(kit *rest.Kit, inputParam metadata.UpdateOption) (*metadata.UpdatedCount, error) {

	cond, err := mongo.NewConditionFromMapStr(util.SetModOwner(inputParam.Condition.ToMapInterface(), kit.SupplierAccount))
	if nil != err {
		blog.Errorf("UpdateModelAttributesByCondition failed, failed to convert mapstr(%#v) into a condition object, err: %s, rid: %s", inputParam.Condition, err.Error(), kit.Rid)
		return &metadata.UpdatedCount{}, err
	}

	cnt, err := m.update(kit, inputParam.Data, cond)
	if nil != err {
		blog.Errorf("UpdateModelAttributesByCondition failed, failed to update fields (%#v) by condition(%#v), err: %s, rid: %s", inputParam.Data, cond.ToMapStr(), err.Error(), kit.Rid)
		return &metadata.UpdatedCount{}, err
	}

	return &metadata.UpdatedCount{Count: cnt}, nil
}

func (m *modelAttribute) DeleteModelAttributes(kit *rest.Kit, objID string, inputParam metadata.DeleteOption) (*metadata.DeletedCount, error) {

	if err := m.model.isValid(kit, objID); nil != err {
		blog.Errorf("request(%s): it is failed to check if the model(%s) is valid, error info is %s", kit.Rid, objID, err.Error())
		return &metadata.DeletedCount{}, err
	}

	cond, err := mongo.NewConditionFromMapStr(util.SetModOwner(inputParam.Condition.ToMapInterface(), kit.SupplierAccount))
	if nil != err {
		blog.Errorf("request(%s): it is failed to convert from mapstr(%#v) into a condition object, error info is %s", kit.Rid, inputParam.Condition, err.Error())
		return &metadata.DeletedCount{}, err
	}

	cond.Element(&mongo.Eq{Key: metadata.AttributeFieldSupplierAccount, Val: kit.SupplierAccount})
	cnt, err := m.delete(kit, cond)
	return &metadata.DeletedCount{Count: cnt}, err
}

func (m *modelAttribute) SearchModelAttributes(kit *rest.Kit, objID string, inputParam metadata.QueryCondition) (*metadata.QueryModelAttributeDataResult, error) {

	if err := m.model.isValid(kit, objID); nil != err {
		blog.Errorf("request(%s): it is failed to check if the model(%s) is valid, error info is %s", kit.Rid, objID, err.Error())
		return nil, err
	}

	inputParam.Condition[common.BKObjIDField] = objID
	inputParam.Condition = util.SetQueryOwner(inputParam.Condition, kit.SupplierAccount)

	attrResult, err := m.newSearch(kit, inputParam.Condition)
	if nil != err {
		blog.Errorf("request(%s): it is failed to search the attributes of the model(%s), error info is %s", kit.Rid, objID, err.Error())
		return nil, err
	}

	return &metadata.QueryModelAttributeDataResult{
		Count: int64(len(attrResult)),
		Info:  attrResult,
	}, nil
}

func (m *modelAttribute) SearchModelAttributesByCondition(kit *rest.Kit, inputParam metadata.QueryCondition) (*metadata.QueryModelAttributeDataResult, error) {

	dataResult := &metadata.QueryModelAttributeDataResult{
		Info: []metadata.Attribute{},
	}

	inputParam.Condition = util.SetQueryOwner(inputParam.Condition, kit.SupplierAccount)

	attrResult, err := m.searchWithSort(kit, inputParam)
	if nil != err {
		blog.Errorf("request(%s): it is failed to search the attributes of the model(%+v), error info is %s", kit.Rid, inputParam, err.Error())
		return &metadata.QueryModelAttributeDataResult{}, err
	}

	dataResult.Count = int64(len(attrResult))
	dataResult.Info = attrResult
	return dataResult, nil
}
