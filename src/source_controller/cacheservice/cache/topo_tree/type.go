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

package topo_tree

// search one of the node's path in the the business topology.
type SearchNodePathOption struct {
	Business int64           `json:"bk_biz_id"`
	Nodes    []*MainlineNode `json:"bk_nodes"`
}

type MainlineNode struct {
	// mainline topology object
	Object string `json:"bk_obj_id"`
	// object's instance id, eg, bk_set_id, bk_module_id...
	InstanceID int64 `json:"bk_inst_id"`
}

// NodePaths describes a topology instance's path from itself to the top biz.
// cause a host may exists in multiple modules, so it's may have several paths.
type NodePaths struct {
	*MainlineNode
	InstanceName string `json:"bk_inst_name"`
	// host may have multiple paths
	Paths [][]Node `json:"bk_paths"`
}

type Node struct {
	// mainline topology object
	Object string `json:"bk_obj_id"`
	// object's instance id, eg, bk_set_id, bk_module_id...
	InstanceID int64 `json:"bk_inst_id"`
	// instance's name, eg: bk_set_name, bk_module_name...
	InstanceName string `json:"bk_inst_name"`
	// node's parent id, only used for internal, do not return to user.
	ParentID int64 `json:"-"`
}

type module struct {
	ID       int64  `json:"bk_module_id"`
	Name     string `json:"bk_module_name"`
	Biz      int64  `json:"bk_biz_id"`
	ParentID int64  `json:"bk_parent_id"`
}

type set struct {
	ID       int64  `json:"bk_set_id"`
	Name     string `json:"bk_set_name"`
	Biz      int64  `json:"bk_biz_id"`
	ParentID int64  `json:"bk_parent_id"`
}

type biz struct {
	ID   int64  `json:"bk_biz_id"`
	Name string `json:"bk_biz_name"`
}

type custom struct {
	ID       int64  `json:"bk_inst_id"`
	ParentID int64  `json:"bk_parent_id"`
	Name     string `json:"bk_inst_name"`
}
