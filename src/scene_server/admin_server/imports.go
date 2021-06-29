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

package main

import (
	_ "configcenter/src/scene_server/admin_server/upgrader/v3.0.8"
	_ "configcenter/src/scene_server/admin_server/upgrader/v3.0.9-beta.1"
	_ "configcenter/src/scene_server/admin_server/upgrader/v3.0.9-beta.3"
	_ "configcenter/src/scene_server/admin_server/upgrader/v3.1.0-alpha.2"
	_ "configcenter/src/scene_server/admin_server/upgrader/x08.09.04.01"
	_ "configcenter/src/scene_server/admin_server/upgrader/x08.09.17.01"
	_ "configcenter/src/scene_server/admin_server/upgrader/x08.09.18.01"
	_ "configcenter/src/scene_server/admin_server/upgrader/x08.09.26.01"
	_ "configcenter/src/scene_server/admin_server/upgrader/x18.09.30.01"

	// 3.2.x
	_ "configcenter/src/scene_server/admin_server/upgrader/x18.10.10.01"
	_ "configcenter/src/scene_server/admin_server/upgrader/x18.10.30.01"
	_ "configcenter/src/scene_server/admin_server/upgrader/x18.11.19.01"
	_ "configcenter/src/scene_server/admin_server/upgrader/x18.12.12.01"
	_ "configcenter/src/scene_server/admin_server/upgrader/x18.12.12.02"
	_ "configcenter/src/scene_server/admin_server/upgrader/x18.12.12.03"
	_ "configcenter/src/scene_server/admin_server/upgrader/x18.12.12.04"
	_ "configcenter/src/scene_server/admin_server/upgrader/x18.12.12.05"
	_ "configcenter/src/scene_server/admin_server/upgrader/x18.12.12.06"
	_ "configcenter/src/scene_server/admin_server/upgrader/x18.12.13.01"
	_ "configcenter/src/scene_server/admin_server/upgrader/x19.01.18.01"
	_ "configcenter/src/scene_server/admin_server/upgrader/x19.02.15.10"

	// 3.4.x
	_ "configcenter/src/scene_server/admin_server/upgrader/x19.04.16.01"
	_ "configcenter/src/scene_server/admin_server/upgrader/x19.04.16.02"
	_ "configcenter/src/scene_server/admin_server/upgrader/x19.04.16.03"
	_ "configcenter/src/scene_server/admin_server/upgrader/x19.05.16.01"
	_ "configcenter/src/scene_server/admin_server/upgrader/x19.08.19.01"

	// v3.5.x
	_ "configcenter/src/scene_server/admin_server/upgrader/x19.08.20.01"
	_ "configcenter/src/scene_server/admin_server/upgrader/x19.08.26.02"
	_ "configcenter/src/scene_server/admin_server/upgrader/x19.09.03.01"
	_ "configcenter/src/scene_server/admin_server/upgrader/x19.09.03.02"
	_ "configcenter/src/scene_server/admin_server/upgrader/x19.09.03.03"
	_ "configcenter/src/scene_server/admin_server/upgrader/x19.09.03.04"
	_ "configcenter/src/scene_server/admin_server/upgrader/x19.09.03.05"
	_ "configcenter/src/scene_server/admin_server/upgrader/x19.09.03.06"
	_ "configcenter/src/scene_server/admin_server/upgrader/x19.09.03.07"
	_ "configcenter/src/scene_server/admin_server/upgrader/x19.09.03.08"
	_ "configcenter/src/scene_server/admin_server/upgrader/x19.10.22.01"
	_ "configcenter/src/scene_server/admin_server/upgrader/x19.10.22.02"
	_ "configcenter/src/scene_server/admin_server/upgrader/x19.10.22.03"
	_ "configcenter/src/scene_server/admin_server/upgrader/x20.01.13.01"
	_ "configcenter/src/scene_server/admin_server/upgrader/x20.02.17.01"

	// v3.6.x
	_ "configcenter/src/scene_server/admin_server/upgrader/y3.6.201909062359"
	_ "configcenter/src/scene_server/admin_server/upgrader/y3.6.201909272359"
	_ "configcenter/src/scene_server/admin_server/upgrader/y3.6.201910091234"
	_ "configcenter/src/scene_server/admin_server/upgrader/y3.6.201911121930"
	_ "configcenter/src/scene_server/admin_server/upgrader/y3.6.201911122106"
	_ "configcenter/src/scene_server/admin_server/upgrader/y3.6.201911141015"
	_ "configcenter/src/scene_server/admin_server/upgrader/y3.6.201911141516"
	_ "configcenter/src/scene_server/admin_server/upgrader/y3.6.201911261109"
	_ "configcenter/src/scene_server/admin_server/upgrader/y3.6.201912241627"

	// v3.7.x
	_ "configcenter/src/scene_server/admin_server/upgrader/y3.7.201911141719"
	_ "configcenter/src/scene_server/admin_server/upgrader/y3.7.201912121117"
	_ "configcenter/src/scene_server/admin_server/upgrader/y3.7.201912171427"
	_ "configcenter/src/scene_server/admin_server/upgrader/y3.7.202002231026"

	// v3.8.x
	_ "configcenter/src/scene_server/admin_server/upgrader/y3.8.202001172032"
	_ "configcenter/src/scene_server/admin_server/upgrader/y3.8.202002101113"
	_ "configcenter/src/scene_server/admin_server/upgrader/y3.8.202004141131"
	_ "configcenter/src/scene_server/admin_server/upgrader/y3.8.202004151435"
	_ "configcenter/src/scene_server/admin_server/upgrader/y3.8.202004241035"
	_ "configcenter/src/scene_server/admin_server/upgrader/y3.8.202004291536"
	_ "configcenter/src/scene_server/admin_server/upgrader/y3.8.202006021120"
	_ "configcenter/src/scene_server/admin_server/upgrader/y3.8.202006092135"
	_ "configcenter/src/scene_server/admin_server/upgrader/y3.8.202006231730"
	_ "configcenter/src/scene_server/admin_server/upgrader/y3.8.202006241144"
	_ "configcenter/src/scene_server/admin_server/upgrader/y3.8.202006281530"
	_ "configcenter/src/scene_server/admin_server/upgrader/y3.8.202007011748"
	_ "configcenter/src/scene_server/admin_server/upgrader/y3.8.202008051650"
	_ "configcenter/src/scene_server/admin_server/upgrader/y3.8.202008111026"
	_ "configcenter/src/scene_server/admin_server/upgrader/y3.8.202008241747"
	_ "configcenter/src/scene_server/admin_server/upgrader/y3.8.202009101702"

	// v3.9.x
	_ "configcenter/src/scene_server/admin_server/upgrader/y3.9.202002131522"
	_ "configcenter/src/scene_server/admin_server/upgrader/y3.9.202008101530"
	_ "configcenter/src/scene_server/admin_server/upgrader/y3.9.202008121631"
	_ "configcenter/src/scene_server/admin_server/upgrader/y3.9.202008172134"
	_ "configcenter/src/scene_server/admin_server/upgrader/y3.9.202010131456"
	_ "configcenter/src/scene_server/admin_server/upgrader/y3.9.202010151455"
	_ "configcenter/src/scene_server/admin_server/upgrader/y3.9.202010151650"
	_ "configcenter/src/scene_server/admin_server/upgrader/y3.9.202010211805"
	_ "configcenter/src/scene_server/admin_server/upgrader/y3.9.202010281615"
	_ "configcenter/src/scene_server/admin_server/upgrader/y3.9.202011021415"
	_ "configcenter/src/scene_server/admin_server/upgrader/y3.9.202011021501"
	_ "configcenter/src/scene_server/admin_server/upgrader/y3.9.202011171550"
	_ "configcenter/src/scene_server/admin_server/upgrader/y3.9.202011172152"
	_ "configcenter/src/scene_server/admin_server/upgrader/y3.9.202011192014"
	_ "configcenter/src/scene_server/admin_server/upgrader/y3.9.202011201146"
	_ "configcenter/src/scene_server/admin_server/upgrader/y3.9.202011241510"
	_ "configcenter/src/scene_server/admin_server/upgrader/y3.9.202011251014"
	_ "configcenter/src/scene_server/admin_server/upgrader/y3.9.202011301723"
	_ "configcenter/src/scene_server/admin_server/upgrader/y3.9.202012011450"
	_ "configcenter/src/scene_server/admin_server/upgrader/y3.9.202101061721"
	_ "configcenter/src/scene_server/admin_server/upgrader/y3.9.202102011055"
	_ "configcenter/src/scene_server/admin_server/upgrader/y3.9.202102261105"
	_ "configcenter/src/scene_server/admin_server/upgrader/y3.9.202103031533"
	_ "configcenter/src/scene_server/admin_server/upgrader/y3.9.202103231621"
	_ "configcenter/src/scene_server/admin_server/upgrader/y3.9.202104011012"
	_ "configcenter/src/scene_server/admin_server/upgrader/y3.9.202104211151"
	_ "configcenter/src/scene_server/admin_server/upgrader/y3.9.202105261459"
	_ "configcenter/src/scene_server/admin_server/upgrader/y3.9.202106031151"
	_ "configcenter/src/scene_server/admin_server/upgrader/y3.9.202106101134"
)
