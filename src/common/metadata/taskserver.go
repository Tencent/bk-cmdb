/*
 * Tencent is pleased to support the open source community by making 蓝鲸 available.,
 * Copyright (C) 2017,-2018 THL A29 Limited, a Tencent company. All rights reserved.
 * Licensed under the MIT License (the ",License",); you may not use this file except
 * in compliance with the License. You may obtain a copy of the License at
 * http://opensource.org/licenses/MIT
 * Unless required by applicable law or agreed to in writing, software distributed under
 * the License is distributed on an ",AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
 * either express or implied. See the License for the specific language governing permissions and
 * limitations under the License.
 */

package metadata

import (
	"net/http"
	"time"

	"configcenter/src/common/mapstr"
)

// CreateTaskRequest create task request parameters
type CreateTaskRequest struct {
	// task name
	Name string `json:"name"`

	// flag task 任务标识，留给业务方做识别任务
	Flag string `json:"flag"`

	// bk_inst_id 实例id，该任务关联的实例id
	InstID int64 `json:"bk_inst_id"`

	Data []interface{} `json:"data"`
}

// APITaskDetail task info detail
type APITaskDetail struct {
	// task id
	TaskID string `json:"task_id" bson:"task_id"`
	// task name. 表示所在的任务队列
	Name string `json:"name" bson:"name"`
	// flag task 任务标识，留给业务方做识别任务
	Flag string `json:"flag" bson:"flag"`
	// bk_inst_id 实例id，该任务关联的实例id
	InstID int64 `json:"bk_inst_id" bson:"bk_inst_id"`
	// task create user
	User string `json:"user" bson:"user"`
	//  http header
	Header http.Header `json:"header" bson:"header"`
	// task status
	Status APITaskStatus `json:"status" bson:"status"`
	// sub task detail
	Detail []APISubTaskDetail `json:"detail" bson:"detail"`

	CreateTime time.Time `json:"create_time" bson:"create_time"`
	LastTime   time.Time `json:"last_time" bson:"last_time"`
}

// APISubTaskDetail task data and execute detail
type APISubTaskDetail struct {
	SubTaskID string        `json:"sub_task_id" bson:"sub_task_id"`
	Data      interface{}   `json:"data" bson:"data"`
	Status    APITaskStatus `json:"status" bson:"status"`
	Response  *Response     `json:"response" bson:"response"`
}

// APITaskStatus task status type
type APITaskStatus int64

func (s APITaskStatus) IsFinished() bool {
	if s == 200 || s == 500 {
		return true
	}
	return false
}
func (s APITaskStatus) IsSuccessful() bool {
	if s == 200 {
		return true
	}
	return false
}

func (s APITaskStatus) IsFailure() bool {
	if s == 500 {
		return true
	}
	return false
}

const (
	// APITaskStatusNew new task ,waiting execute
	APITaskStatusNew APITaskStatus = 0
	// APITaskStatusWaitExecute 正在执行的任务中断了。 补偿后。确定需要重新执行
	APITaskStatusWaitExecute APITaskStatus = 1

	// APITaskStatuExecute task executing
	APITaskStatuExecute APITaskStatus = 100

	// APITaskStatusSuccess task execute success
	APITaskStatusSuccess APITaskStatus = 200

	// APITAskStatusFail task execute failure
	APITAskStatusFail APITaskStatus = 500
)

type ListAPITaskRequest struct {
	Condition mapstr.MapStr `json:"condition"`
	Page      BasePage      `json:"page"`
}

type ListAPITaskLatestRequest struct {
	Condition mapstr.MapStr `json:"condition"`
	Fields    []string      `json:"fields"`
}

type ListAPITaskLatestResponse struct {
	BaseResp
	Data []APITaskDetail `json:"data"`
}

type ListAPITaskData struct {
	Info  []APITaskDetail `json:"info"`
	Count int64           `json:"count"`
	Page  BasePage        `json:"page"`
}

type ListAPITaskResponse struct {
	BaseResp
	Data ListAPITaskData `json:"data"`
}

type CreateTaskResponse struct {
	BaseResp
	Data APITaskDetail `json:"data"`
}

type TaskDetailResponse struct {
	BaseResp
	Data struct {
		Info APITaskDetail `json:"info"`
	} `json:"data"`
}

type ListAPITaskDetail struct {
	SetID  []int64  `json:"bk_set_id"`
	Fields []string `json:"fields"`
}
