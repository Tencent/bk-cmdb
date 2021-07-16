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

import (
	"context"
	"errors"
	"fmt"

	"configcenter/src/common"
	"configcenter/src/common/blog"
	ccError "configcenter/src/common/errors"
	"configcenter/src/common/json"
	"configcenter/src/source_controller/cacheservice/cache/business"
)

func NewTopologyTree(client *business.Client) *TopologyTree {
	return &TopologyTree{bizCache: client}
}

type TopologyTree struct {
	bizCache *business.Client
}

func (t *TopologyTree) SearchNodePath(ctx context.Context, opt *SearchNodePathOption,
	supplierAccount string) ([]NodePaths, error) {

	if opt.Business <= 0 {
		return nil, ccError.New(common.CCErrCommParamsIsInvalid, fmt.Sprintf("invalid bk_biz_id: %d", opt.Business))
	}

	topo, err := t.bizCache.GetTopology()
	if err != nil {
		return nil, err
	}

	// reverse the topology sequence, so that we can construct the path conveniently.
	reverseTopo := reverse(topo)
	topoMap := make(map[string]struct{})
	for _, to := range topo {
		topoMap[to] = struct{}{}
	}

	// filter object type and instances for batch query
	objects := make(map[string][]int64)
	for _, node := range opt.Nodes {
		// validate object at first.
		if node.Object == "biz" {
			return nil, ccError.New(common.CCErrCommParamsIsInvalid, "not support biz")
		}

		if len(node.Object) == 0 || node.InstanceID <= 0 {
			// return nil, errors.New("invalid bk_nodes parameter")
			return nil, ccError.New(common.CCErrCommParamsIsInvalid, "object or inst id")
		}

		if _, exist := topoMap[node.Object]; !exist {
			// return nil, fmt.Errorf("invalid object %s", node.Object)
			return nil, ccError.New(common.CCErrCommParamsIsInvalid, "object")
		}

		objects[node.Object] = append(objects[node.Object], node.InstanceID)
	}

	var paths map[int64][]Node
	var nameMap map[int64]string

	objNameMap := make(map[string]map[int64]string)
	all := make(map[string]map[int64][]Node)
	for object, instances := range objects {

		switch object {
		case "host":
			// TODO: support this later
			return nil, ccError.New(common.CCErrCommParamsInvalid, "host")

		case "module":
			nameMap, paths, err = t.genModuleParentPaths(ctx, opt.Business, supplierAccount, reverseTopo, instances)
			if err != nil {
				return nil, err
			}

			all[object] = paths
			objNameMap[object] = nameMap

		case "set":
			nameMap, paths, err = t.genSetParentPaths(ctx, opt.Business, supplierAccount, reverseTopo, instances)
			if err != nil {
				return nil, err
			}

			all[object] = paths
			objNameMap[object] = nameMap

		default:
			nameMap, paths, err = t.genCustomParentPaths(ctx, opt.Business, "set",
				supplierAccount, reverseTopo, instances)
			if err != nil {
				return nil, err
			}

			// trim the head path
			for id, nodes := range paths {
				if len(nodes) < 1 {
					// normally, this can not be happen
					continue
				}
				paths[id] = nodes[1:]
			}

			all[object] = paths
			objNameMap[object] = nameMap

		}
	}

	for obj, paths := range all {
		for id, nodes := range paths {
			paths[id] = reverseNode(nodes)
		}
		all[obj] = paths
	}

	nodePath := make([]NodePaths, 0)
	for _, node := range opt.Nodes {
		// 过滤掉不存在的实例
		if _, exist := objNameMap[node.Object][node.InstanceID]; !exist {
			continue
		}
		nodePath = append(nodePath, NodePaths{
			MainlineNode: node,
			InstanceName: objNameMap[node.Object][node.InstanceID],
			Paths:        [][]Node{all[node.Object][node.InstanceID]},
		})
	}

	return nodePath, nil
}

func (t *TopologyTree) genModuleParentPaths(ctx context.Context, biz int64, supplierAccount string, revTopo []string,
	moduleIDs []int64) (names map[int64]string, paths map[int64][]Node, err error) {

	rid := ctx.Value(common.BKHTTPCCRequestID)

	moduleMap, setList, err := t.searchModules(ctx, biz, moduleIDs)
	if err != nil {
		return nil, nil, err
	}

	setMap, _, err := t.searchSets(ctx, biz, setList)
	if err != nil {
		return nil, nil, err
	}

	// add set path
	paths = make(map[int64][]Node)
	for id, mod := range moduleMap {
		set := setMap[mod.ParentID]
		paths[id] = append(paths[id], Node{
			Object:       "set",
			InstanceID:   set.ID,
			InstanceName: set.Name,
			ParentID:     set.ParentID,
		})
	}

	_, setPaths, err := t.genSetParentPaths(ctx, biz, supplierAccount, revTopo, setList)
	if err != nil {
		blog.Errorf("gen set paths failed, err: %v, rid: %v", err, rid)
		return nil, nil, err
	}

	nameMap := make(map[int64]string, len(moduleMap))
	for id, mod := range moduleMap {
		ns, exist := setPaths[mod.ParentID]
		if !exist {
			blog.ErrorJSON("can not find set %d node from set path: %s", id, setPaths)
			return nil, nil, errors.New("can not find set node from set path")
		}
		paths[id] = append(paths[id], ns...)

		nameMap[id] = mod.Name
	}

	return nameMap, paths, nil
}

func (t *TopologyTree) genSetParentPaths(ctx context.Context, biz int64, supplierAccount string, revTopo []string,
	previousList []int64) (names map[int64]string, paths map[int64][]Node, err error) {

	rid := ctx.Value(common.BKHTTPCCRequestID)

	nextNode, err := nextNode("set", revTopo)
	if err != nil {
		return nil, nil, err
	}

	setMap, customList, err := t.searchSets(ctx, biz, previousList)
	if err != nil {
		return nil, nil, err
	}

	// define at first, so that we can use it later, but return earlier.
	nameMap := make(map[int64]string, len(setMap))

	paths = make(map[int64][]Node)
	// loop until we go to biz node.
	if nextNode == "biz" {
		bizDetail, err := t.bizDetail(ctx, biz)
		if err != nil {
			return nil, nil, err
		}

		// add biz path
		for id, set := range setMap {
			paths[id] = append(paths[id], Node{
				Object:       "biz",
				InstanceID:   bizDetail.ID,
				InstanceName: bizDetail.Name,
				ParentID:     0,
			})

			nameMap[id] = set.Name
		}

		return nameMap, paths, nil
	}

	_, customPaths, err := t.genCustomParentPaths(ctx, biz, "set", supplierAccount, revTopo, customList)
	if err != nil {
		blog.Errorf("gen custom parent %s/%v paths failed, err: %v, rid: %v", nextNode, previousList, err, rid)
		return nil, nil, err
	}

	for id, set := range setMap {

		custom, exist := customPaths[set.ParentID]
		if !exist {
			blog.ErrorJSON("can not find instance %d node from custom path: %s", id, customPaths)
			return nil, nil, errors.New("can not find instance node from custom path")
		}

		paths[id] = append(paths[id], custom...)

		nameMap[id] = set.Name
	}

	return nameMap, paths, nil
}

func (t *TopologyTree) genCustomParentPaths(ctx context.Context, biz int64, prevNode, supplierAccount string,
	revTopo []string, previousList []int64) (names map[int64]string, paths map[int64][]Node, err error) {

	rid := ctx.Value(common.BKHTTPCCRequestID)

	bizDetail, err := t.bizDetail(ctx, biz)
	if err != nil {
		return nil, nil, err
	}

	nameMap := make(map[int64]string, 0)

	paths = make(map[int64][]Node)
	for {

		nextNode, err := nextNode(prevNode, revTopo)
		if err != nil {
			return nil, nil, err
		}

		// loop until we go to biz node.
		if nextNode == "biz" {
			// add biz path
			for _, id := range previousList {
				paths[id] = append(paths[id], Node{
					Object:       "biz",
					InstanceID:   bizDetail.ID,
					InstanceName: bizDetail.Name,
					ParentID:     0,
				})
			}
			return nameMap, paths, nil
		}

		customMap, prevList, err := t.searchCustomInstances(ctx, nextNode, supplierAccount, previousList)
		if err != nil {
			blog.Errorf("search supplier account %s custom instance %s/%v failed, err: %v, rid: %v",
				supplierAccount, nextNode, previousList, err, rid)
			return nil, nil, err
		}

		// first paths, as is the bottom topology
		if len(paths) == 0 {
			for id, cu := range customMap {
				paths[id] = append(paths[id], Node{
					Object:       nextNode,
					InstanceID:   cu.ID,
					InstanceName: cu.Name,
					ParentID:     cu.ParentID,
				})
				// first custom's name id map, it's all we need.
				nameMap[id] = cu.Name
			}

			prevNode = nextNode
			previousList = prevList
			// handle next custom level.
			continue
		}

		// generate and add custom path
		for id, nodes := range paths {
			hit := -1
			var prev Node
			for idx, node := range nodes {
				if node.Object == prevNode {
					hit = idx
					prev = node
					break
				}
			}

			if hit < 0 {
				blog.Errorf("gen custom topo instance path, but got invalid nodes: %v, rid: %v", nodes, rid)
				return nil, nil, errors.New("got invalid custom topo nodes")
			}

			custom, exist := customMap[prev.ParentID]
			if !exist {
				blog.Errorf("gen custom topo instance path, but can not find node: %v parent, rid: %v", prev, rid)
				return nil, nil, fmt.Errorf("can not find node %v parent", nodes)
			}

			paths[id] = append(paths[id], Node{
				Object:       nextNode,
				InstanceID:   custom.ID,
				InstanceName: custom.Name,
				ParentID:     custom.ParentID,
			})
		}

		prevNode = nextNode
		previousList = prevList
	}

}

func (t *TopologyTree) searchModules(ctx context.Context, biz int64, moduleIDs []int64) (
	moduleMap map[int64]*module, setList []int64, err error) {

	rid := ctx.Value(common.BKHTTPCCRequestID)

	ms, err := t.bizCache.ListModuleDetails(ctx, moduleIDs)
	if err != nil {
		blog.Errorf("list module detail from cache failed, err: %v, rid: %v", err, rid)
		return nil, nil, err
	}
	moduleMap = make(map[int64]*module)
	setMap := make(map[int64]struct{})
	for _, m := range ms {
		mod := new(module)
		if err := json.Unmarshal([]byte(m), mod); err != nil {
			blog.Errorf("unmarshal module failed, err: %v, rid: %v", err, rid)
			return nil, nil, err
		}

		// verify
		if mod.Biz != biz {
			return nil, nil, fmt.Errorf("module %d do not belongs to biz: %d", mod.ID, biz)
		}

		moduleMap[mod.ID] = mod

		setMap[mod.ParentID] = struct{}{}
	}

	for id := range setMap {
		setList = append(setList, id)
	}

	return moduleMap, setList, nil
}

func (t *TopologyTree) searchSets(ctx context.Context, biz int64, setIDs []int64) (
	setMap map[int64]*set, parentList []int64, err error) {

	rid := ctx.Value(common.BKHTTPCCRequestID)

	setDetails, err := t.bizCache.ListSetDetails(ctx, setIDs)
	if err != nil {
		blog.Errorf("construct module path, but get set details failed, err: %v, rid: %v", err, rid)
		return nil, nil, err
	}
	setMap = make(map[int64]*set)
	parentMap := make(map[int64]struct{})
	for _, s := range setDetails {
		set := new(set)
		if err := json.Unmarshal([]byte(s), set); err != nil {
			blog.Errorf("unmarshal set failed, err: %v, rid: %s", err, rid)
			return nil, nil, err
		}

		if set.Biz != biz {
			return nil, nil, fmt.Errorf("set %d do not belongs to biz: %d", set.ID, biz)
		}

		setMap[set.ID] = set
		parentMap[set.ParentID] = struct{}{}
	}

	for id := range parentMap {
		parentList = append(parentList, id)
	}

	return setMap, parentList, nil
}

func (t *TopologyTree) searchCustomInstances(ctx context.Context, object, supplierAccount string, instIDs []int64) (
	instMap map[int64]*custom, parentList []int64, err error) {

	rid := ctx.Value(common.BKHTTPCCRequestID)

	instances, err := t.bizCache.ListCustomLevelDetail(ctx, object, supplierAccount, instIDs)
	if err != nil {
		blog.Errorf("list custom level %s instances: %v failed, err: %v", object, instIDs, err)
		return nil, nil, err
	}

	instMap = make(map[int64]*custom)
	for _, inst := range instances {
		c := new(custom)
		if err := json.Unmarshal([]byte(inst), c); err != nil {
			blog.Errorf("unmarshal custom level failed, detail: %s, err: %v, rid: %v", inst, err, rid)
			return nil, nil, err
		}

		instMap[c.ID] = c
		parentList = append(parentList, c.ParentID)
	}

	return instMap, parentList, nil
}

func (t *TopologyTree) bizDetail(ctx context.Context, bizID int64) (
	*biz, error) {

	rid := ctx.Value(common.BKHTTPCCRequestID)

	business, err := t.bizCache.GetBusiness(ctx, bizID)
	if err != nil {
		return nil, fmt.Errorf("get biz: %d detail failed, err: %v, rid: %v", bizID, err, rid)
	}

	detail := new(biz)
	if err := json.Unmarshal([]byte(business), detail); err != nil {
		blog.Errorf("unmarshal business %s failed, err: %v, rid: %v", business, err, rid)
		return nil, err
	}

	return detail, nil
}

func nextNode(from string, topo []string) (string, error) {

	if from == "biz" {
		return "", errors.New("biz do not have next topo node")
	}

	var next string
	for idx := range topo {
		if topo[idx] == from {
			if len(topo) < (idx + 1) {
				return "", errors.New("invalid topology rank")
			}
			next = topo[idx+1]
			break
		}
	}

	if len(next) == 0 {
		return "", fmt.Errorf("invalid mainline topo without %s model", from)
	}

	return next, nil

}

// reverse the slice's element from tail to head.
func reverse(t []string) []string {
	if len(t) == 0 {
		return t
	}
	for i, j := 0, len(t)-1; i < j; i, j = i+1, j-1 {
		t[i], t[j] = t[j], t[i]
	}
	return t
}

// reverse the slice's element from tail to head.
func reverseNode(t []Node) []Node {
	if len(t) == 0 {
		return t
	}
	for i, j := 0, len(t)-1; i < j; i, j = i+1, j-1 {
		t[i], t[j] = t[j], t[i]
	}
	return t
}
