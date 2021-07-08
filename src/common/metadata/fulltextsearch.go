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

package metadata

import (
	"fmt"

	ccjson "configcenter/src/common/json"
)

// elastic index names(maybe a alias name to the real index).
const (
	// IndexNamePrefix prefix of index name.
	IndexNamePrefix = "bk_cmdb."

	// IndexNameBiz name of model business application instance es index.
	IndexNameBiz = IndexNamePrefix + "biz"

	// IndexNameSet name of model set instance es index.
	IndexNameSet = IndexNamePrefix + "set"

	// IndexNameModule name of model module instance es index.
	IndexNameModule = IndexNamePrefix + "module"

	// IndexNameHost name of model host instance es index.
	IndexNameHost = IndexNamePrefix + "host"

	// IndexNameModel name of model es index.
	IndexNameModel = IndexNamePrefix + "model"

	// IndexNameObjectInstance name of common object instance es index.
	IndexNameObjectInstance = IndexNamePrefix + "object_instance"
)

// elastic document data kind.
const (
	// DataKindModel data kind model.
	DataKindModel = "model"

	// DataKindInstance data kind instance.
	DataKindInstance = "instance"
)

// elastic index property types.
const (
	// IndexPropertyTypeKeyword es index property type keyword.
	IndexPropertyTypeKeyword = "keyword"

	// IndexPropertyTypeText es index property type text.
	IndexPropertyTypeText = "text"
)

// elastic index properties.
const (
	// IndexPropertyID es index property for metadata id.
	IndexPropertyID = "meta_id"

	// IndexPropertyDataKind es index property for metadata kind.
	IndexPropertyDataKind = "meta_data_kind"

	// IndexPropertyBKObjID es index property for object id.
	IndexPropertyBKObjID = "meta_bk_obj_id"

	// IndexPropertyBKSupplierAccount es index property for supplier account.
	IndexPropertyBKSupplierAccount = "meta_bk_supplier_account"

	// IndexPropertyBKBizID es index property for business id.
	IndexPropertyBKBizID = "meta_bk_biz_id"

	// IndexPropertyBKParentID es index property for model parent id.
	IndexPropertyBKParentID = "meta_bk_parent_id"

	// IndexPropertyBKCloudID es index property for host cloud id.
	IndexPropertyBKCloudID = "meta_bk_cloud_id"

	// IndexPropertyKeywords es index property for metadata keywords.
	IndexPropertyKeywords = "keywords"
)

// ESIndexMetaSettings elasticsearch index settings.
type ESIndexMetaSettings struct {
	// Shards number of index shards as string type.
	Shards string `json:"number_of_shards"`

	// Replicas number of index document replicas as string type.
	Replicas string `json:"number_of_replicas"`
}

// ESIndexMetaMappings elasticsearch index mappings.
type ESIndexMetaMappings struct {
	// Properties elastic index properties.
	Properties map[string]ESIndexMetaMappingsProperty `json:"properties"`
}

// ESIndexMetaMappingsProperty elasticsearch index mappings property.
type ESIndexMetaMappingsProperty struct {
	// PropertyType elastic index property type. Support 'keyword' 'text'.
	PropertyType string `json:"type"`
}

// ESIndexMetadata is elasticsearch index settings.
type ESIndexMetadata struct {
	// Settings elastic index settings.
	Settings ESIndexMetaSettings `json:"settings"`

	// Mappings elastic index mappings.
	Mappings ESIndexMetaMappings `json:"mappings"`
}

// ESIndex elasticsearch index.
type ESIndex struct {
	// name index name.
	name string

	// version is the plugin index version, as a postfix in target index.
	// the plugin would check and create the version index if it not exist,
	// and alias to bk-cmdb default index name.
	// NOTE: CHANGE the version name if you have modify the indexes metadata struct.
	version string

	// metadata index metadata including settings and mappings.
	metadata *ESIndexMetadata
}

// NewESIndex creates a new elasticsearch index.
func NewESIndex(name, version string, metadata *ESIndexMetadata) *ESIndex {
	return &ESIndex{name: name, version: version, metadata: metadata}
}

// Name returns the real elastic index name.
func (idx *ESIndex) Name() string {
	// return bk-cmdb.{index}_{version} as the real index name.
	return fmt.Sprintf("%s_%s", idx.name, idx.version)
}

// AliasName returns real bk-cmdb index name as index alias name.
func (idx *ESIndex) AliasName() string {
	return idx.name
}

// Metadata returns index metadata.
func (idx *ESIndex) Metadata() string {
	meta, err := ccjson.MarshalToString(idx.metadata)
	if err != nil {
		return ""
	}
	return meta
}
