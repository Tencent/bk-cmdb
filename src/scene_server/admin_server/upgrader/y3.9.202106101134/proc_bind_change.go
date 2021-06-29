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

package y3_9_202106101134

import (
	"context"

	"configcenter/src/common"
	"configcenter/src/common/blog"
	"configcenter/src/scene_server/admin_server/upgrader"
	"configcenter/src/storage/dal"
)

func updateProcessBindInfoAttr(ctx context.Context, db dal.RDB, conf *upgrader.Config) error {
	filter := map[string]interface{}{
		common.BKObjIDField:      common.BKInnerObjIDProc,
		common.BKPropertyIDField: common.BKProcBindInfo,
		common.BkSupplierAccount: conf.OwnerID,
	}

	attr := new(bindInfoAttribute)
	if err := db.Table(common.BKTableNameObjAttDes).Find(filter).One(ctx, attr); err != nil {
		blog.ErrorJSON("get process bind info attribute failed, err: %s, filter: %s", err, filter)
		return err
	}

	for index, option := range attr.Option {
		if option.PropertyID == common.BKIP {
			option.Option = `^((2(5[0-5]|[0-4]\d))|[0-1]?\d{1,2})(\.((2(5[0-5]|[0-4]\d))|[0-1]?\d{1,2})){3}$`
			attr.Option[index] = option
			continue
		}

		if option.PropertyID == common.BKPort {
			option.Option = `^(((([1-9][0-9]{0,3})|([1-5][0-9]{4})|(6[0-4][0-9]{3})|(65[0-4][0-9]{2})|(655[0-2][0-9])|(6553[0-5]))-(([1-9][0-9]{0,3})|([1-5][0-9]{4})|(6[0-4][0-9]{3})|(65[0-4][0-9]{2})|(655[0-2][0-9])|(6553[0-5])))|((([1-9][0-9]{0,3})|([1-5][0-9]{4})|(6[0-4][0-9]{3})|(65[0-4][0-9]{2})|(655[0-2][0-9])|(6553[0-5]))))$`
			attr.Option[index] = option
			continue
		}
	}

	doc := map[string]interface{}{
		common.BKOptionField: attr.Option,
	}

	if err := db.Table(common.BKTableNameObjAttDes).Update(ctx, filter, doc); err != nil {
		blog.ErrorJSON("update process bind info attribute failed, err: %s, filter: %s, doc: %s", err, filter, doc)
		return err
	}
	return nil
}

type subAttribute struct {
	PropertyID    string      `field:"bk_property_id" json:"bk_property_id" bson:"bk_property_id"`
	PropertyName  string      `field:"bk_property_name" json:"bk_property_name" bson:"bk_property_name"`
	Placeholder   string      `field:"placeholder" json:"placeholder" bson:"placeholder"`
	IsEditable    bool        `field:"editable" json:"editable" bson:"editable"`
	IsRequired    bool        `field:"isrequired" json:"isrequired" bson:"isrequired"`
	IsReadOnly    bool        `field:"isreadonly" json:"isreadonly" bson:"isreadonly"`
	IsSystem      bool        `field:"bk_issystem" json:"bk_issystem" bson:"bk_issystem"`
	IsAPI         bool        `field:"bk_isapi" json:"bk_isapi" bson:"bk_isapi"`
	PropertyType  string      `field:"bk_property_type" json:"bk_property_type" bson:"bk_property_type"`
	Option        interface{} `field:"option" json:"option" bson:"option"`
	Description   string      `field:"description" json:"description" bson:"description"`
	PropertyGroup string      `field:"bk_property_group" json:"bk_property_group" bson:"bk_property_group"`
}

type bindInfoAttribute struct {
	Option []subAttribute `field:"option" json:"option" bson:"option"`
}
