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

package y3_9_202107161611

import (
	"context"

	"configcenter/src/common"
	"configcenter/src/common/metadata"
	"configcenter/src/scene_server/admin_server/upgrader"
	"configcenter/src/storage/dal"
)

// 更新绑定进程port提示信息
func updateProcBindInfo(ctx context.Context, db dal.RDB, conf *upgrader.Config) error {
	bindIPAttrFilter := map[string]interface{}{
		common.BKObjIDField:      common.BKInnerObjIDProc,
		common.BKPropertyIDField: "bind_info",
		"option.bk_property_id":  "port",
	}

	nowTime := metadata.Now()
	doc := map[string]interface{}{
		common.LastTimeField:   &nowTime,
		"option.$.placeholder": "Single port: 8080</br>Range port: 8080-9090",
	}
	return db.Table(common.BKTableNameObjAttDes).Update(ctx, bindIPAttrFilter, doc)
}
