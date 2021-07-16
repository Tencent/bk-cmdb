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
	"net/http"

	"configcenter/src/common/http/rest"

	"github.com/emicklei/go-restful"
)

func (s *Service) initBusinessObject(web *restful.WebService) {
	utility := rest.NewRestUtility(rest.Config{
		ErrorIf:  s.Engine.CCErr,
		Language: s.Engine.Language,
	})

	utility.AddHandler(rest.Action{Verb: http.MethodPost, Path: "/createmany/object", Handler: s.CreateObjectBatch})
	utility.AddHandler(rest.Action{Verb: http.MethodPost, Path: "/findmany/object", Handler: s.SearchObjectBatch})
	utility.AddHandler(rest.Action{Verb: http.MethodPost, Path: "/create/object", Handler: s.CreateObject})
	utility.AddHandler(rest.Action{Verb: http.MethodPost, Path: "/find/object", Handler: s.SearchObject})
	utility.AddHandler(rest.Action{Verb: http.MethodPut, Path: "/update/object/{id}", Handler: s.UpdateObject})
	utility.AddHandler(rest.Action{Verb: http.MethodDelete, Path: "/delete/object/{id}", Handler: s.DeleteObject})
	utility.AddHandler(rest.Action{Verb: http.MethodPost, Path: "/find/objecttopology", Handler: s.SearchObjectTopo})

	utility.AddToRestfulWebService(web)
}

func (s *Service) initBusinessClassification(web *restful.WebService) {
	utility := rest.NewRestUtility(rest.Config{
		ErrorIf:  s.Engine.CCErr,
		Language: s.Engine.Language,
	})

	utility.AddHandler(rest.Action{Verb: http.MethodPost, Path: "/create/objectclassification", Handler: s.CreateClassification})
	utility.AddHandler(rest.Action{Verb: http.MethodPost, Path: "/find/classificationobject", Handler: s.SearchClassificationWithObjects})
	utility.AddHandler(rest.Action{Verb: http.MethodPost, Path: "/find/objectclassification", Handler: s.SearchClassification})
	utility.AddHandler(rest.Action{Verb: http.MethodPut, Path: "/update/objectclassification/{id}", Handler: s.UpdateClassification})
	utility.AddHandler(rest.Action{Verb: http.MethodDelete, Path: "/delete/objectclassification/{id}", Handler: s.DeleteClassification})

	utility.AddToRestfulWebService(web)
}

func (s *Service) initBusinessObjectAttribute(web *restful.WebService) {
	utility := rest.NewRestUtility(rest.Config{
		ErrorIf:  s.Engine.CCErr,
		Language: s.Engine.Language,
	})

	utility.AddHandler(rest.Action{Verb: http.MethodPost, Path: "/create/objectattr", Handler: s.CreateObjectAttribute})
	utility.AddHandler(rest.Action{Verb: http.MethodPost, Path: "/create/objectattr/biz/{bk_biz_id}", Handler: s.CreateObjectAttribute})
	utility.AddHandler(rest.Action{Verb: http.MethodPost, Path: "/find/objectattr", Handler: s.SearchObjectAttribute})
	utility.AddHandler(rest.Action{Verb: http.MethodPost, Path: "/find/objectattr/host", Handler: s.ListHostModelAttribute})
	utility.AddHandler(rest.Action{Verb: http.MethodPut, Path: "/update/objectattr/{id}", Handler: s.UpdateObjectAttribute})
	utility.AddHandler(rest.Action{Verb: http.MethodPut, Path: "/update/objectattr/biz/{bk_biz_id}/id/{id}", Handler: s.UpdateObjectAttribute})
	utility.AddHandler(rest.Action{Verb: http.MethodDelete, Path: "/delete/objectattr/{id}", Handler: s.DeleteObjectAttribute})

	utility.AddToRestfulWebService(web)
}

func (s *Service) initBusinessObjectUnique(web *restful.WebService) {
	utility := rest.NewRestUtility(rest.Config{
		ErrorIf:  s.Engine.CCErr,
		Language: s.Engine.Language,
	})

	utility.AddHandler(rest.Action{Verb: http.MethodPost, Path: "/create/objectunique/object/{bk_obj_id}", Handler: s.CreateObjectUnique})
	utility.AddHandler(rest.Action{Verb: http.MethodPut, Path: "/update/objectunique/object/{bk_obj_id}/unique/{id}", Handler: s.UpdateObjectUnique})
	utility.AddHandler(rest.Action{Verb: http.MethodPost, Path: "/delete/objectunique/object/{bk_obj_id}/unique/{id}", Handler: s.DeleteObjectUnique})
	utility.AddHandler(rest.Action{Verb: http.MethodPost, Path: "/find/objectunique/object/{bk_obj_id}", Handler: s.SearchObjectUnique})

	utility.AddToRestfulWebService(web)
}

func (s *Service) initBusinessObjectAttrGroup(web *restful.WebService) {
	utility := rest.NewRestUtility(rest.Config{
		ErrorIf:  s.Engine.CCErr,
		Language: s.Engine.Language,
	})

	utility.AddHandler(rest.Action{Verb: http.MethodPost, Path: "/create/objectattgroup", Handler: s.CreateObjectGroup})
	utility.AddHandler(rest.Action{Verb: http.MethodPut, Path: "/update/objectattgroup", Handler: s.UpdateObjectGroup})
	utility.AddHandler(rest.Action{Verb: http.MethodDelete, Path: "/delete/objectattgroup/{id}", Handler: s.DeleteObjectGroup})
	utility.AddHandler(rest.Action{Verb: http.MethodPut, Path: "/update/objectattgroupproperty", Handler: s.UpdateObjectAttributeGroupProperty})
	utility.AddHandler(rest.Action{Verb: http.MethodPost, Path: "/find/objectattgroup/object/{bk_obj_id}", Handler: s.SearchGroupByObject})

	utility.AddToRestfulWebService(web)
}

func (s *Service) initBusinessGraphics(web *restful.WebService) {
	utility := rest.NewRestUtility(rest.Config{
		ErrorIf:  s.Engine.CCErr,
		Language: s.Engine.Language,
	})

	utility.AddHandler(rest.Action{Verb: http.MethodPost, Path: "/find/objecttopo/scope_type/{scope_type}/scope_id/{scope_id}", Handler: s.SelectObjectTopoGraphics})
	utility.AddHandler(rest.Action{Verb: http.MethodPost, Path: "/update/objecttopo/scope_type/{scope_type}/scope_id/{scope_id}", Handler: s.UpdateObjectTopoGraphicsNew})

	utility.AddToRestfulWebService(web)
}

func (s *Service) initBusinessAssociation(web *restful.WebService) {
	utility := rest.NewRestUtility(rest.Config{
		ErrorIf:  s.Engine.CCErr,
		Language: s.Engine.Language,
	})

	// mainline topo methods
	utility.AddHandler(rest.Action{Verb: http.MethodPost, Path: "/create/topomodelmainline", Handler: s.CreateMainLineObject})
	utility.AddHandler(rest.Action{Verb: http.MethodDelete, Path: "/delete/topomodelmainline/object/{bk_obj_id}", Handler: s.DeleteMainLineObject})
	utility.AddHandler(rest.Action{Verb: http.MethodPost, Path: "/find/topomodelmainline", Handler: s.SearchMainLineObjectTopo})
	utility.AddHandler(rest.Action{Verb: http.MethodPost, Path: "/find/topoinst/biz/{bk_biz_id}", Handler: s.SearchBusinessTopo})
	utility.AddHandler(rest.Action{Verb: http.MethodPost, Path: "/find/topoinst_with_statistics/biz/{bk_biz_id}", Handler: s.SearchBusinessTopoWithStatistics})
	utility.AddHandler(rest.Action{Verb: http.MethodPost, Path: "/find/topoinstnode/host_serviceinst_count/{bk_biz_id}", Handler: s.SearchBusinessTopoNodeHostAndServiceInstanceCount})
	utility.AddHandler(rest.Action{Verb: http.MethodPost, Path: "/find/topoinst/bk_biz_id/{bk_biz_id}/host_apply_rule_related", Handler: s.SearchRuleRelatedTopoNodes})
	utility.AddHandler(rest.Action{Verb: http.MethodPost, Path: "/find/topopath/biz/{bk_biz_id}", Handler: s.SearchTopoPath})

	// association type methods ,NOT SUPPORT BUSINESS
	utility.AddHandler(rest.Action{Verb: http.MethodPost, Path: "/find/topoassociationtype", Handler: s.SearchObjectAssocWithAssocKindList})
	utility.AddHandler(rest.Action{Verb: http.MethodPost, Path: "/find/associationtype", Handler: s.SearchAssociationType})
	utility.AddHandler(rest.Action{Verb: http.MethodPost, Path: "/create/associationtype", Handler: s.CreateAssociationType})
	utility.AddHandler(rest.Action{Verb: http.MethodPut, Path: "/update/associationtype/{id}", Handler: s.UpdateAssociationType})
	utility.AddHandler(rest.Action{Verb: http.MethodDelete, Path: "/delete/associationtype/{id}", Handler: s.DeleteAssociationType})

	// object association methods
	utility.AddHandler(rest.Action{Verb: http.MethodPost, Path: "/find/objectassociation", Handler: s.SearchObjectAssociation})
	utility.AddHandler(rest.Action{Verb: http.MethodPost, Path: "/create/objectassociation", Handler: s.CreateObjectAssociation})
	utility.AddHandler(rest.Action{Verb: http.MethodPut, Path: "/update/objectassociation/{id}", Handler: s.UpdateObjectAssociation})
	utility.AddHandler(rest.Action{Verb: http.MethodDelete, Path: "/delete/objectassociation/{id}", Handler: s.DeleteObjectAssociation})

	// inst association methods
	utility.AddHandler(rest.Action{Verb: http.MethodPost, Path: "/find/instassociation", Handler: s.SearchAssociationInst})
	utility.AddHandler(rest.Action{Verb: http.MethodPost, Path: "/find/instassociation/related", Handler: s.SearchAssociationRelatedInst})
	utility.AddHandler(rest.Action{Verb: http.MethodPost, Path: "/create/instassociation", Handler: s.CreateAssociationInst})
	utility.AddHandler(rest.Action{Verb: http.MethodDelete, Path: "/delete/instassociation/{association_id}", Handler: s.DeleteAssociationInst})
	utility.AddHandler(rest.Action{Verb: http.MethodDelete, Path: "/delete/instassociation/batch", Handler: s.DeleteAssociationInstBatch})

	// topo search methods
	utility.AddHandler(rest.Action{Verb: http.MethodPost, Path: "/find/instassociation/object/{bk_obj_id}", Handler: s.SearchInstByAssociation})
	utility.AddHandler(rest.Action{Verb: http.MethodPost, Path: "/find/instassttopo/object/{bk_obj_id}/inst/{inst_id}", Handler: s.SearchInstTopo})

	// ATTENTION: the following methods is not recommended
	utility.AddHandler(rest.Action{Verb: http.MethodPost, Path: "/find/insttopo/object/{bk_obj_id}/inst/{inst_id}", Handler: s.SearchInstChildTopo})
	utility.AddHandler(rest.Action{Verb: http.MethodPost, Path: "/import/instassociation/{bk_obj_id}", Handler: s.ImportInstanceAssociation})

	utility.AddToRestfulWebService(web)
}

func (s *Service) initBusinessInst(web *restful.WebService) {
	utility := rest.NewRestUtility(rest.Config{
		ErrorIf:  s.Engine.CCErr,
		Language: s.Engine.Language,
	})

	utility.AddHandler(rest.Action{Verb: http.MethodPost, Path: "/create/instance/object/{bk_obj_id}", Handler: s.CreateInst})
	utility.AddHandler(rest.Action{Verb: http.MethodDelete, Path: "/delete/instance/object/{bk_obj_id}/inst/{inst_id}", Handler: s.DeleteInst})
	utility.AddHandler(rest.Action{Verb: http.MethodDelete, Path: "/deletemany/instance/object/{bk_obj_id}", Handler: s.DeleteInsts})
	utility.AddHandler(rest.Action{Verb: http.MethodPut, Path: "/update/instance/object/{bk_obj_id}/inst/{inst_id}", Handler: s.UpdateInst})
	utility.AddHandler(rest.Action{Verb: http.MethodPut, Path: "/updatemany/instance/object/{bk_obj_id}", Handler: s.UpdateInsts})
	utility.AddHandler(rest.Action{Verb: http.MethodPost, Path: "/find/instance/object/{bk_obj_id}", Handler: s.SearchInstAndAssociationDetail})
	utility.AddHandler(rest.Action{Verb: http.MethodPost, Path: "/find/instance/object/{bk_obj_id}/unique_fields", Handler: s.SearchInstUniqueFields})
	utility.AddHandler(rest.Action{Verb: http.MethodPost, Path: "/find/instdetail/object/{bk_obj_id}/inst/{inst_id}", Handler: s.SearchInstByInstID})
	utility.AddHandler(rest.Action{Verb: http.MethodPost, Path: "/findmany/object/instances/names", Handler: s.SearchInstsNames})

	utility.AddToRestfulWebService(web)
}
