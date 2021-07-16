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
	"strconv"

	"configcenter/src/common"
	"configcenter/src/common/blog"
	"configcenter/src/common/http/rest"
	"configcenter/src/common/metadata"
	"configcenter/src/common/util"
)

var ForbiddenModifyMainlineObjectUniqueWhiteList = []string{
	common.BKInnerObjIDHost,
}

// CreateObjectUnique create a new object unique
func (s *Service) CreateObjectUnique(ctx *rest.Contexts) {
	request := &metadata.CreateUniqueRequest{}
	if err := ctx.DecodeInto(request); err != nil {
		ctx.RespAutoError(err)
		return
	}

	objectID := ctx.Request.PathParameter(common.BKObjIDField)

	// mainline object's unique can not be changed.
	yes, err := s.Core.AssociationOperation().IsMainlineObject(ctx.Kit, objectID)
	if err != nil {
		ctx.RespAutoError(err)
		return
	}
	if yes {
		if util.InStrArr(ForbiddenModifyMainlineObjectUniqueWhiteList, objectID) == false {
			ctx.RespAutoError(ctx.Kit.CCError.Error(common.CCErrorTopoMainlineObjectCanNotBeChanged))
			return
		}
	}

	var id *metadata.RspID
	txnErr := s.Engine.CoreAPI.CoreService().Txn().AutoRunTxn(ctx.Kit.Ctx, ctx.Kit.Header, func() error {
		var err error
		id, err = s.Logics.UniqueOperation().CreateUnique(ctx.Kit, objectID, request)
		if err != nil {
			blog.Errorf("[CreateObjectUnique] create for [%s] failed: %v, raw: %#v, rid: %s", objectID, err, request, ctx.Kit.Rid)
			return err
		}

		return nil
	})

	if txnErr != nil {
		ctx.RespAutoError(txnErr)
		return
	}
	ctx.RespEntity(id)
}

// UpdateObjectUnique update a object unique
func (s *Service) UpdateObjectUnique(ctx *rest.Contexts) {
	request := &metadata.UpdateUniqueRequest{}

	if err := ctx.DecodeInto(request); err != nil {
		ctx.RespAutoError(err)
		return
	}

	objectID := ctx.Request.PathParameter(common.BKObjIDField)
	id, err := strconv.ParseUint(ctx.Request.PathParameter("id"), 10, 64)
	if err != nil {
		ctx.RespAutoError(ctx.Kit.CCError.Errorf(common.CCErrCommParamsInvalid, "id"))
		return
	}

	// validate unique keys.
	for _, key := range request.Keys {
		if key.ID == 0 {
			ctx.RespAutoError(ctx.Kit.CCError.New(common.CCErrCommParamsInvalid, "unique key_id is 0"))
			return
		}
		if len(key.Kind) == 0 {
			ctx.RespAutoError(ctx.Kit.CCError.New(common.CCErrCommParamsInvalid, "unique key_kind is empty"))
			return
		}
	}

	// mainline object's unique can not be changed.
	yes, err := s.Core.AssociationOperation().IsMainlineObject(ctx.Kit, objectID)
	if err != nil {
		ctx.RespAutoError(err)
		return
	}
	if yes {
		if util.InStrArr(ForbiddenModifyMainlineObjectUniqueWhiteList, objectID) == false {
			ctx.RespAutoError(ctx.Kit.CCError.Error(common.CCErrorTopoMainlineObjectCanNotBeChanged))
			return
		}
	}

	txnErr := s.Engine.CoreAPI.CoreService().Txn().AutoRunTxn(ctx.Kit.Ctx, ctx.Kit.Header, func() error {
		err := s.Logics.UniqueOperation().UpdateUnique(ctx.Kit, objectID, id, request)
		if err != nil {
			blog.Errorf("[UpdateObjectUnique] update for [%s](%d) failed: %v, raw: %#v, rid: %s", objectID, id, err, request, ctx.Kit.Rid)
			return err
		}
		return nil
	})

	if txnErr != nil {
		ctx.RespAutoError(txnErr)
		return
	}
	ctx.RespEntity(nil)
}

// DeleteObjectUnique delete a object unique
func (s *Service) DeleteObjectUnique(ctx *rest.Contexts) {
	objectID := ctx.Request.PathParameter(common.BKObjIDField)
	id, err := strconv.ParseUint(ctx.Request.PathParameter("id"), 10, 64)
	if err != nil {
		ctx.RespAutoError(ctx.Kit.CCError.Errorf(common.CCErrCommParamsInvalid, "id"))
		return
	}

	// mainline object's unique can not be changed.
	yes, err := s.Core.AssociationOperation().IsMainlineObject(ctx.Kit, objectID)
	if err != nil {
		ctx.RespAutoError(err)
		return
	}
	if yes {
		if util.InStrArr(ForbiddenModifyMainlineObjectUniqueWhiteList, objectID) == false {
			ctx.RespAutoError(ctx.Kit.CCError.Error(common.CCErrorTopoMainlineObjectCanNotBeChanged))
			return
		}
	}

	uniques, err := s.Logics.UniqueOperation().SearchUnique(ctx.Kit, objectID)
	if err != nil {
		ctx.RespAutoError(err)
		return
	}

	if len(uniques) <= 1 {
		blog.Errorf("[DeleteObjectUnique][%s] unique should have more than one, rid: %s", objectID, ctx.Kit.Rid)
		ctx.RespAutoError(ctx.Kit.CCError.Error(common.CCErrTopoObjectUniqueShouldHaveMoreThanOne))
		return
	}

	txnErr := s.Engine.CoreAPI.CoreService().Txn().AutoRunTxn(ctx.Kit.Ctx, ctx.Kit.Header, func() error {
		err = s.Logics.UniqueOperation().DeleteUnique(ctx.Kit, objectID, id)
		if err != nil {
			blog.Errorf("[DeleteObjectUnique] delete [%s](%d) failed: %v, rid: %s", objectID, id, err, ctx.Kit.Rid)
			return err
		}

		return nil
	})

	if txnErr != nil {
		ctx.RespAutoError(txnErr)
		return
	}
	ctx.RespEntity(nil)
}

// SearchObjectUnique search object uniques
func (s *Service) SearchObjectUnique(ctx *rest.Contexts) {
	objectID := ctx.Request.PathParameter(common.BKObjIDField)
	uniques, err := s.Logics.UniqueOperation().SearchUnique(ctx.Kit, objectID)
	if err != nil {
		blog.Errorf("[SearchObjectUnique] search for [%s] failed: %v, rid: %s", objectID, err, ctx.Kit.Rid)
		ctx.RespAutoError(err)
		return
	}

	ctx.RespEntity(uniques)
}
