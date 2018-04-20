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

package instdata

import (
	"configcenter/src/common"
	"configcenter/src/source_controller/common/commondata"
	"errors"
)

//GetCntByCondition get count by condition
func GetCntByCondition(objType string, condition interface{}) (int, error) {
	tName := commondata.ObjTableMap[objType]
	cnt, err := DataH.GetCntByCondition(tName, condition)
	if nil != err {
		return 0, err
	}
	return cnt, nil
}

//DelObjByCondition delete object by condition
func DelObjByCondition(objType string, condition interface{}) error {
	tName := commondata.ObjTableMap[objType]
	err := DataH.DelByCondition(tName, condition)
	if nil != err {
		return err
	}
	return nil
}

//UpdateObjByCondition update object by condition
func UpdateObjByCondition(objType string, data interface{}, condition interface{}) error {
	tName := commondata.ObjTableMap[objType]
	err := DataH.UpdateByCondition(tName, data, condition)
	if nil != err {
		return err
	}
	return nil
}

//GetObjectByCondition get object by condition
func GetObjectByCondition(objType string, fields []string, condition, result interface{}, sort string, skip, limit int) error {
	tName := commondata.ObjTableMap[objType]
	return DataH.GetMutilByCondition(tName, fields, condition, result, sort, skip, limit)
}

//CreateObject add new object
func CreateObject(objType string, input interface{}, idName *string) (int, error) {
	tName := commondata.ObjTableMap[objType]
	objID, err := DataH.GetIncID(tName)
	if err != nil {
		return 0, err
	}
	inputc := input.(map[string]interface{})
	*idName = GetIDNameByType(objType)
	inputc[*idName] = objID
	DataH.Insert(tName, inputc)
	return int(objID), nil
}

//GetIDNameByType get id name by type
func GetIDNameByType(objType string) string {
	switch objType {
	case common.BKInnerObjIDApp:
		return common.BKAppIDField
	case common.BKInnerObjIDSet:
		return common.BKSetIDField
	case common.BKInnerObjIDModule:
		return common.BKModuleIDField
	case common.BKINnerObjIDObject:
		return common.BKInstIDField
	case common.BKInnerObjIDHost:
		return common.BKHostIDField
	case common.BKInnerObjIDProc:
		return common.BKProcIDField
	case common.BKInnerObjIDPlat:
		return common.BKCloudIDField
	case common.BKTableNameInstAsst:
		return common.BKFieldID
	default:
		return common.BKInstIDField
	}
}

//GetObjectByID get object by id
func GetObjectByID(objType string, fields []string, id int, result interface{}, sort string) error {
	tName := commondata.ObjTableMap[objType]
	condition := make(map[string]interface{}, 1)
	switch objType {
	case common.BKInnerObjIDApp:
		condition[common.BKAppIDField] = id
	case common.BKInnerObjIDSet:
		condition[common.BKSetIDField] = id
	case common.BKInnerObjIDModule:
		condition[common.BKModuleIDField] = id
	case common.BKINnerObjIDObject:
		condition[common.BKInstIDField] = id
	case common.BKInnerObjIDHost:
		condition[common.BKHostIDField] = id
	case common.BKInnerObjIDProc:
		condition[common.BKProcIDField] = id
	case common.BKInnerObjIDPlat:
		condition[common.BKCloudIDField] = id
	default:
		return errors.New("invalid object type")
	}
	err := DataH.GetOneByCondition(tName, fields, condition, result)
	return err
}
