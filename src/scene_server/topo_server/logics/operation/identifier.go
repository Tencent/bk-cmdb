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
	"configcenter/src/apimachinery"
	"configcenter/src/common"
	"configcenter/src/common/blog"
	"configcenter/src/common/condition"
	"configcenter/src/common/http/rest"
	"configcenter/src/common/mapstr"
	"configcenter/src/common/metadata"
)

type IdentifierOperationInterface interface {
	SearchIdentifier(kit *rest.Kit, objType string,
		param *metadata.SearchIdentifierParam) (*metadata.SearchHostIdentifierData, error)
}

func NewIdentifier(client apimachinery.ClientSetInterface) IdentifierOperationInterface {
	return &identifier{clientSet: client}
}

type identifier struct {
	clientSet apimachinery.ClientSetInterface
}

func (g *identifier) SearchIdentifier(kit *rest.Kit, objType string,
	param *metadata.SearchIdentifierParam) (*metadata.SearchHostIdentifierData, error) {

	if len(param.IP.Data) == 0 {
		blog.Errorf("host ip can't be empty, rid: %s", kit.Rid)
		return nil, kit.CCError.CCErrorf(common.CCErrCommParamsNeedSet, "ip.data")
	}

	cond := condition.CreateCondition()

	or := []mapstr.MapStr{
		{
			common.BKHostInnerIPField: map[string]interface{}{
				common.BKDBIN: param.IP.Data,
			},
		}, {
			common.BKHostOuterIPField: map[string]interface{}{
				common.BKDBIN: param.IP.Data,
			},
		},
	}

	cond.NewOR().MapStrArr(or)
	if param.IP.CloudID != nil {
		cond.Field(common.BKCloudIDField).In(param.IP.CloudID)
	}

	if param.Page.Limit > common.BKMaxPageSize {
		return nil, kit.CCError.CCError(common.CCErrCommOverLimit)
	}
	if param.Page.Limit == 0 {
		param.Page.Limit = common.BKMaxPageSize
	}

	hostQuery := &metadata.QueryCondition{
		Condition: cond.ToMapStr(),
		Fields:    []string{common.BKHostIDField},
		Page:      param.Page,
	}
	hostRet, err := g.clientSet.CoreService().Instance().ReadInstance(kit.Ctx, kit.Header,
		common.BKInnerObjIDHost, hostQuery)
	if nil != err {
		blog.Errorf("query host failed. error: %v, input: %+v,  rid:%s", err, hostQuery, kit.Rid)
		return nil, kit.CCError.CCErrorf(common.CCErrCommHTTPDoRequestFailed)
	}
	if !hostRet.Result {
		blog.Errorf("query host failed, output:%s, condition:%s, rid:%s", hostRet, hostQuery, kit.Rid)
		return nil, kit.CCError.New(hostRet.Code, hostRet.ErrMsg)
	}

	if len(hostRet.Data.Info) == 0 {
		return &metadata.SearchHostIdentifierData{Count: 0, Info: []metadata.HostIdentifier{}}, nil
	}

	var hostIDs []int64
	for _, hostInfo := range hostRet.Data.Info {
		hostID, err := hostInfo.Int64(common.BKHostIDField)
		if err != nil {
			blog.Errorf("bk_host_id not int . error: %v, host info:%s,  rid:%s", err, hostInfo, kit.Rid)
			return nil, kit.CCError.Errorf(common.CCErrCommInstFieldConvertFail, common.BKInnerObjIDHost,
				common.BKHostIDField, "int64", err.Error())
		}
		hostIDs = append(hostIDs, hostID)
	}

	if len(hostIDs) == 0 {
		return new(metadata.SearchHostIdentifierData), nil
	}

	queryHostIdentifier := &metadata.SearchHostIdentifierParam{HostIDs: hostIDs}
	rsp, err := g.clientSet.CoreService().Host().FindIdentifier(kit.Ctx, kit.Header, queryHostIdentifier)
	if nil != err {
		blog.Errorf("search identifier failed. err: %v, ids: %v,  rid:%s", err, hostIDs, kit.Rid)
		return nil, kit.CCError.CCErrorf(common.CCErrCommHTTPDoRequestFailed)
	}

	if !rsp.Result {
		blog.Errorf("search identifier failed, reply: %s, ids: %s, condition: %s, rid: %s", rsp, hostIDs,
			queryHostIdentifier, kit.Rid)
		return nil, kit.CCError.New(rsp.Code, rsp.ErrMsg)
	}

	return &rsp.Data, nil
}
