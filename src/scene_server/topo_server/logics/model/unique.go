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
	"configcenter/src/ac/extensions"
	"configcenter/src/apimachinery"
	"configcenter/src/common"
	"configcenter/src/common/blog"
	"configcenter/src/common/condition"
	"configcenter/src/common/http/rest"
	"configcenter/src/common/metadata"
)

// UniqueOperationInterface group operation methods
type UniqueOperationInterface interface {
	CreateUnique(kit *rest.Kit, objectID string,
		request *metadata.CreateUniqueRequest) (uniqueID *metadata.RspID, err error)
	UpdateUnique(kit *rest.Kit, objectID string, id uint64, request *metadata.UpdateUniqueRequest) (err error)
	DeleteUnique(kit *rest.Kit, objectID string, id uint64) (err error)
	SearchUnique(kit *rest.Kit, objectID string) (objectUniques []metadata.ObjectUnique, err error)
}

// NewUniqueOperation create a new group operation instance
func NewUniqueOperation(client apimachinery.ClientSetInterface,
	authManager *extensions.AuthManager) UniqueOperationInterface {

	return &unique{
		clientSet:   client,
		authManager: authManager,
	}
}

type unique struct {
	clientSet   apimachinery.ClientSetInterface
	authManager *extensions.AuthManager
}

func (a *unique) CreateUnique(kit *rest.Kit, objectID string,
	request *metadata.CreateUniqueRequest) (uniqueID *metadata.RspID, err error) {

	unique := metadata.ObjectUnique{
		ObjID: request.ObjID,
		Keys:  request.Keys,
	}

	resp, err := a.clientSet.CoreService().Model().CreateModelAttrUnique(kit.Ctx, kit.Header,
		objectID, metadata.CreateModelAttrUnique{Data: unique})
	if err != nil {
		blog.Errorf("create for %s, %#v failed %v, rid: %s", objectID, request, err, kit.Rid)
		return nil, kit.CCError.Error(common.CCErrTopoObjectUniqueCreateFailed)
	}
	if !resp.Result {
		return nil, kit.CCError.New(resp.Code, resp.ErrMsg)
	}

	return &metadata.RspID{ID: int64(resp.Data.Created.ID)}, nil
}

func (a *unique) UpdateUnique(kit *rest.Kit, objectID string, id uint64,
	request *metadata.UpdateUniqueRequest) (err error) {

	update := metadata.UpdateModelAttrUnique{
		Data: *request,
	}
	resp, err := a.clientSet.CoreService().Model().UpdateModelAttrUnique(kit.Ctx, kit.Header, objectID, id, update)
	if err != nil {
		blog.Errorf("update for %s, %d, %#v failed %v, rid: %s", objectID, id, request, err, kit.Rid)
		return kit.CCError.Error(common.CCErrTopoObjectUniqueUpdateFailed)
	}
	if !resp.Result {
		return kit.CCError.New(resp.Code, resp.ErrMsg)
	}

	return nil
}

func (a *unique) DeleteUnique(kit *rest.Kit, objectID string, id uint64) (err error) {
	resp, err := a.clientSet.CoreService().Model().DeleteModelAttrUnique(kit.Ctx, kit.Header, objectID, id)
	if err != nil {
		blog.Errorf("delete for %s, %d failed %v, rid: %s", objectID, id, err, kit.Rid)
		return kit.CCError.Error(common.CCErrTopoObjectUniqueDeleteFailed)
	}
	if !resp.Result {
		return kit.CCError.New(resp.Code, resp.ErrMsg)
	}
	return nil
}

func (a *unique) SearchUnique(kit *rest.Kit, objectID string) (objectUniques []metadata.ObjectUnique, err error) {
	fCond := condition.CreateCondition().Field(common.BKObjIDField).Eq(objectID).ToMapStr()

	cond := metadata.QueryCondition{
		Condition: fCond,
	}
	resp, err := a.clientSet.CoreService().Model().ReadModelAttrUnique(kit.Ctx, kit.Header, cond)
	if err != nil {
		blog.Errorf("[UniqueOperation] search for %s, failed %v, rid: %s", objectID, err, kit.Rid)
		return nil, kit.CCError.Error(common.CCErrTopoObjectUniqueSearchFailed)
	}
	if !resp.Result {
		return nil, kit.CCError.New(resp.Code, resp.ErrMsg)
	}
	return resp.Data.Info, nil
}
