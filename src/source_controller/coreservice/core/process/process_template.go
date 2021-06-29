/*
/*
 * Tencent is pleased to support the open source community by making 蓝鲸 available.,
 * Copyright (C) 2017-2018 THL A29 Limited, a Tencent company. All rights reserved.
 * Licensed under the MIT License (the ",License",); you may not use this file except
 * in compliance with the License. You may obtain a copy of the License at
 * http://opensource.org/licenses/MIT
 * Unless required by applicable law or agreed to in writing, software distributed under
 * the License is distributed on an ",AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
 * either express or implied. See the License for the specific language governing permissions and
 * limitations under the License.
*/

package process

import (
	"time"

	"configcenter/src/common"
	"configcenter/src/common/blog"
	"configcenter/src/common/errors"
	"configcenter/src/common/http/rest"
	"configcenter/src/common/mapstr"
	"configcenter/src/common/metadata"
	"configcenter/src/storage/driver/mongodb"
)

func (p *processOperation) CreateProcessTemplate(kit *rest.Kit, template metadata.ProcessTemplate) (*metadata.ProcessTemplate, errors.CCErrorCoder) {
	// base attribute validate
	if field, err := template.Validate(); err != nil {
		blog.Errorf("CreateProcessTemplate failed, validation failed, code: %d, field: %s, err: %+v, rid: %s", common.CCErrCommParamsInvalid, field, err, kit.Rid)
		err := kit.CCError.CCErrorf(common.CCErrCommParamsInvalid, field)
		return nil, err
	}
	*template.Property.ProcessName.AsDefaultValue = true
	*template.Property.FuncName.AsDefaultValue = true
	if template.Property != nil && template.Property.ProcessName.Value != nil {
		template.ProcessName = *template.Property.ProcessName.Value
	}

	var bizID int64
	var err error
	if bizID, err = p.validateBizID(kit, template.BizID); err != nil {
		blog.Errorf("CreateProcessTemplate failed, validation failed, code: %d, err: %+v, rid: %s", common.CCErrCommParamsInvalid, err, kit.Rid)
		return nil, kit.CCError.CCErrorf(common.CCErrCommParamsInvalid, common.BKAppIDField)
	}

	template.BizID = bizID

	// validate service template id field
	serviceTemplate, err := p.GetServiceTemplate(kit, template.ServiceTemplateID)
	if err != nil {
		blog.Errorf("CreateProcessTemplate failed, template id invalid, code: %d, err: %+v, rid: %s", common.CCErrCommParamsInvalid, err, kit.Rid)
		return nil, kit.CCError.CCErrorf(common.CCErrCommParamsInvalid, "service_template_id")
	}

	// make sure biz id identical with service template
	if bizID != serviceTemplate.BizID {
		blog.Errorf("CreateProcessTemplate failed, validation failed, input bizID:%d not equal service template bizID:%d, rid: %s", bizID, serviceTemplate.BizID, kit.Rid)
		return nil, kit.CCError.CCErrorf(common.CCErrCommParamsInvalid, common.BKAppIDField)
	}

	if err := p.processNameUniqueValidate(kit, &template); err != nil {
		return nil, err
	}

	// generate id field
	id, err := mongodb.Client().NextSequence(kit.Ctx, common.BKTableNameProcessTemplate)
	if nil != err {
		blog.Errorf("CreateProcessTemplate failed, generate id failed, err: %+v, rid: %s", err, kit.Rid)
		return nil, kit.CCError.CCErrorf(common.CCErrCommGenerateRecordIDFailed)
	}

	template.ID = int64(id)

	template.Creator = kit.User
	template.Modifier = kit.User
	template.CreateTime = time.Now()
	template.LastTime = time.Now()
	template.SupplierAccount = kit.SupplierAccount

	if err := mongodb.Client().Table(common.BKTableNameProcessTemplate).Insert(kit.Ctx, &template); nil != err {
		blog.ErrorJSON("CreateProcessTemplate failed, mongodb failed, table: %s, template: %s, err: %s, rid: %s", common.BKTableNameProcessTemplate, template, err, kit.Rid)
		return nil, kit.CCError.CCErrorf(common.CCErrCommDBInsertFailed)
	}
	return &template, nil
}

func (p *processOperation) processNameUniqueValidate(kit *rest.Kit, template *metadata.ProcessTemplate) errors.CCErrorCoder {
	// process name unique
	processName := ""
	if template.Property.ProcessName.Value != nil {
		processName = *template.Property.ProcessName.Value
	}
	processNameFilter := map[string]interface{}{
		common.BKServiceTemplateIDField:  template.ServiceTemplateID,
		"property.bk_process_name.value": processName,
	}
	if template.ID != 0 {
		processNameFilter[common.BKFieldID] = map[string]interface{}{
			common.BKDBNE: template.ID,
		}
	}
	count, err := mongodb.Client().Table(common.BKTableNameProcessTemplate).Find(processNameFilter).Count(kit.Ctx)
	if err != nil {
		blog.Errorf("CreateProcessTemplate failed, check process_name unique failed, err: %+v, rid: %s", err, kit.Rid)
		return kit.CCError.CCErrorf(common.CCErrCommDBSelectFailed)
	}
	if count > 0 {
		return kit.CCError.CCErrorf(common.CCErrCoreServiceProcessNameDuplicated, processName)
	}
	return nil
}

func (p *processOperation) GetProcessTemplate(kit *rest.Kit, templateID int64) (*metadata.ProcessTemplate, errors.CCErrorCoder) {
	template := metadata.ProcessTemplate{}

	filter := map[string]int64{common.BKFieldID: templateID}
	if err := mongodb.Client().Table(common.BKTableNameProcessTemplate).Find(filter).One(kit.Ctx, &template); nil != err {
		blog.Errorf("GetProcessTemplate failed, mongodb failed, table: %s, filter: %+v, err: %+v, rid: %s", common.BKTableNameProcessTemplate, filter, err, kit.Rid)
		if mongodb.Client().IsNotFoundError(err) {
			return nil, kit.CCError.CCError(common.CCErrCommNotFound)
		}
		return nil, kit.CCError.CCErrorf(common.CCErrCommDBSelectFailed)
	}

	return &template, nil
}

func (p *processOperation) UpdateProcessTemplate(kit *rest.Kit, templateID int64, rawProperty map[string]interface{}) (*metadata.ProcessTemplate, errors.CCErrorCoder) {
	template, err := p.GetProcessTemplate(kit, templateID)
	if err != nil {
		return nil, err
	}

	property := metadata.ProcessProperty{}
	if err := mapstr.DecodeFromMapStr(&property, rawProperty); err != nil {
		blog.ErrorJSON("UpdateProcessTemplate failed, unmarshal failed, property: %s, err: %s, rid: %s", property, err, kit.Rid)
		err := kit.CCError.CCError(common.CCErrCommJSONUnmarshalFailed)
		return nil, err
	}

	// update fields to local object
	template.Property.Update(property, rawProperty)

	if field, err := template.Validate(); err != nil {
		blog.Errorf("UpdateProcessTemplate failed, validation failed, code: %d, err: %+v, rid: %s", common.CCErrCommParamsInvalid, err, kit.Rid)
		err := kit.CCError.CCErrorf(common.CCErrCommParamsInvalid, field)
		return nil, err
	}

	*template.Property.ProcessName.AsDefaultValue = true
	*template.Property.FuncName.AsDefaultValue = true
	template.Modifier = kit.User
	template.LastTime = time.Now()

	if err := p.processNameUniqueValidate(kit, template); err != nil {
		return nil, err
	}
	if template.Property != nil {
		if template.Property.ProcessName.Value != nil {
			template.ProcessName = *template.Property.ProcessName.Value
		}
	}

	// do update
	filter := map[string]int64{common.BKFieldID: templateID}
	if err := mongodb.Client().Table(common.BKTableNameProcessTemplate).Update(kit.Ctx, filter, &template); nil != err {
		blog.Errorf("UpdateProcessTemplate failed, mongodb failed, table: %s, filter: %+v, template: %+v, err: %+v, rid: %s", common.BKTableNameProcessTemplate, filter, template, err, kit.Rid)
		return nil, kit.CCError.CCErrorf(common.CCErrCommDBUpdateFailed)
	}
	return template, nil
}

func (p *processOperation) ListProcessTemplates(kit *rest.Kit, option metadata.ListProcessTemplatesOption) (*metadata.MultipleProcessTemplate, errors.CCErrorCoder) {
	filter := map[string]interface{}{
		common.BKAppIDField: option.BusinessID,
	}

	if len(option.ServiceTemplateIDs) != 0 {
		filter[common.BKServiceTemplateIDField] = map[string]interface{}{
			common.BKDBIN: option.ServiceTemplateIDs,
		}
	}

	if option.ProcessTemplateIDs != nil {
		filter[common.BKFieldID] = map[string][]int64{
			common.BKDBIN: option.ProcessTemplateIDs,
		}
	}

	var total uint64
	var err error
	if total, err = mongodb.Client().Table(common.BKTableNameProcessTemplate).Find(filter).Count(kit.Ctx); nil != err {
		blog.Errorf("ListProcessTemplates failed, mongodb failed, table: %s, filter: %+v, err: %+v, rid: %s", common.BKTableNameProcessTemplate, filter, err, kit.Rid)
		return nil, kit.CCError.CCErrorf(common.CCErrCommDBSelectFailed)
	}
	templates := make([]metadata.ProcessTemplate, 0)

	// ex: "-id,name"
	sort := "-id"
	if len(option.Page.Sort) > 0 {
		sort = option.Page.Sort
	}

	if err := mongodb.Client().Table(common.BKTableNameProcessTemplate).Find(filter).Start(uint64(option.Page.Start)).Limit(uint64(option.Page.Limit)).Sort(sort).All(kit.Ctx, &templates); nil != err {
		blog.Errorf("ListProcessTemplates failed, mongodb failed, table: %s, err: %+v, rid: %s", common.BKTableNameProcessTemplate, err, kit.Rid)
		return nil, kit.CCError.CCErrorf(common.CCErrCommDBSelectFailed)
	}

	result := &metadata.MultipleProcessTemplate{
		Count: total,
		Info:  templates,
	}
	return result, nil
}

func (p *processOperation) DeleteProcessTemplate(kit *rest.Kit, processTemplateID int64) errors.CCErrorCoder {
	template, err := p.GetProcessTemplate(kit, processTemplateID)
	if err != nil {
		blog.Errorf("DeleteProcessTemplate failed, GetProcessTemplate failed, templateID: %d, err: %+v, rid: %s", processTemplateID, err, kit.Rid)
		return err
	}

	updateFilter := map[string]int64{
		common.BKProcessTemplateIDField: template.ID,
	}
	updateDoc := map[string]interface{}{
		common.BKProcessTemplateIDField: common.ServiceTemplateIDNotSet,
	}
	e := mongodb.Client().Table(common.BKTableNameProcessInstanceRelation).Update(kit.Ctx, updateFilter, updateDoc)
	if nil != e {
		blog.Errorf("DeleteProcessTemplate failed, clear process instance templateID field failed, table: %s, filter: %+v, err: %+v, rid: %s", common.BKTableNameProcessInstanceRelation, updateFilter, e, kit.Rid)
		return kit.CCError.CCErrorf(common.CCErrCommDBSelectFailed)
	}

	deleteFilter := map[string]int64{common.BKFieldID: template.ID}
	if err := mongodb.Client().Table(common.BKTableNameProcessTemplate).Delete(kit.Ctx, deleteFilter); nil != err {
		blog.Errorf("DeleteProcessTemplate failed, mongodb failed, table: %s, filter: %+v, err: %+v, rid: %s", common.BKTableNameProcessTemplate, deleteFilter, err, kit.Rid)
		return kit.CCError.CCErrorf(common.CCErrCommDBSelectFailed)
	}
	return nil
}
