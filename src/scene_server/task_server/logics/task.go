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

package logics

import (
	"context"
	"net/http"
	"strings"
	"time"

	"configcenter/src/common"
	"configcenter/src/common/blog"
	"configcenter/src/common/mapstr"
	"configcenter/src/common/metadata"

	"github.com/rs/xid"
)

// Create add task
func (lgc *Logics) Create(ctx context.Context, input *metadata.CreateTaskRequest) (metadata.APITaskDetail, error) {

	dbTask := metadata.APITaskDetail{}
	input.Name = strings.TrimSpace(input.Name)

	if input.Name == "" {
		return dbTask, lgc.ccErr.Errorf(common.CCErrCommParamsNeedString, "name")
	}

	if len(input.Data) == 0 {
		return dbTask, lgc.ccErr.Errorf(common.CCErrCommParamsNeedString, "data")
	}

	dbTask.TaskID = getStrTaskID("id")
	dbTask.Name = input.Name
	dbTask.User = lgc.user
	dbTask.Flag = input.Flag
	dbTask.InstID = input.InstID
	dbTask.Header = GetDBHTTPHeader(lgc.header)
	dbTask.Status = metadata.APITaskStatusNew
	dbTask.CreateTime = time.Now()
	dbTask.LastTime = time.Now()
	for _, taskItem := range input.Data {
		dbTask.Detail = append(dbTask.Detail, metadata.APISubTaskDetail{
			SubTaskID: getStrTaskID("sid"),
			Data:      taskItem,
			Status:    metadata.APITaskStatusNew,
		})
	}
	err := lgc.db.Table(common.BKTableNameAPITask).Insert(ctx, dbTask)
	if err != nil {
		blog.ErrorJSON("create task table:%s, data:%s, err:%s, rid:%s", common.BKTableNameAPITask, dbTask, err.Error(), lgc.rid)
		return dbTask, lgc.ccErr.Error(common.CCErrCommDBInsertFailed)
	}
	return dbTask, nil
}

// List list task
func (lgc *Logics) List(ctx context.Context, name string, input *metadata.ListAPITaskRequest) ([]metadata.APITaskDetail, uint64, error) {
	if input == nil {
		input = &metadata.ListAPITaskRequest{}
	}
	if input.Condition == nil {
		input.Condition = mapstr.New()
	}
	input.Condition.Set("name", name)
	if input.Page.IsIllegal() {
		return nil, 0, lgc.ccErr.Errorf(common.CCErrCommPageLimitIsExceeded)
	}
	cnt, err := lgc.db.Table(common.BKTableNameAPITask).Find(input.Condition).Count(ctx)
	if err != nil {
		blog.ErrorJSON("list task table:%s, input:%s, err:%s, rid:%s", common.BKTableNameAPITask, input, err.Error(), lgc.rid)
		return nil, 0, lgc.ccErr.Error(common.CCErrCommDBSelectFailed)
	}

	rows := make([]metadata.APITaskDetail, 0)
	err = lgc.db.Table(common.BKTableNameAPITask).Find(input.Condition).
		Start(uint64(input.Page.Start)).Limit(uint64(input.Page.Limit)).
		Sort(input.Page.Sort).All(ctx, &rows)

	return rows, cnt, nil
}

// ListLatestTask list latest task
func (lgc *Logics) ListLatestTask(ctx context.Context, name string,
	input *metadata.ListAPITaskLatestRequest) ([]metadata.APITaskDetail, error) {
	/*
		aggregateCond parameter of aggregate to search the latest created task in input.Condition need.
		because multiple results of the same task may be at the front end of sorting by
		create_time field, use group to get the first result of each task
	*/
	aggregateCond := []map[string]interface{}{
		{common.BKDBSort: map[string]interface{}{common.CreateTimeField: -1}},
		{common.BKDBGroup: map[string]interface{}{
			"_id": "$bk_inst_id",
			"doc": map[string]interface{}{"$first": "$$ROOT"},
		}},
		{common.BKDBReplaceRoot: map[string]interface{}{"newRoot": "$doc"}},
	}

	if input == nil {
		input = &metadata.ListAPITaskLatestRequest{}
	}

	if input.Condition == nil {
		input.Condition = mapstr.New()
	}

	if len(name) != 0 {
		input.Condition.Set("name", name)
	}

	if len(input.Condition) != 0 {
		aggregateCond = append([]map[string]interface{}{{common.BKDBMatch: input.Condition}}, aggregateCond...)
	}

	if len(input.Fields) != 0 {
		cond := map[string]int64{}
		for _, field := range input.Fields {
			cond[field] = 1
		}
		aggregateCond = append(aggregateCond, map[string]interface{}{
			common.BKDBProject: cond,
		})
	}

	result := make([]metadata.APITaskDetail, 0)
	if err := lgc.db.Table(common.BKTableNameAPITask).AggregateAll(ctx, aggregateCond, &result); err != nil {
		blog.Errorf("list latest task failed, aggregateCond: %v, err: %v, rid: %v", aggregateCond, err, lgc.rid)
		return nil, err
	}

	return result, nil
}

// Detail  task detail
func (lgc *Logics) Detail(ctx context.Context, taskID string) (*metadata.APITaskDetail, error) {

	condition := mapstr.New()
	condition.Set("task_id", taskID)

	rows := make([]metadata.APITaskDetail, 0)
	err := lgc.db.Table(common.BKTableNameAPITask).Find(condition).All(ctx, &rows)
	if err != nil {
		blog.ErrorJSON("detail task table:%s, input:%s, err:%s, rid:%s", common.BKTableNameAPITask, condition, err.Error(), lgc.rid)
		return nil, lgc.ccErr.Error(common.CCErrCommDBSelectFailed)
	}
	if len(rows) == 0 {
		return nil, nil
	}
	return &rows[0], nil
}

func (lgc *Logics) DeleteTask(ctx context.Context, taskCond *metadata.DeleteOption) error {
	if len(taskCond.Condition) == 0 {
		blog.Errorf("task condition is empty, rid: %s", lgc.rid)
		return lgc.ccErr.CCErrorf(common.CCErrCommInstDataNil, "task condition")
	}

	err := lgc.db.Table(common.BKTableNameAPITask).Delete(ctx, taskCond.Condition)
	if err != nil {
		blog.Errorf("delete task failed, err: %s, rid: %s", err.Error(), lgc.rid)
		return err
	}

	return nil
}

// ChangeStatusToSuccess task status change to success
func (lgc *Logics) ChangeStatusToSuccess(ctx context.Context, taskID, subTaskID string) error {

	if taskID == "" {
		return lgc.ccErr.CCErrorf(common.CCErrCommParamsNeedSet, "task_id")
	}
	if subTaskID == "" {
		return lgc.ccErr.CCErrorf(common.CCErrCommParamsNeedSet, "sub_task_id")
	}
	return lgc.changeStatus(ctx, taskID, subTaskID, metadata.APITaskStatusSuccess, nil)
}

// ChangeStatusToFailure change status to failure
func (lgc *Logics) ChangeStatusToFailure(ctx context.Context, taskID, subTaskID string, errResponse *metadata.Response) error {

	if taskID == "" {
		return lgc.ccErr.CCErrorf(common.CCErrCommParamsNeedSet, "task_id")
	}
	if subTaskID == "" {
		return lgc.ccErr.CCErrorf(common.CCErrCommParamsNeedSet, "sub_task_id")
	}
	if errResponse == nil || errResponse.Code == 0 {
		return lgc.ccErr.CCError(common.CCErrTaskErrResponseEmtpyFail)
	}
	return lgc.changeStatus(ctx, taskID, subTaskID, metadata.APITAskStatusFail, errResponse)
}

func (lgc *Logics) changeStatus(ctx context.Context, taskID, subTaskID string, status metadata.APITaskStatus, errResponse *metadata.Response) error {
	condition := mapstr.New()
	condition.Set("task_id", taskID)

	rows := make([]metadata.APITaskDetail, 0)
	err := lgc.db.Table(common.BKTableNameAPITask).Find(condition).All(ctx, &rows)
	if err != nil {
		blog.ErrorJSON("change task status, table:%s, input:%s, err:%s, rid:%s", common.BKTableNameAPITask, condition, err.Error(), lgc.rid)
		return lgc.ccErr.Error(common.CCErrCommDBSelectFailed)
	}
	if len(rows) == 0 {
		blog.ErrorJSON("change task status, table:%s, input:%s, task not found, rid:%s", common.BKTableNameAPITask, condition, err.Error(), lgc.rid)
		return lgc.ccErr.CCError(common.CCErrTaskNotFound)
	}

	existSubTask := false
	canChangeStatus := false
	for _, subTask := range rows[0].Detail {
		if subTask.SubTaskID == subTaskID {
			existSubTask = true
			if (subTask.Status == metadata.APITaskStatusNew || subTask.Status == metadata.APITaskStatuExecute) && subTask.Status != status {
				canChangeStatus = true
			}
			break
		}
	}
	if !existSubTask {
		return lgc.ccErr.CCError(common.CCErrTaskSubTaskNotFound)
	}
	if !canChangeStatus {
		return lgc.ccErr.CCError(common.CCErrTaskStatusNotAllowChangeTo)
	}

	updateConditon := mapstr.New()
	updateConditon.Set("task_id", taskID)
	updateConditon.Set("detail.sub_task_id", subTaskID)
	updateData := mapstr.New()
	updateData.Set("detail.$.status", status)
	updateData.Set(common.LastTimeField, time.Now())
	if status == metadata.APITAskStatusFail {
		// 任务的一个子任务失败，则任务失败
		updateData.Set("status", status)
		updateData.Set("detail.$.response", errResponse)
	}
	err = lgc.db.Table(common.BKTableNameAPITask).Update(ctx, updateConditon, updateData)
	if err != nil {
		blog.ErrorJSON("change task status, table:%s, input:%s, err:%s, rid:%s", common.BKTableNameAPITask, condition, err.Error(), lgc.rid)
		return lgc.ccErr.Error(common.CCErrCommDBSelectFailed)
	}

	// 如果修改为执行成功。 判断是否所有的子项都成功。如果都成功，则任务完成
	if status == metadata.APITaskStatusSuccess {
		rows := make([]metadata.APITaskDetail, 0)
		err := lgc.db.Table(common.BKTableNameAPITask).Find(condition).All(ctx, &rows)
		if err != nil {
			blog.ErrorJSON("change task status, table:%s, input:%s, err:%s, rid:%s", common.BKTableNameAPITask, condition, err.Error(), lgc.rid)
			return lgc.ccErr.Error(common.CCErrCommDBSelectFailed)
		}
		allSuccess := true
		for _, subTask := range rows[0].Detail {

			if subTask.Status != metadata.APITaskStatusSuccess {
				canChangeStatus = false
				break
			}
		}
		if allSuccess {
			updateConditon := mapstr.New()
			updateConditon.Set("task_id", taskID)
			updateData := mapstr.New()
			updateData.Set("status", metadata.APITaskStatusSuccess)
			updateData.Set(common.LastTimeField, time.Now())

			err = lgc.db.Table(common.BKTableNameAPITask).Update(ctx, updateConditon, updateData)
			if err != nil {
				blog.ErrorJSON("change task status, table:%s, input:%s, err:%s, rid:%s", common.BKTableNameAPITask, condition, err.Error(), lgc.rid)
				return lgc.ccErr.Error(common.CCErrCommDBSelectFailed)
			}
		}
	}
	return nil
}

func GetDBHTTPHeader(header http.Header) http.Header {

	newHeader := make(http.Header, 0)
	newHeader.Add(common.BKHTTPCCRequestID, header.Get(common.BKHTTPCCRequestID))
	newHeader.Add(common.BKHTTPCookieLanugageKey, header.Get(common.BKHTTPCookieLanugageKey))
	newHeader.Add(common.BKHTTPHeaderUser, header.Get(common.BKHTTPHeaderUser))
	newHeader.Add(common.BKHTTPLanguage, header.Get(common.BKHTTPLanguage))
	newHeader.Add(common.BKHTTPOwner, header.Get(common.BKHTTPOwner))
	newHeader.Add(common.BKHTTPOwnerID, header.Get(common.BKHTTPOwnerID))
	newHeader.Add(common.BKHTTPRequestAppCode, header.Get(common.BKHTTPRequestAppCode))
	newHeader.Add(common.BKHTTPRequestRealIP, header.Get(common.BKHTTPRequestRealIP))

	return newHeader
}

func getStrTaskID(prefix string) string {
	prefix = strings.TrimSpace(prefix)
	if prefix != "" {
		prefix = prefix + ":"
	}
	return "task:" + prefix + xid.New().String()
}
