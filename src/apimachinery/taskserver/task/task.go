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

package task

import (
	"context"
	"net/http"

	"configcenter/src/common/blog"
	"configcenter/src/common/errors"
	"configcenter/src/common/metadata"
)

// Create  新加任务， name 任务名，flag:任务标识，留给业务方做识别任务，instID:任务的执行源实例id，data 每一项任务需要的参数
func (t *task) Create(ctx context.Context, header http.Header, name, flag string, instID int64, data []interface{}) (resp *metadata.CreateTaskResponse, err error) {
	resp = new(metadata.CreateTaskResponse)
	subPath := "/task/create"
	body := metadata.CreateTaskRequest{
		Name:   name,
		Flag:   flag,
		InstID: instID,
		Data:   data,
	}

	err = t.client.Post().
		WithContext(ctx).
		Body(body).
		SubResourcef(subPath).
		WithHeaders(header).
		Do().
		Into(resp)
	return
}

func (t *task) ListTask(ctx context.Context, header http.Header, name string, data *metadata.ListAPITaskRequest) (resp *metadata.ListAPITaskResponse, err error) {
	resp = new(metadata.ListAPITaskResponse)
	subPath := "/task/findmany/list/%s"

	err = t.client.Post().
		WithContext(ctx).
		Body(data).
		SubResourcef(subPath, name).
		WithHeaders(header).
		Do().
		Into(resp)
	return
}

func (t *task) ListLatestTask(ctx context.Context, header http.Header, name string, data *metadata.ListAPITaskLatestRequest) (resp *metadata.ListAPITaskLatestResponse, err error) {
	resp = new(metadata.ListAPITaskLatestResponse)
	subPath := "/task/findmany/list/latest/%s"

	err = t.client.Post().
		WithContext(ctx).
		Body(data).
		SubResourcef(subPath, name).
		WithHeaders(header).
		Do().
		Into(resp)
	return
}

func (t *task) TaskDetail(ctx context.Context, header http.Header, taskID string) (resp *metadata.TaskDetailResponse, err error) {
	resp = new(metadata.TaskDetailResponse)
	subPath := "/task/findone/detail/%s"

	err = t.client.Post().
		WithContext(ctx).
		Body(nil).
		SubResourcef(subPath, taskID).
		WithHeaders(header).
		Do().
		Into(resp)
	return
}

func (t *task) DeleteTask(ctx context.Context, header http.Header, taskCond *metadata.DeleteOption) error {
	resp := new(metadata.Response)
	subPath := "/task/deletemany"

	err := t.client.Post().
		WithContext(ctx).
		Body(taskCond).
		SubResourcef(subPath).
		WithHeaders(header).
		Do().
		Into(resp)

	if err != nil {
		blog.Errorf("delete task failed, http request failed, err: %+v", err)
		return errors.CCHttpError
	}
	if resp.CCError() != nil {
		return resp.CCError()
	}

	return nil
}

func (t *task) TaskStatusToSuccess(ctx context.Context, header http.Header, taskID, subTaskID string) (resp *metadata.Response, err error) {
	resp = new(metadata.Response)
	subPath := "/task/set/status/sucess/id/%s/sub_id/%s"

	err = t.client.Put().
		WithContext(ctx).
		Body(nil).
		SubResourcef(subPath, taskID, subTaskID).
		WithHeaders(header).
		Do().
		Into(resp)
	return
}

func (t *task) TaskStatusToFailure(ctx context.Context, header http.Header, taskID, subTaskID string, errResponse *metadata.Response) (resp *metadata.Response, err error) {
	resp = new(metadata.Response)
	subPath := "/task/set/status/failure/id/%s/sub_id/%s"

	err = t.client.Put().
		WithContext(ctx).
		Body(errResponse).
		SubResourcef(subPath, taskID, subTaskID).
		WithHeaders(header).
		Do().
		Into(resp)
	return
}

/*


 http.MethodPost, Path: "/task/create", Handler: s.CreateTask})
 http.MethodPost, Path: "/task/findmany/list/{name}", Handler: s.ListTask})
 http.MethodPost, Path: "/task/findone/detail/{task_id}", Handler: s.DetailTask})
 http.MethodPut, Path: "/task/set/status/sucess/id/{task_id}/sub_id/{sub_task_id}", Handler: s.StatusToSuccess})
 http.MethodPut, Path: "/task/set/status/failure/id/{task_id}/sub_id/{sub_task_id}", Handler: s.StatusToFailure})

*/
