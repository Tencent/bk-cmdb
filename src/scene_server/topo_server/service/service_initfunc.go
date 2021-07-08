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

func (s *Service) initAssociation(web *restful.WebService) {
	utility := rest.NewRestUtility(rest.Config{
		ErrorIf:  s.Engine.CCErr,
		Language: s.Engine.Language,
	})

	// mainline topo methods
	utility.AddHandler(rest.Action{Verb: http.MethodGet, Path: "/topo/model/{owner_id}/{cls_id}/{bk_obj_id}", Handler: s.SearchObjectByClassificationID})
	utility.AddHandler(rest.Action{Verb: http.MethodPost, Path: "/find/topo/tree/brief/biz/{bk_biz_id}", Handler: s.SearchBriefBizTopo})
	utility.AddHandler(rest.Action{Verb: http.MethodPost, Path: "/find/topo/biz/brief_node_relation", Handler: s.GetBriefTopologyNodeRelation})

	utility.AddToRestfulWebService(web)
}

func (s *Service) initAuditLog(web *restful.WebService) {
	utility := rest.NewRestUtility(rest.Config{
		ErrorIf:  s.Engine.CCErr,
		Language: s.Engine.Language,
	})

	utility.AddHandler(rest.Action{Verb: http.MethodGet, Path: "/find/audit_dict", Handler: s.SearchAuditDict})
	utility.AddHandler(rest.Action{Verb: http.MethodPost, Path: "/findmany/audit_list", Handler: s.SearchAuditList})
	utility.AddHandler(rest.Action{Verb: http.MethodPost, Path: "/find/audit", Handler: s.SearchAuditDetail})

	utility.AddToRestfulWebService(web)
}

func (s *Service) initBusiness(web *restful.WebService) {
	utility := rest.NewRestUtility(rest.Config{
		ErrorIf:  s.Engine.CCErr,
		Language: s.Engine.Language,
	})

	utility.AddHandler(rest.Action{Verb: http.MethodPost, Path: "/app/search/{owner_id}", Handler: s.SearchBusiness})
	utility.AddHandler(rest.Action{Verb: http.MethodPost, Path: "/app/{owner_id}", Handler: s.CreateBusiness})
	utility.AddHandler(rest.Action{Verb: http.MethodPut, Path: "/app/{owner_id}/{app_id}", Handler: s.UpdateBusiness})
	utility.AddHandler(rest.Action{Verb: http.MethodPut, Path: "/app/status/{flag}/{owner_id}/{app_id}", Handler: s.UpdateBusinessStatus})
	// utility.AddHandler(rest.Action{Verb: http.MethodPost, Path: "/app/search/{owner_id}", Handler: s.SearchBusiness})
	utility.AddHandler(rest.Action{Verb: http.MethodGet, Path: "/app/{app_id}/basic_info", Handler: s.GetBusinessBasicInfo})
	utility.AddHandler(rest.Action{Verb: http.MethodPost, Path: "/app/default/{owner_id}/search", Handler: s.SearchOwnerResourcePoolBusiness})
	utility.AddHandler(rest.Action{Verb: http.MethodPost, Path: "/app/default/{owner_id}", Handler: s.CreateDefaultBusiness})
	utility.AddHandler(rest.Action{Verb: http.MethodGet, Path: "/topo/internal/{owner_id}/{app_id}", Handler: s.GetInternalModule})
	utility.AddHandler(rest.Action{Verb: http.MethodGet, Path: "/topo/internal/{owner_id}/{app_id}/with_statistics", Handler: s.GetInternalModuleWithStatistics})
	// find reduced business list with only few fields for business itself.
	utility.AddHandler(rest.Action{Verb: http.MethodGet, Path: "/app/with_reduced", Handler: s.SearchReducedBusinessList})
	utility.AddHandler(rest.Action{Verb: http.MethodGet, Path: "/app/simplify", Handler: s.ListAllBusinessSimplify})

	utility.AddToRestfulWebService(web)
}

func (s *Service) initModule(web *restful.WebService) {
	utility := rest.NewRestUtility(rest.Config{
		ErrorIf:  s.Engine.CCErr,
		Language: s.Engine.Language,
	})

	utility.AddHandler(rest.Action{Verb: http.MethodPost, Path: "/module/{app_id}/{set_id}", Handler: s.CreateModule})
	utility.AddHandler(rest.Action{Verb: http.MethodDelete, Path: "/module/{app_id}/{set_id}/{module_id}", Handler: s.DeleteModule})
	utility.AddHandler(rest.Action{Verb: http.MethodPut, Path: "/module/{app_id}/{set_id}/{module_id}", Handler: s.UpdateModule})
	utility.AddHandler(rest.Action{Verb: http.MethodPost, Path: "/module/search/{owner_id}/{bk_biz_id}/{bk_set_id}", Handler: s.SearchModule})
	utility.AddHandler(rest.Action{Verb: http.MethodPost, Path: "/findmany/module/biz/{bk_biz_id}", Handler: s.SearchModuleByCondition})
	utility.AddHandler(rest.Action{Verb: http.MethodPost, Path: "/findmany/module/bk_biz_id/{bk_biz_id}", Handler: s.SearchModuleBatch})
	utility.AddHandler(rest.Action{Verb: http.MethodPost, Path: "/findmany/module/with_relation/biz/{bk_biz_id}", Handler: s.SearchModuleWithRelation})
	utility.AddHandler(rest.Action{Verb: http.MethodPost, Path: "/module/bk_biz_id/{bk_biz_id}/service_template_id/{service_template_id}", Handler: s.ListModulesByServiceTemplateID})
	utility.AddHandler(rest.Action{Verb: http.MethodPut, Path: "/module/host_apply_enable_status/bk_biz_id/{bk_biz_id}/bk_module_id/{bk_module_id}", Handler: s.UpdateModuleHostApplyEnableStatus})

	utility.AddToRestfulWebService(web)
}

func (s *Service) initSet(web *restful.WebService) {
	utility := rest.NewRestUtility(rest.Config{
		ErrorIf:  s.Engine.CCErr,
		Language: s.Engine.Language,
	})

	utility.AddHandler(rest.Action{Verb: http.MethodPost, Path: "/set/{app_id}", Handler: s.CreateSet})
	utility.AddHandler(rest.Action{Verb: http.MethodPost, Path: "/set/{app_id}/batch", Handler: s.BatchCreateSet})
	utility.AddHandler(rest.Action{Verb: http.MethodDelete, Path: "/set/{app_id}/{set_id}", Handler: s.DeleteSet})
	utility.AddHandler(rest.Action{Verb: http.MethodPut, Path: "/set/{app_id}/{set_id}", Handler: s.UpdateSet})
	utility.AddHandler(rest.Action{Verb: http.MethodPost, Path: "/set/search/{owner_id}/{app_id}", Handler: s.SearchSet})
	utility.AddHandler(rest.Action{Verb: http.MethodPost, Path: "/findmany/set/bk_biz_id/{bk_biz_id}", Handler: s.SearchSetBatch})

	utility.AddToRestfulWebService(web)
}

func (s *Service) initInst(web *restful.WebService) {
	utility := rest.NewRestUtility(rest.Config{
		ErrorIf:  s.Engine.CCErr,
		Language: s.Engine.Language,
	})

	utility.AddHandler(rest.Action{Verb: http.MethodPost, Path: "/inst/search/{owner_id}/{bk_obj_id}", Handler: s.SearchInsts})
	utility.AddHandler(rest.Action{Verb: http.MethodPost, Path: "/findmany/inst/association/object/{bk_obj_id}/inst_id/{id}/offset/{start}/limit/{limit}/web", Handler: s.SearchInstAssociationUI})
	utility.AddHandler(rest.Action{Verb: http.MethodPost, Path: "/findmany/inst/association/association_object/inst_base_info", Handler: s.SearchInstAssociationWithOtherObject})

	utility.AddToRestfulWebService(web)
}

func (s *Service) initObjectAttribute(web *restful.WebService) {
	utility := rest.NewRestUtility(rest.Config{
		ErrorIf:  s.Engine.CCErr,
		Language: s.Engine.Language,
	})

	utility.AddHandler(rest.Action{Verb: http.MethodPost, Path: "/update/objectattr/index/{bk_obj_id}/{id}", Handler: s.UpdateObjectAttributeIndex})

	utility.AddToRestfulWebService(web)
}

func (s *Service) initObjectGroup(web *restful.WebService) {
	utility := rest.NewRestUtility(rest.Config{
		ErrorIf:  s.Engine.CCErr,
		Language: s.Engine.Language,
	})

	utility.AddHandler(rest.Action{Verb: http.MethodDelete, Path: "/objectatt/group/owner/{owner_id}/object/{bk_object_id}/propertyids/{property_id}/groupids/{group_id}", Handler: s.DeleteObjectAttributeGroup})

	utility.AddToRestfulWebService(web)
}

func (s *Service) initObject(web *restful.WebService) {
	utility := rest.NewRestUtility(rest.Config{
		ErrorIf:  s.Engine.CCErr,
		Language: s.Engine.Language,
	})

	utility.AddHandler(rest.Action{Verb: http.MethodGet, Path: "/object/statistics", Handler: s.GetModelStatistics})

	utility.AddToRestfulWebService(web)
}

func (s *Service) initIdentifier(web *restful.WebService) {
	utility := rest.NewRestUtility(rest.Config{
		ErrorIf:  s.Engine.CCErr,
		Language: s.Engine.Language,
	})

	utility.AddHandler(rest.Action{Verb: http.MethodPost, Path: "/identifier/{obj_type}/search", Handler: s.SearchIdentifier})

	utility.AddToRestfulWebService(web)
}

// 全文索引
func (s *Service) initFullTextSearch(web *restful.WebService) {
	utility := rest.NewRestUtility(rest.Config{
		ErrorIf:  s.Engine.CCErr,
		Language: s.Engine.Language,
	})

	utility.AddHandler(rest.Action{Verb: http.MethodPost, Path: "/find/full_text", Handler: s.FullTextSearch})

	utility.AddToRestfulWebService(web)
}

// 资源池目录
func (s *Service) initResourceDirectory(web *restful.WebService) {
	utility := rest.NewRestUtility(rest.Config{
		ErrorIf:  s.Engine.CCErr,
		Language: s.Engine.Language,
	})

	utility.AddHandler(rest.Action{Verb: http.MethodPost, Path: "/create/resource/directory", Handler: s.CreateResourceDirectory})
	utility.AddHandler(rest.Action{Verb: http.MethodPut, Path: "/update/resource/directory/{bk_module_id}", Handler: s.UpdateResourceDirectory})
	utility.AddHandler(rest.Action{Verb: http.MethodPost, Path: "/findmany/resource/directory", Handler: s.SearchResourceDirectory})
	utility.AddHandler(rest.Action{Verb: http.MethodDelete, Path: "/delete/resource/directory/{bk_module_id}", Handler: s.DeleteResourceDirectory})

	utility.AddToRestfulWebService(web)
}

func (s *Service) initService(web *restful.WebService) {
	s.initAssociation(web)
	s.initAuditLog(web)
	s.initBusiness(web)
	s.initInst(web)
	s.initModule(web)
	s.initSet(web)
	s.initObject(web)
	s.initObjectAttribute(web)
	s.initObjectGroup(web)
	s.initIdentifier(web)

	s.initBusinessObject(web)
	s.initBusinessClassification(web)
	s.initBusinessObjectAttribute(web)
	s.initBusinessObjectUnique(web)
	s.initBusinessObjectAttrGroup(web)
	s.initBusinessAssociation(web)
	s.initBusinessGraphics(web)
	s.initBusinessInst(web)

	s.initFullTextSearch(web)
	s.initSetTemplate(web)
	s.initInternalTask(web)

	s.initResourceDirectory(web)
}
