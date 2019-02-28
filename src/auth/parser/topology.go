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

package parser

import (
	"fmt"
	"net/http"
	"regexp"
	"strconv"

	"configcenter/src/auth/meta"
	"configcenter/src/framework/core/errors"
)

func (ps *parseStream) topology() *parseStream {
	if ps.err != nil {
		return ps
	}

	ps.business().
		mainline().
		associationType().
		objectAssociation().
		objectInstanceAssociation().
		objectInstance().
		object().
		ObjectClassification().
		objectAttributeGroup().
		objectAttribute().
		ObjectModule().
		ObjectSet().
		objectUnique()

	return ps
}

var (
	createBusinessRegexp       = regexp.MustCompile(`^/api/v3/biz/[\S][^/]+$`)
	updateBusinessRegexp       = regexp.MustCompile(`^/api/v3/biz/[\S][^/]+/[0-9]+$`)
	deleteBusinessRegexp       = regexp.MustCompile(`^/api/v3/biz/[\S][^/]+/[0-9]+$`)
	findBusinessRegexp         = regexp.MustCompile(`^/api/v3/biz/search/[\S][^/]+$`)
	updateBusinessStatusRegexp = regexp.MustCompile(`^/api/v3/biz/status/[\S][^/]+/[\S][^/]+/[0-9]+$`)
)

func (ps *parseStream) business() *parseStream {
	if ps.err != nil {
		return ps
	}

	// create business, this is not a normalize api.
	// TODO: update this api format.
	if createBusinessRegexp.MatchString(ps.RequestCtx.URI) && ps.RequestCtx.Method == http.MethodPost {
		ps.Attribute.Resources = []meta.Resource{
			meta.Resource{
				Basic: meta.Basic{
					Type: meta.Business,
				},
				Action: meta.Create,
			},
		}
		return ps
	}

	// update business, this is not a normalize api.
	// TODO: update this api format.
	if updateBusinessRegexp.MatchString(ps.RequestCtx.URI) && ps.RequestCtx.Method == http.MethodPut {
		if len(ps.RequestCtx.Elements) != 5 {
			ps.err = errors.New("invalid update business request uri")
			return ps
		}

		bizID, err := strconv.ParseInt(ps.RequestCtx.Elements[4], 10, 64)
		if err != nil {
			ps.err = fmt.Errorf("udpate business, but got invalid business id %s", ps.RequestCtx.Elements[4])
			return ps
		}

		ps.Attribute.Resources = []meta.Resource{
			meta.Resource{
				Basic: meta.Basic{
					Type:       meta.Business,
					InstanceID: bizID,
				},
				Action: meta.Update,
			},
		}
		return ps
	}

	// update business enable status, this is not a normalize api.
	// TODO: update this api format.
	if updateBusinessRegexp.MatchString(ps.RequestCtx.URI) && ps.RequestCtx.Method == http.MethodPut {
		if len(ps.RequestCtx.Elements) != 7 {
			ps.err = errors.New("invalid update business enable status request uri")
			return ps
		}

		bizID, err := strconv.ParseInt(ps.RequestCtx.Elements[6], 10, 64)
		if err != nil {
			ps.err = fmt.Errorf("udpate business enable status, but got invalid business id %s", ps.RequestCtx.Elements[4])
			return ps
		}

		ps.Attribute.Resources = []meta.Resource{
			meta.Resource{
				Basic: meta.Basic{
					Type:       meta.Business,
					InstanceID: bizID,
				},
				Action: meta.Update,
			},
		}
		return ps
	}

	// delete business, this is not a normalize api.
	// TODO: update this api format
	if updateBusinessRegexp.MatchString(ps.RequestCtx.URI) && ps.RequestCtx.Method == http.MethodDelete {
		if len(ps.RequestCtx.Elements) != 5 {
			ps.err = errors.New("invalid delete business request uri")
			return ps
		}

		bizID, err := strconv.ParseInt(ps.RequestCtx.Elements[4], 10, 64)
		if err != nil {
			ps.err = fmt.Errorf("delete business, but got invalid business id %s", ps.RequestCtx.Elements[4])
			return ps
		}

		ps.Attribute.Resources = []meta.Resource{
			meta.Resource{
				Basic: meta.Basic{
					Type:       meta.Business,
					InstanceID: bizID,
				},
				Action: meta.Delete,
			},
		}
		return ps
	}

	// find business, this is not a normalize api.
	// TODO: update this api format
	if findBusinessRegexp.MatchString(ps.RequestCtx.URI) && ps.RequestCtx.Method == http.MethodPost {
		ps.Attribute.Resources = []meta.Resource{
			meta.Resource{
				Basic: meta.Basic{
					Type: meta.Business,
				},
				// we don't know if one or more business is to find, so we assume it's a find many
				// business operation.
				Action: meta.FindMany,
			},
		}
		return ps
	}

	return ps
}

const (
	createMainlineObjectPattern = "/api/v3/topo/model/mainline"
)

var (
	deleteMainlineObjectRegexp        = regexp.MustCompile(`^/api/v3/topo/model/mainline/owners/[\S][^/]+/objectids/[\S][^/]+$`)
	findMainlineObjectTopoRegexp      = regexp.MustCompile(`^/api/v3/topo/model/[\S][^/]+$`)
	findMainlineInstanceTopoRegexp    = regexp.MustCompile(`^/api/v3/topo/inst/[\S][^/]+/[0-9]+$`)
	findMainineSubInstanceTopoRegexp  = regexp.MustCompile(`^/api/v3/topo/inst/child/[\S][^/]+/[\S][^/]+/[0-9]+/[0-9]+$`)
	findMainlineIdleFaultModuleRegexp = regexp.MustCompile(`^/api/v3/topo/internal/[\S][^/]+/[0-9]+$`)
)

func (ps *parseStream) mainline() *parseStream {
	if ps.err != nil {
		return ps
	}

	// create mainline object operation.
	if ps.hitPattern(createMainlineObjectPattern, http.MethodPost) {
		ps.Attribute.Resources = []meta.Resource{
			meta.Resource{
				Basic: meta.Basic{
					Type: meta.MainlineObject,
				},
				Action: meta.Create,
			},
		}
		return ps
	}

	// delete mainline object operation
	if ps.hitRegexp(deleteMainlineObjectRegexp, http.MethodDelete) {
		ps.Attribute.Resources = []meta.Resource{
			meta.Resource{
				Basic: meta.Basic{
					Type: meta.MainlineObject,
				},
				Action: meta.Delete,
			},
		}

		return ps
	}

	// get mainline object operation
	if ps.hitRegexp(findMainlineObjectTopoRegexp, http.MethodGet) {
		ps.Attribute.Resources = []meta.Resource{
			meta.Resource{
				Basic: meta.Basic{
					Type: meta.MainlineObjectTopology,
				},
				Action: meta.Find,
			},
		}

		return ps
	}

	// find mainline instance topology operation.
	if ps.hitRegexp(findMainlineInstanceTopoRegexp, http.MethodGet) {
		if len(ps.RequestCtx.Elements) != 6 {
			ps.err = errors.New("find mainline instance topology, but got invalid url")
			return ps
		}

		bizID, err := strconv.ParseInt(ps.RequestCtx.Elements[5], 10, 64)
		if err != nil {
			ps.err = fmt.Errorf("find mainline instance topology, but got invalid business id %s", ps.RequestCtx.Elements[5])
			return ps
		}
		ps.Attribute.BusinessID = bizID
		ps.Attribute.Resources = []meta.Resource{
			meta.Resource{
				Basic: meta.Basic{
					Type: meta.MainlineInstanceTopology,
				},
				Action: meta.Find,
			},
		}

		return ps
	}

	// find mainline object instance's sub-instance topology operation.
	if ps.hitRegexp(findMainineSubInstanceTopoRegexp, http.MethodGet) {
		if len(ps.RequestCtx.Elements) != 9 {
			ps.err = errors.New("find mainline object's sub instance topology, but got invalid url")
			return ps
		}

		bizID, err := strconv.ParseInt(ps.RequestCtx.Elements[7], 10, 64)
		if err != nil {
			ps.err = fmt.Errorf("find mainline object's sub instance topology, but got invalid business id %s", ps.RequestCtx.Elements[7])
			return ps
		}
		ps.Attribute.BusinessID = bizID
		ps.Attribute.Resources = []meta.Resource{
			meta.Resource{
				Basic: meta.Basic{
					Type: meta.MainlineInstanceTopology,
				},
				Action: meta.Find,
			},
		}

		return ps
	}

	// find mainline internal idle and fault module operation.
	if ps.hitRegexp(findMainlineIdleFaultModuleRegexp, http.MethodGet) {
		if len(ps.RequestCtx.Elements) != 6 {
			ps.err = errors.New("find mainline idle and fault module, but got invalid url")
			return ps
		}

		bizID, err := strconv.ParseInt(ps.RequestCtx.Elements[5], 10, 64)
		if err != nil {
			ps.err = fmt.Errorf("find mainline idle and fault module, but got invalid business id %s", ps.RequestCtx.Elements[5])
			return ps
		}
		ps.Attribute.BusinessID = bizID
		ps.Attribute.Resources = []meta.Resource{
			meta.Resource{
				Basic: meta.Basic{
					Type: meta.MainlineObject,
				},
				Action: meta.Find,
			},
		}

		return ps
	}

	return ps
}

const (
	findManyAssociationKindPattern = "/api/v3/topo/association/type/action/search"
	createAssociationKindPattern   = "/api/v3/topo/association/type/action/search"
)

var (
	updateAssociationKindRegexp = regexp.MustCompile(`^/api/v3/topo/association/type/[0-9]+/action/update$`)
	deleteAssociationKindRegexp = regexp.MustCompile(`^/api/v3/topo/association/type/[0-9]+/action/delete$`)
)

func (ps *parseStream) associationType() *parseStream {
	if ps.err != nil {
		return ps
	}

	// find association kind operation
	if ps.hitPattern(findManyAssociationKindPattern, http.MethodPost) {
		ps.Attribute.Resources = []meta.Resource{
			meta.Resource{
				Basic: meta.Basic{
					Type: meta.AssociationType,
				},
				Action: meta.FindMany,
			},
		}
		return ps
	}

	// create association kind operation
	if ps.hitPattern(createAssociationKindPattern, http.MethodPost) {
		ps.Attribute.Resources = []meta.Resource{
			meta.Resource{
				Basic: meta.Basic{
					Type: meta.AssociationType,
				},
				Action: meta.Create,
			},
		}
		return ps
	}

	// update association kind operation
	if ps.hitRegexp(updateAssociationKindRegexp, http.MethodPut) {
		if len(ps.RequestCtx.Elements) != 8 {
			ps.err = errors.New("update association kind, but got invalid url")
			return ps
		}

		kindID, err := strconv.ParseInt(ps.RequestCtx.Elements[5], 10, 64)
		if err != nil {
			ps.err = fmt.Errorf("update association kind, but got invalid kind id %s", ps.RequestCtx.Elements[5])
			return ps
		}
		ps.Attribute.Resources = []meta.Resource{
			meta.Resource{
				Basic: meta.Basic{
					Type:       meta.AssociationType,
					InstanceID: kindID,
				},
				Action: meta.Update,
			},
		}

		return ps
	}

	// delete association kind operation
	if ps.hitRegexp(deleteAssociationKindRegexp, http.MethodDelete) {
		if len(ps.RequestCtx.Elements) != 8 {
			ps.err = errors.New("delete association kind, but got invalid url")
			return ps
		}

		kindID, err := strconv.ParseInt(ps.RequestCtx.Elements[5], 10, 64)
		if err != nil {
			ps.err = fmt.Errorf("delete association kind, but got invalid kind id %s", ps.RequestCtx.Elements[5])
			return ps
		}
		ps.Attribute.Resources = []meta.Resource{
			meta.Resource{
				Basic: meta.Basic{
					Type:       meta.AssociationType,
					InstanceID: kindID,
				},
				Action: meta.Delete,
			},
		}

		return ps
	}

	return ps
}

const (
	findObjectAssociationPattern                    = "/api/v3/object/association/action/search"
	createObjectAssociationPattern                  = "/api/v3/object/association/action/create"
	findObjectAssociationWithAssociationKindPattern = "/api/v3/topo/association/type/action/search/batch"
)

var (
	updateObjectAssociationRegexp = regexp.MustCompile(`^/api/v3/object/association/[0-9]+/action/update$`)
	deleteObjectAssociationRegexp = regexp.MustCompile(`^/api/v3/object/association/[0-9]+/action/delete$`)
)

func (ps *parseStream) objectAssociation() *parseStream {
	if ps.err != nil {
		return ps
	}

	// search object association operation
	if ps.RequestCtx.URI == findObjectAssociationPattern && ps.RequestCtx.Method == http.MethodPost {
		ps.Attribute.Resources = []meta.Resource{
			meta.Resource{
				Basic: meta.Basic{
					Type: meta.ObjectAssociation,
				},
				Action: meta.FindMany,
			},
		}
		return ps
	}

	// create object association operation
	if ps.RequestCtx.URI == createObjectAssociationPattern && ps.RequestCtx.Method == http.MethodPost {
		ps.Attribute.Resources = []meta.Resource{
			meta.Resource{
				Basic: meta.Basic{
					Type: meta.ObjectAssociation,
				},
				Action: meta.Create,
			},
		}
		return ps
	}

	// update object association operation
	if updateObjectAssociationRegexp.MatchString(ps.RequestCtx.URI) && ps.RequestCtx.Method == http.MethodPut {
		if len(ps.RequestCtx.Elements) != 7 {
			ps.err = errors.New("update object association, but got invalid url")
			return ps
		}

		assoID, err := strconv.ParseInt(ps.RequestCtx.Elements[4], 10, 64)
		if err != nil {
			ps.err = fmt.Errorf("update object association, but got invalid association id %s", ps.RequestCtx.Elements[4])
			return ps
		}

		ps.Attribute.Resources = []meta.Resource{
			meta.Resource{
				Basic: meta.Basic{
					Type:       meta.ObjectAssociation,
					InstanceID: assoID,
				},
				Action: meta.Update,
			},
		}
		return ps
	}

	// delete object association operation
	if deleteObjectAssociationRegexp.MatchString(ps.RequestCtx.URI) && ps.RequestCtx.Method == http.MethodDelete {
		if len(ps.RequestCtx.Elements) != 7 {
			ps.err = errors.New("delete object association, but got invalid url")
			return ps
		}

		assoID, err := strconv.ParseInt(ps.RequestCtx.Elements[4], 10, 64)
		if err != nil {
			ps.err = fmt.Errorf("delete object association, but got invalid association id %s", ps.RequestCtx.Elements[4])
			return ps
		}

		ps.Attribute.Resources = []meta.Resource{
			meta.Resource{
				Basic: meta.Basic{
					Type:       meta.ObjectAssociation,
					InstanceID: assoID,
				},
				Action: meta.Delete,
			},
		}
		return ps
	}

	// find object association with a association kind list.
	if ps.RequestCtx.URI == findObjectAssociationWithAssociationKindPattern && ps.RequestCtx.Method == http.MethodPost {
		ps.Attribute.Resources = []meta.Resource{
			meta.Resource{
				Basic: meta.Basic{
					Type: meta.ObjectAssociation,
				},
				Action: meta.FindMany,
			},
		}
		return ps
	}

	return ps
}

const (
	findObjectInstanceAssociationPattern   = "/api/v3/inst/association/action/search"
	createObjectInstanceAssociationPattern = "/api/v3/inst/association/action/create"
)

var (
	deleteObjectInstanceAssociationRegexp = regexp.MustCompile("/api/v3/inst/association/[0-9]+/action/delete")
)

func (ps *parseStream) objectInstanceAssociation() *parseStream {
	if ps.err != nil {
		return ps
	}

	// find object instance's association operation.
	if ps.RequestCtx.URI == findObjectInstanceAssociationPattern && ps.RequestCtx.Method == http.MethodPost {
		ps.Attribute.Resources = []meta.Resource{
			meta.Resource{
				Basic: meta.Basic{
					Type: meta.ObjectInstanceAssociation,
				},
				Action: meta.FindMany,
			},
		}
		return ps
	}

	// create object's instance association operation.
	if ps.RequestCtx.URI == createObjectInstanceAssociationPattern && ps.RequestCtx.Method == http.MethodPost {
		ps.Attribute.Resources = []meta.Resource{
			meta.Resource{
				Basic: meta.Basic{
					Type: meta.ObjectInstanceAssociation,
				},
				Action: meta.Create,
			},
		}
		return ps
	}

	// delete object's instance association operation.
	if ps.hitRegexp(deleteObjectInstanceAssociationRegexp, http.MethodDelete) {
		if len(ps.RequestCtx.Elements) != 7 {
			ps.err = errors.New("delete object instance association, but got invalid url")
			return ps
		}

		assoID, err := strconv.ParseInt(ps.RequestCtx.Elements[4], 10, 64)
		if err != nil {
			ps.err = fmt.Errorf("delete object instance association, but got invalid association id %s", ps.RequestCtx.Elements[4])
			return ps
		}

		ps.Attribute.Resources = []meta.Resource{
			meta.Resource{
				Basic: meta.Basic{
					Type:       meta.ObjectInstanceAssociation,
					InstanceID: assoID,
				},
				Action: meta.Delete,
			},
		}
		return ps
	}

	return ps
}

var (
	createObjectInstanceRegexp          = regexp.MustCompile(`^/api/v3/inst/[\S][^/]+/[\S][^/]+$`)
	findObjectInstanceRegexp            = regexp.MustCompile(`^/api/v3/inst/association/search/owner/[\S][^/]+/object/[\S][^/]+$`)
	updateObjectInstanceRegexp          = regexp.MustCompile(`^/api/v3/inst/[\S][^/]+/[\S][^/]+/[0-9]+$`)
	updateObjectInstanceBatchRegexp     = regexp.MustCompile(`^/api/v3/inst/[\S][^/]+/[\S][^/]+/batch$`)
	deleteObjectInstanceBatchRegexp     = regexp.MustCompile(`^/api/v3/inst/[\S][^/]+/[\S][^/]+/batch$`)
	deleteObjectInstanceRegexp          = regexp.MustCompile(`^/api/v3/inst/[\S][^/]+/[\S][^/]+/[0-9]+$`)
	findObjectInstanceSubTopologyRegexp = regexp.MustCompile(`^/api/v3/inst/association/topo/search/owner/[\S][^/]+/object/[\S][^/]+/inst/[0-9]+$`)
	findObjectInstanceTopologyRegexp    = regexp.MustCompile(`^/api/v3/inst/association/topo/search/owner/[\S][^/]+/object/[\S][^/]+/inst/[0-9]+$`)
	findBusinessInstanceTopologyRegexp  = regexp.MustCompile(`^/api/v3/topo/inst/[\S][^/]+/[0-9]+$`)
	findObjectInstancesRegexp           = regexp.MustCompile(`^/api/v3/inst/search/owner/[\S][^/]+/object/[\S][^/]+$`)
)

func (ps *parseStream) objectInstance() *parseStream {
	if ps.err != nil {
		return ps
	}

	// create object instance operation.
	if ps.hitRegexp(createObjectInstanceRegexp, http.MethodPost) {
		ps.Attribute.Resources = []meta.Resource{
			meta.Resource{
				Basic: meta.Basic{
					Type: meta.ObjectInstance,
				},
				Action: meta.Create,
			},
		}
		return ps
	}

	// find object instance operation.
	if ps.hitRegexp(findObjectInstanceRegexp, http.MethodPost) {
		if len(ps.RequestCtx.Elements) != 9 {
			ps.err = errors.New("search object instance, but got invalid url")
			return ps
		}
		ps.Attribute.Resources = []meta.Resource{
			meta.Resource{
				Basic: meta.Basic{
					Type: meta.ObjectInstance,
				},
				Action: meta.Find,
				Affiliated: meta.Affiliated{
					Type: meta.Object,
					Name: ps.RequestCtx.Elements[8],
				},
			},
		}
		return ps
	}

	// update object instance operation.
	if ps.hitRegexp(updateObjectInstanceRegexp, http.MethodPut) {
		if len(ps.RequestCtx.Elements) != 6 {
			ps.err = errors.New("update object instance, but got invalid url")
			return ps
		}

		instID, err := strconv.ParseInt(ps.RequestCtx.Elements[5], 10, 64)
		if err != nil {
			ps.err = fmt.Errorf("update object instance, but got invalid instance id %s", ps.RequestCtx.Elements[5])
			return ps
		}

		ps.Attribute.Resources = []meta.Resource{
			meta.Resource{
				Basic: meta.Basic{
					Type:       meta.ObjectInstance,
					InstanceID: instID,
				},
				Action: meta.Update,
				Affiliated: meta.Affiliated{
					Type: meta.Object,
					Name: ps.RequestCtx.Elements[4],
				},
			},
		}
		return ps
	}

	// update object instance batch operation.
	if ps.hitRegexp(updateObjectInstanceBatchRegexp, http.MethodPut) {
		if len(ps.RequestCtx.Elements) != 6 {
			ps.err = errors.New("update object instance batch, but got invalid url")
			return ps
		}

		ps.Attribute.Resources = []meta.Resource{
			meta.Resource{
				Basic: meta.Basic{
					Type: meta.ObjectInstance,
				},
				Action: meta.UpdateMany,
				Affiliated: meta.Affiliated{
					Type: meta.Object,
					Name: ps.RequestCtx.Elements[4],
				},
			},
		}
		return ps
	}

	// delete object instance batch operation.
	if ps.hitRegexp(deleteObjectInstanceBatchRegexp, http.MethodDelete) {
		if len(ps.RequestCtx.Elements) != 6 {
			ps.err = errors.New("delete object instance batch, but got invalid url")
			return ps
		}

		ps.Attribute.Resources = []meta.Resource{
			meta.Resource{
				Basic: meta.Basic{
					Type: meta.ObjectInstance,
				},
				Action: meta.DeleteMany,
				Affiliated: meta.Affiliated{
					Type: meta.Object,
					Name: ps.RequestCtx.Elements[4],
				},
			},
		}
		return ps
	}

	// delete object instance operation.
	if ps.hitRegexp(deleteObjectInstanceRegexp, http.MethodDelete) {
		if len(ps.RequestCtx.Elements) != 6 {
			ps.err = errors.New("delete object instance, but got invalid url")
			return ps
		}

		instID, err := strconv.ParseInt(ps.RequestCtx.Elements[5], 10, 64)
		if err != nil {
			ps.err = fmt.Errorf("delete object instance, but got invalid instance id %s", ps.RequestCtx.Elements[5])
			return ps
		}

		ps.Attribute.Resources = []meta.Resource{
			meta.Resource{
				Basic: meta.Basic{
					Type:       meta.ObjectInstance,
					InstanceID: instID,
				},
				Action: meta.Delete,
				Affiliated: meta.Affiliated{
					Type: meta.Object,
					Name: ps.RequestCtx.Elements[4],
				},
			},
		}
		return ps
	}

	// find object instance topology operation
	if ps.hitRegexp(findObjectInstanceSubTopologyRegexp, http.MethodPost) {
		if len(ps.RequestCtx.Elements) != 12 {
			ps.err = errors.New("find object instance topology, but got invalid url")
			return ps
		}

		instID, err := strconv.ParseInt(ps.RequestCtx.Elements[11], 10, 64)
		if err != nil {
			ps.err = fmt.Errorf("find object instance topology, but got invalid instance id %s", ps.RequestCtx.Elements[11])
			return ps
		}

		ps.Attribute.Resources = []meta.Resource{
			meta.Resource{
				Basic: meta.Basic{
					Type:       meta.ObjectInstanceTopology,
					InstanceID: instID,
				},
				Action: meta.Find,
				Affiliated: meta.Affiliated{
					Type: meta.Object,
					Name: ps.RequestCtx.Elements[9],
				},
			},
		}
		return ps
	}

	// find object instance fully topology operation.
	if ps.hitRegexp(findObjectInstanceTopologyRegexp, http.MethodPost) {
		if len(ps.RequestCtx.Elements) != 12 {
			ps.err = errors.New("find object instance topology, but got invalid url")
			return ps
		}

		instID, err := strconv.ParseInt(ps.RequestCtx.Elements[11], 10, 64)
		if err != nil {
			ps.err = fmt.Errorf("find object instance, but get instance id %s", ps.RequestCtx.Elements[11])
			return ps
		}

		ps.Attribute.Resources = []meta.Resource{
			meta.Resource{
				Basic: meta.Basic{
					Type:       meta.ObjectInstanceTopology,
					InstanceID: instID,
				},
				Action: meta.Find,
				Affiliated: meta.Affiliated{
					Type: meta.Object,
					Name: ps.RequestCtx.Elements[9],
				},
			},
		}

		return ps
	}

	// find business instance topology operation.
	if ps.hitRegexp(findBusinessInstanceTopologyRegexp, http.MethodGet) {
		if len(ps.RequestCtx.Elements) != 6 {
			ps.err = errors.New("find business instance topology, but got invalid url")
			return ps
		}

		bizID, err := strconv.ParseInt(ps.RequestCtx.Elements[5], 10, 64)
		if err != nil {
			ps.err = fmt.Errorf("find business instance topology, but got invalid instance id %s", ps.RequestCtx.Elements[5])
			return ps
		}

		ps.Attribute.Resources = []meta.Resource{
			meta.Resource{
				Basic: meta.Basic{
					Type:       meta.ObjectInstanceTopology,
					InstanceID: bizID,
				},
				Action: meta.Find,
				Affiliated: meta.Affiliated{
					Type: meta.Object,
					Name: string(meta.Business),
				},
			},
		}
		return ps
	}

	// find object's instance list operation
	if ps.hitRegexp(findObjectInstancesRegexp, http.MethodPost) {
		if len(ps.RequestCtx.Elements) != 8 {
			ps.err = errors.New("find object's instance  list, but got invalid url")
			return ps
		}

		ps.Attribute.Resources = []meta.Resource{
			meta.Resource{
				Basic: meta.Basic{
					Type: meta.ObjectInstanceTopology,
				},
				Action: meta.FindMany,
				Affiliated: meta.Affiliated{
					Type: meta.Object,
					Name: ps.RequestCtx.Elements[7],
				},
			},
		}
		return ps
	}

	return ps
}

const (
	createObjectPattern       = "/api/v3/object"
	findObjectsPattern        = "/api/v3/objects"
	findObjectTopologyPattern = "/api/v3/objects/topo"
)

var (
	deleteObjectRegexp                = regexp.MustCompile(`^/api/v3/object/[0-9]+$`)
	updateObjectRegexp                = regexp.MustCompile(`^/api/v3/object/[0-9]+$`)
	findObjectTopologyGraphicRegexp   = regexp.MustCompile(`^/api/v3/objects/topographics/scope_type/[\S][^/]+/scope_id/[\S][^/]+/action/search$`)
	updateObjectTopologyGraphicRegexp = regexp.MustCompile(`^/api/v3/objects/topographics/scope_type/[\S][^/]+/scope_id/[\S][^/]+/action/[a-z]+$`)
)

func (ps *parseStream) object() *parseStream {
	if ps.err != nil {
		return ps
	}

	// create common object operation.
	if ps.hitPattern(createObjectPattern, http.MethodPost) {
		ps.Attribute.Resources = []meta.Resource{
			meta.Resource{
				Basic: meta.Basic{
					Type: meta.Object,
				},
				Action: meta.Create,
			},
		}
		return ps
	}

	// delete object operation
	if ps.hitRegexp(deleteObjectRegexp, http.MethodDelete) {
		if len(ps.RequestCtx.Elements) != 4 {
			ps.err = errors.New("delete object, but got invalid url")
			return ps
		}

		objID, err := strconv.ParseInt(ps.RequestCtx.Elements[3], 10, 64)
		if err != nil {
			ps.err = fmt.Errorf("delete object, but got invalid object's id %s", ps.RequestCtx.Elements[3])
			return ps
		}

		ps.Attribute.Resources = []meta.Resource{
			meta.Resource{
				Basic: meta.Basic{
					Type:       meta.Object,
					InstanceID: objID,
				},
				Action: meta.Delete,
			},
		}
		return ps
	}

	// update object operation.
	if ps.hitRegexp(updateObjectRegexp, http.MethodPut) {
		if len(ps.RequestCtx.Elements) != 4 {
			ps.err = errors.New("update object, but got invalid url")
			return ps
		}

		objID, err := strconv.ParseInt(ps.RequestCtx.Elements[3], 10, 64)
		if err != nil {
			ps.err = fmt.Errorf("update object, but got invalid object's id %s", ps.RequestCtx.Elements[3])
			return ps
		}

		ps.Attribute.Resources = []meta.Resource{
			meta.Resource{
				Basic: meta.Basic{
					Type:       meta.Object,
					InstanceID: objID,
				},
				Action: meta.Update,
			},
		}
		return ps
	}

	// get object operation.
	if ps.hitPattern(findObjectsPattern, http.MethodPost) {
		ps.Attribute.Resources = []meta.Resource{
			meta.Resource{
				Basic: meta.Basic{
					Type: meta.Object,
				},
				Action: meta.FindMany,
			},
		}
		return ps
	}

	// find object's topology operation.
	if ps.hitPattern(findObjectTopologyPattern, http.MethodPost) {
		ps.Attribute.Resources = []meta.Resource{
			meta.Resource{
				Basic: meta.Basic{
					Type: meta.ObjectTopology,
				},
				Action: meta.Find,
			},
		}
		return ps
	}

	// find object's topology graphic operation.
	if ps.hitRegexp(findObjectTopologyGraphicRegexp, http.MethodPost) {
		ps.Attribute.Resources = []meta.Resource{
			meta.Resource{
				Basic: meta.Basic{
					Type: meta.ObjectTopology,
				},
				Action: meta.Find,
			},
		}
		return ps
	}

	// update object's topology graphic operation.
	if ps.hitRegexp(updateObjectTopologyGraphicRegexp, http.MethodPost) {
		ps.Attribute.Resources = []meta.Resource{
			meta.Resource{
				Basic: meta.Basic{
					Type: meta.ObjectTopology,
				},
				Action: meta.Update,
			},
		}
		return ps
	}

	return ps
}

const (
	createObjectClassificationPattern   = "/api/v3/object/classification"
	findObjectClassificationListPattern = "/api/v3/object/classifications"
)

var (
	deleteObjectClassificationRegexp         = regexp.MustCompile("^/api/v3/object/classification/[0-9]+$")
	updateObjectClassificationRegexp         = regexp.MustCompile("^/api/v3/object/classification/[0-9]+$")
	findObjectsBelongsToClassificationRegexp = regexp.MustCompile(`^/api/v3/object/classification/[\S][^/]+/objects$`)
)

func (ps *parseStream) ObjectClassification() *parseStream {
	if ps.err != nil {
		return ps
	}

	// create object's classification operation.
	if ps.hitPattern(createObjectClassificationPattern, http.MethodPost) {
		ps.Attribute.Resources = []meta.Resource{
			meta.Resource{
				Basic: meta.Basic{
					Type: meta.ObjectClassification,
				},
				Action: meta.Create,
			},
		}
		return ps
	}

	// delete object's classification operation.
	if ps.hitRegexp(deleteObjectClassificationRegexp, http.MethodDelete) {
		if len(ps.RequestCtx.Elements) != 5 {
			ps.err = errors.New("delete object classification, but got invalid url")
			return ps
		}

		classID, err := strconv.ParseInt(ps.RequestCtx.Elements[4], 10, 64)
		if err != nil {
			ps.err = fmt.Errorf("delete object classification, but got invalid object's id %s", ps.RequestCtx.Elements[4])
			return ps
		}

		ps.Attribute.Resources = []meta.Resource{
			meta.Resource{
				Basic: meta.Basic{
					Type:       meta.ObjectClassification,
					InstanceID: classID,
				},
				Action: meta.Delete,
			},
		}
		return ps
	}

	// update object's classification operation.
	if ps.hitRegexp(updateObjectClassificationRegexp, http.MethodPut) {
		if len(ps.RequestCtx.Elements) != 5 {
			ps.err = errors.New("update object classification, but got invalid url")
			return ps
		}

		classID, err := strconv.ParseInt(ps.RequestCtx.Elements[4], 10, 64)
		if err != nil {
			ps.err = fmt.Errorf("update object classification, but got invalid object's  classification id %s", ps.RequestCtx.Elements[4])
			return ps
		}

		ps.Attribute.Resources = []meta.Resource{
			meta.Resource{
				Basic: meta.Basic{
					Type:       meta.ObjectClassification,
					InstanceID: classID,
				},
				Action: meta.Update,
			},
		}
		return ps
	}

	// find object's classification list operation.
	if ps.hitPattern(findObjectClassificationListPattern, http.MethodPost) {
		ps.Attribute.Resources = []meta.Resource{
			meta.Resource{
				Basic: meta.Basic{
					Type: meta.ObjectClassification,
				},
				Action: meta.FindMany,
			},
		}
		return ps
	}

	// find all the objects belongs to a classification
	if ps.hitRegexp(findObjectsBelongsToClassificationRegexp, http.MethodPost) {
		ps.Attribute.Resources = []meta.Resource{
			meta.Resource{
				Basic: meta.Basic{
					Type: meta.ObjectClassification,
				},
				Action: meta.FindMany,
			},
		}
		return ps
	}

	return ps
}

const (
	createObjectAttributeGroupPattern = "/api/v3/objectatt/group/new"
	updateObjectAttributeGroupPattern = "/api/v3/objectatt/group/update"
)

var (
	findObjectAttributeGroupRegexp     = regexp.MustCompile(`^/api/v3/objectatt/group/property/owner/[\S][^/]+/object/[\S][^/]+$`)
	deleteObjectAttributeGroupRegexp   = regexp.MustCompile(`^/api/v3/objectatt/group/groupid/[0-9]+$`)
	removeAttributeAwayFromGroupRegexp = regexp.MustCompile(`^/api/v3/objectatt/group/owner/[\S][^/]+/object/[\S][^/]+/propertyids/[\S][^/]+/groupids/[\S][^/]+$`)
)

func (ps *parseStream) objectAttributeGroup() *parseStream {
	if ps.err != nil {
		return ps
	}

	// create object's attribute group operation.
	if ps.hitPattern(createObjectAttributeGroupPattern, http.MethodPost) {
		ps.Attribute.Resources = []meta.Resource{
			meta.Resource{
				Basic: meta.Basic{
					Type: meta.ObjectAttributeGroup,
				},
				Action: meta.Create,
			},
		}
		return ps
	}

	// find object's attribute group operation.
	if ps.hitRegexp(findObjectAttributeGroupRegexp, http.MethodPost) {
		if len(ps.RequestCtx.Elements) != 9 {
			ps.err = errors.New("find object's attribute group, but got invalid uri")
			return ps
		}

		ps.Attribute.Resources = []meta.Resource{
			meta.Resource{
				Basic: meta.Basic{
					Type: meta.ObjectAttributeGroup,
				},
				Action: meta.Find,
				Affiliated: meta.Affiliated{
					Type: meta.Object,
					Name: ps.RequestCtx.Elements[8],
				},
			},
		}
		return ps
	}

	// update object's attribute group operation.
	if ps.hitPattern(updateObjectAttributeGroupPattern, http.MethodPut) {
		ps.Attribute.Resources = []meta.Resource{
			meta.Resource{
				Basic: meta.Basic{
					Type: meta.ObjectClassification,
				},
				Action: meta.Update,
			},
		}
		return ps
	}

	// delete object's attribute group operation.
	if ps.hitRegexp(deleteObjectAttributeGroupRegexp, http.MethodDelete) {
		if len(ps.RequestCtx.Elements) != 6 {
			ps.err = errors.New("delete object's attribute group, but got invalid url")
			return ps
		}

		groupID, err := strconv.ParseInt(ps.RequestCtx.Elements[5], 10, 64)
		if err != nil {
			ps.err = fmt.Errorf("delete object's attribute group, but got invalid group's id %s", ps.RequestCtx.Elements[5])
			return ps
		}

		ps.Attribute.Resources = []meta.Resource{
			meta.Resource{
				Basic: meta.Basic{
					Type:       meta.ObjectAttributeGroup,
					InstanceID: groupID,
				},
				Action: meta.Delete,
			},
		}
		return ps
	}

	// remove a object's attribute away from a group.
	if ps.hitRegexp(removeAttributeAwayFromGroupRegexp, http.MethodDelete) {
		if len(ps.RequestCtx.Elements) != 12 {
			ps.err = errors.New("remove a object attribute away from a group, but got invalid uri")
			return ps
		}
		ps.Attribute.Resources = []meta.Resource{
			meta.Resource{
				Basic: meta.Basic{
					Type: meta.ObjectAttributeGroup,
					Name: ps.RequestCtx.Elements[11],
				},
				Action: meta.Delete,
			},
		}
		return ps
	}

	return ps
}

const (
	createObjectAttributePattern = "/api/v3/object/attr"
	findObjectAttributePattern   = "/api/v3/object/attr/search"
)

var (
	deleteObjectAttributeRegexp = regexp.MustCompile(`^/api/v3/object/attr/[0-9]+$`)
	updateObjectAttributeRegexp = regexp.MustCompile(`^/api/v3/object/attr/[0-9]+$`)
)

func (ps *parseStream) objectAttribute() *parseStream {
	if ps.err != nil {
		return ps
	}

	// create object's attribute operation.
	if ps.hitPattern(createObjectAttributePattern, http.MethodPost) {
		ps.Attribute.Resources = []meta.Resource{
			meta.Resource{
				Basic: meta.Basic{
					Type: meta.ObjectAttribute,
				},
				Action: meta.Create,
			},
		}
		return ps
	}

	// delete object's attribute operation.
	if ps.hitRegexp(deleteObjectAttributeRegexp, http.MethodDelete) {
		if len(ps.RequestCtx.Elements) != 5 {
			ps.err = errors.New("delete object attribute, but got invalid url")
			return ps
		}

		attrID, err := strconv.ParseInt(ps.RequestCtx.Elements[4], 10, 64)
		if err != nil {
			ps.err = fmt.Errorf("delete object attribute, but got invalid attribute id %s", ps.RequestCtx.Elements[4])
			return ps
		}

		ps.Attribute.Resources = []meta.Resource{
			meta.Resource{
				Basic: meta.Basic{
					Type:       meta.ObjectAttribute,
					InstanceID: attrID,
				},
				Action: meta.Delete,
			},
		}
		return ps
	}

	// update object attribute operation
	if ps.hitRegexp(updateObjectAttributeRegexp, http.MethodPut) {
		if len(ps.RequestCtx.Elements) != 5 {
			ps.err = errors.New("update object attribute, but got invalid url")
			return ps
		}

		attrID, err := strconv.ParseInt(ps.RequestCtx.Elements[4], 10, 64)
		if err != nil {
			ps.err = fmt.Errorf("update object attribute, but got invalid attribute id %s", ps.RequestCtx.Elements[4])
			return ps
		}

		ps.Attribute.Resources = []meta.Resource{
			meta.Resource{
				Basic: meta.Basic{
					Type:       meta.ObjectAttribute,
					InstanceID: attrID,
				},
				Action: meta.Update,
			},
		}
		return ps
	}

	// get object's attribute operation.
	if ps.hitPattern(findObjectAttributePattern, http.MethodPost) {
		ps.Attribute.Resources = []meta.Resource{
			meta.Resource{
				Basic: meta.Basic{
					Type: meta.ObjectAttribute,
				},
				Action: meta.Find,
			},
		}
		return ps
	}

	return ps
}

var (
	createModuleRegexp = regexp.MustCompile(`^/api/v3/module/[0-9]+/[0-9]+$`)
	deleteModuleRegexp = regexp.MustCompile(`^/api/v3/module/[0-9]+/[0-9]+/[0-9]+$`)
	updateModuleRegexp = regexp.MustCompile(`^/api/v3/module/[0-9]+/[0-9]+/[0-9]+$`)
	findModuleRegexp   = regexp.MustCompile(`^/api/v3/module/search/[\S][^/]+/[0-9]+/[0-9]+$`)
)

func (ps *parseStream) ObjectModule() *parseStream {
	if ps.err != nil {
		return ps
	}

	// create module
	if ps.hitRegexp(createModuleRegexp, http.MethodPost) {
		if len(ps.RequestCtx.Elements) != 5 {
			ps.err = errors.New("create module, but got invalid url")
			return ps
		}

		bizID, err := strconv.ParseInt(ps.RequestCtx.Elements[3], 10, 64)
		if err != nil {
			ps.err = fmt.Errorf("create module, but got invalid business id %s", ps.RequestCtx.Elements[3])
			return ps
		}

		setID, err := strconv.ParseInt(ps.RequestCtx.Elements[4], 10, 64)
		if err != nil {
			ps.err = fmt.Errorf("create module, but got invalid set id %s", ps.RequestCtx.Elements[4])
			return ps
		}

		ps.Attribute.BusinessID = bizID
		ps.Attribute.Resources = []meta.Resource{
			meta.Resource{
				Basic: meta.Basic{
					Type: meta.ObjectModule,
				},
				Action: meta.Create,
				Affiliated: meta.Affiliated{
					Type:       meta.ObjectInstance,
					Name:       "set",
					InstanceID: setID,
				},
			},
		}
		return ps
	}

	// delete module operation.
	if ps.hitRegexp(deleteModuleRegexp, http.MethodDelete) {
		if len(ps.RequestCtx.Elements) != 6 {
			ps.err = errors.New("delete module, but got invalid url")
			return ps
		}

		bizID, err := strconv.ParseInt(ps.RequestCtx.Elements[3], 10, 64)
		if err != nil {
			ps.err = fmt.Errorf("delete module, but got invalid business id %s", ps.RequestCtx.Elements[3])
			return ps
		}

		setID, err := strconv.ParseInt(ps.RequestCtx.Elements[4], 10, 64)
		if err != nil {
			ps.err = fmt.Errorf("delete module, but got invalid set id %s", ps.RequestCtx.Elements[4])
			return ps
		}

		moduleID, err := strconv.ParseInt(ps.RequestCtx.Elements[5], 10, 64)
		if err != nil {
			ps.err = fmt.Errorf("delete module, but got invalid module id %s", ps.RequestCtx.Elements[5])
			return ps
		}

		ps.Attribute.BusinessID = bizID
		ps.Attribute.Resources = []meta.Resource{
			meta.Resource{
				Basic: meta.Basic{
					Type:       meta.ObjectModule,
					InstanceID: moduleID,
				},
				Action: meta.Delete,
				Affiliated: meta.Affiliated{
					Type:       meta.ObjectInstance,
					Name:       "set",
					InstanceID: setID,
				},
			},
		}
		return ps
	}

	// update module operation.
	if ps.hitRegexp(updateModuleRegexp, http.MethodPut) {
		if len(ps.RequestCtx.Elements) != 6 {
			ps.err = errors.New("update module, but got invalid url")
			return ps
		}

		bizID, err := strconv.ParseInt(ps.RequestCtx.Elements[3], 10, 64)
		if err != nil {
			ps.err = fmt.Errorf("update module, but got invalid business id %s", ps.RequestCtx.Elements[3])
			return ps
		}

		setID, err := strconv.ParseInt(ps.RequestCtx.Elements[4], 10, 64)
		if err != nil {
			ps.err = fmt.Errorf("update module, but got invalid set id %s", ps.RequestCtx.Elements[4])
			return ps
		}

		moduleID, err := strconv.ParseInt(ps.RequestCtx.Elements[5], 10, 64)
		if err != nil {
			ps.err = fmt.Errorf("update module, but got invalid module id %s", ps.RequestCtx.Elements[5])
			return ps
		}
		ps.Attribute.BusinessID = bizID
		ps.Attribute.Resources = []meta.Resource{
			meta.Resource{
				Basic: meta.Basic{
					Type:       meta.ObjectModule,
					InstanceID: moduleID,
				},
				Action: meta.Update,
				Affiliated: meta.Affiliated{
					Type:       meta.ObjectInstance,
					Name:       "set",
					InstanceID: setID,
				},
			},
		}
		return ps
	}

	// find module operation.
	if ps.hitRegexp(findObjectTopologyGraphicRegexp, http.MethodPost) {
		if len(ps.RequestCtx.Elements) != 7 {
			ps.err = errors.New("find module, but got invalid url")
			return ps
		}

		bizID, err := strconv.ParseInt(ps.RequestCtx.Elements[5], 10, 64)
		if err != nil {
			ps.err = fmt.Errorf("find module, but got invalid business id %s", ps.RequestCtx.Elements[5])
			return ps
		}

		setID, err := strconv.ParseInt(ps.RequestCtx.Elements[6], 10, 64)
		if err != nil {
			ps.err = fmt.Errorf("find module, but got invalid set id %s", ps.RequestCtx.Elements[6])
			return ps
		}
		ps.Attribute.BusinessID = bizID
		ps.Attribute.Resources = []meta.Resource{
			meta.Resource{
				Basic: meta.Basic{
					Type: meta.ObjectModule,
				},
				Action: meta.FindMany,
				Affiliated: meta.Affiliated{
					Type:       meta.ObjectSet,
					InstanceID: setID,
				},
			},
		}
		return ps
	}

	return ps
}

var (
	createSetRegexp     = regexp.MustCompile(`^/api/v3/set/[0-9]+$`)
	deleteSetRegexp     = regexp.MustCompile(`^/api/v3/set/[0-9]+/[0-9]+$`)
	deleteManySetRegexp = regexp.MustCompile(`^/api/v3/set/[0-9]+/batch$`)
	updateSetRegexp     = regexp.MustCompile(`^/api/v3/set/[0-9]+/[0-9]+$`)
	findSetRegexp       = regexp.MustCompile(`^/api/v3/set/search/[\S][^/]+/[0-9]+$`)
)

func (ps *parseStream) ObjectSet() *parseStream {
	if ps.err != nil {
		return ps
	}

	// create set
	if ps.hitRegexp(createSetRegexp, http.MethodPost) {
		if len(ps.RequestCtx.Elements) != 4 {
			ps.err = errors.New("create set, but got invalid url")
			return ps
		}

		bizID, err := strconv.ParseInt(ps.RequestCtx.Elements[3], 10, 64)
		if err != nil {
			ps.err = fmt.Errorf("create set, but got invalid business id %s", ps.RequestCtx.Elements[3])
			return ps
		}
		ps.Attribute.BusinessID = bizID
		ps.Attribute.Resources = []meta.Resource{
			meta.Resource{
				Basic: meta.Basic{
					Type: meta.ObjectSet,
				},
				Action: meta.Create,
			},
		}
		return ps
	}

	// delete set operation.
	if ps.hitRegexp(deleteSetRegexp, http.MethodDelete) {
		if len(ps.RequestCtx.Elements) != 5 {
			ps.err = errors.New("delete set, but got invalid url")
			return ps
		}

		bizID, err := strconv.ParseInt(ps.RequestCtx.Elements[3], 10, 64)
		if err != nil {
			ps.err = fmt.Errorf("delete set, but got invalid business id %s", ps.RequestCtx.Elements[3])
			return ps
		}

		setID, err := strconv.ParseInt(ps.RequestCtx.Elements[4], 10, 64)
		if err != nil {
			ps.err = fmt.Errorf("delete set, but got invalid set id %s", ps.RequestCtx.Elements[4])
			return ps
		}
		ps.Attribute.BusinessID = bizID
		ps.Attribute.Resources = []meta.Resource{
			meta.Resource{
				Basic: meta.Basic{
					Type:       meta.ObjectSet,
					InstanceID: setID,
				},
				Action: meta.Delete,
			},
		}
		return ps
	}

	// delete many set operation.
	if ps.hitRegexp(deleteManySetRegexp, http.MethodDelete) {
		if len(ps.RequestCtx.Elements) != 5 {
			ps.err = errors.New("delete set list, but got invalid url")
			return ps
		}

		bizID, err := strconv.ParseInt(ps.RequestCtx.Elements[3], 10, 64)
		if err != nil {
			ps.err = fmt.Errorf("delete set list, but got invalid business id %s", ps.RequestCtx.Elements[3])
			return ps
		}
		ps.Attribute.BusinessID = bizID
		ps.Attribute.Resources = []meta.Resource{
			meta.Resource{
				Basic: meta.Basic{
					Type: meta.ObjectSet,
				},
				Action: meta.DeleteMany,
			},
		}
		return ps
	}

	// update set operation.
	if ps.hitRegexp(updateSetRegexp, http.MethodPut) {
		if len(ps.RequestCtx.Elements) != 5 {
			ps.err = errors.New("update set, but got invalid url")
			return ps
		}

		bizID, err := strconv.ParseInt(ps.RequestCtx.Elements[3], 10, 64)
		if err != nil {
			ps.err = fmt.Errorf("update set, but got invalid business id %s", ps.RequestCtx.Elements[3])
			return ps
		}

		setID, err := strconv.ParseInt(ps.RequestCtx.Elements[4], 10, 64)
		if err != nil {
			ps.err = fmt.Errorf("update set, but got invalid set id %s", ps.RequestCtx.Elements[4])
			return ps
		}
		ps.Attribute.BusinessID = bizID
		ps.Attribute.Resources = []meta.Resource{
			meta.Resource{
				Basic: meta.Basic{
					Type:       meta.ObjectSet,
					InstanceID: setID,
				},
				Action: meta.Update,
			},
		}
		return ps
	}

	// find set operation.
	if ps.hitRegexp(findObjectTopologyGraphicRegexp, http.MethodPost) {
		if len(ps.RequestCtx.Elements) != 6 {
			ps.err = errors.New("find set, but got invalid url")
			return ps
		}

		bizID, err := strconv.ParseInt(ps.RequestCtx.Elements[5], 10, 64)
		if err != nil {
			ps.err = fmt.Errorf("find set, but got invalid business id %s", ps.RequestCtx.Elements[5])
			return ps
		}
		ps.Attribute.BusinessID = bizID
		ps.Attribute.Resources = []meta.Resource{
			meta.Resource{
				Basic: meta.Basic{
					Type: meta.ObjectSet,
				},
				Action: meta.FindMany,
			},
		}
		return ps
	}

	return ps
}

var (
	createObjectUniqueRegexp = regexp.MustCompile(`^/api/v3/object/[\S][^/]+/unique/action/create$`)
	updateObjectUniqueRegexp = regexp.MustCompile(`^/api/v3/object/[\S][^/]+/unique/[0-9]+/action/update$`)
	deleteObjectUniqueRegexp = regexp.MustCompile(`^/api/v3/object/[\S][^/]+/unique/[0-9]+/action/delete$`)
	findObjectUniqueRegexp   = regexp.MustCompile(`^/api/v3/object/[\S][^/]+/unique/action/search$`)
)

func (ps *parseStream) objectUnique() *parseStream {
	if ps.err != nil {
		return ps
	}

	// add object unique operation.
	if ps.hitRegexp(createObjectUniqueRegexp, http.MethodPost) {
		ps.Attribute.Resources = []meta.Resource{
			meta.Resource{
				Basic: meta.Basic{
					Type: meta.ObjectUnique,
					Name: ps.RequestCtx.Elements[3],
				},
				Action: meta.Create,
			},
		}
		return ps
	}

	// update object unique operation.
	if ps.hitRegexp(updateObjectUniqueRegexp, http.MethodPut) {
		uniqueID, err := strconv.ParseInt(ps.RequestCtx.Elements[5], 10, 64)
		if err != nil {
			ps.err = fmt.Errorf("update object unique, but got invalid unique id %s", ps.RequestCtx.Elements[5])
			return ps
		}

		ps.Attribute.Resources = []meta.Resource{
			meta.Resource{
				Basic: meta.Basic{
					Type:       meta.ObjectUnique,
					InstanceID: uniqueID,
				},
				Action: meta.Update,
				Affiliated: meta.Affiliated{
					Type: meta.Object,
					Name: ps.RequestCtx.Elements[3],
				},
			},
		}
		return ps
	}

	// delete object unique operation.
	if ps.hitRegexp(deleteObjectUniqueRegexp, http.MethodDelete) {
		uniqueID, err := strconv.ParseInt(ps.RequestCtx.Elements[5], 10, 64)
		if err != nil {
			ps.err = fmt.Errorf("update object unique, but got invalid unique id %s", ps.RequestCtx.Elements[5])
			return ps
		}

		ps.Attribute.Resources = []meta.Resource{
			meta.Resource{
				Basic: meta.Basic{
					Type:       meta.ObjectUnique,
					InstanceID: uniqueID,
				},
				Action: meta.Delete,
				Affiliated: meta.Affiliated{
					Type: meta.Object,
					Name: ps.RequestCtx.Elements[3],
				},
			},
		}
		return ps
	}

	// find object unique operation.
	if ps.hitRegexp(findObjectUniqueRegexp, http.MethodGet) {
		ps.Attribute.Resources = []meta.Resource{
			meta.Resource{
				Basic: meta.Basic{
					Type: meta.ObjectUnique,
				},
				Action: meta.FindMany,
				Affiliated: meta.Affiliated{
					Type: meta.Object,
					Name: ps.RequestCtx.Elements[5],
				},
			},
		}
		return ps
	}

	return ps
}
