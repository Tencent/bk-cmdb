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

package types

// request id key, travel in context.
const (
	RequestIDKey       = "rid"
	RequestIDHeaderKey = "X-Request-Id"
)

const (
	// the key to describe the auth path that this resource need to auth.
	// only if the path is matched one of the use's auth policy, then a use's
	// have this resource's operate authorize.
	IamPathKey = "_bk_iam_path_"
	IamIDKey   = "id"
)
