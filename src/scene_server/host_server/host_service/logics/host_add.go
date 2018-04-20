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
	"configcenter/src/common"
	"configcenter/src/common/auditoplog"
	"configcenter/src/common/blog"
	errorHandle "configcenter/src/common/errors"
	"configcenter/src/common/util"
	sencecommon "configcenter/src/scene_server/common"
	"configcenter/src/scene_server/validator"
	sourceAuditAPI "configcenter/src/source_controller/api/auditlog"
	"errors"
	"fmt"

	restful "github.com/emicklei/go-restful"

	"time"
)

//AddHost, return error info
func AddHost(req *restful.Request, ownerID string, appID int, hostInfos map[int]map[string]interface{}, moduleID int, hostAddr, ObjAddr, auditAddr string, errHandle errorHandle.DefaultCCErrorIf) (error, []string, []string, []string) {

	user := sencecommon.GetUserFromHeader(req)

	addHostURL := hostAddr + "/host/v1/insts/"
	uHostURL := ObjAddr + "/object/v1/insts/host"

	addParams := make(map[string]interface{})
	addParams[common.BKAppIDField] = appID
	addParams[common.BKModuleIDField] = []int{moduleID}
	addModulesURL := hostAddr + "/host/v1/meta/hosts/modules/"

	allHostList, err := GetHostInfoByConds(req, hostAddr, nil)
	if nil != err {
		return errors.New("查询主机信息失败"), nil, nil, nil
	}

	hostMap := convertHostInfo(allHostList)
	input := make(map[string]interface{}, 2)     //更新主机数据
	condInput := make(map[string]interface{}, 1) //更新主机条件
	var errMsg, succMsg, updateErrMsg []string   //新加错误， 成功，  更新失败
	iSubArea := common.BKDefaultDirSubArea

	defaultFields := getHostFields(ownerID, ObjAddr)
	ts := time.Now().UTC()
	//operator log
	var logConents []auditoplog.AuditLogExt
	hostLogFields, _ := GetHostLogFields(req, ownerID, ObjAddr)
	for index, host := range hostInfos {
		if nil == host {
			continue
		}

		innerIP, ok := host[common.BKHostInnerIPField].(string)
		if ok == false || "" == innerIP {
			errMsg = append(errMsg, fmt.Sprintf("%d行内网ip为空", index))
			continue
		}
		notExistFields := []string{} //没有赋值的key，不需要校验
		for key, value := range defaultFields {
			_, ok := host[key]
			if ok {
				//已经存在，
				continue
			}
			require, _ := util.GetIntByInterface(value["require"])
			if require == common.BKTrue {
				errMsg = append(errMsg, fmt.Sprintf("%d行内网%s必填", index, key))
				continue
			}
			notExistFields = append(notExistFields, key)
		}
		blog.Infof("no validate fields %v", notExistFields)

		valid := validator.NewValidMapWithKeyFileds(common.BKDefaultOwnerID, common.BKInnerObjIDHost, ObjAddr, notExistFields, errHandle)

		key := fmt.Sprintf("%s-%v", innerIP, iSubArea)
		iHost, ok := hostMap[key]
		//生产日志
		if ok {
			delete(host, common.BKCloudIDField)
			delete(host, "import_from")
			delete(host, common.CreateTimeField)
			hostInfo := iHost.(map[string]interface{})

			hostID, _ := util.GetIntByInterface(hostInfo[common.BKHostIDField])
			_, err = valid.ValidMap(host, common.ValidUpdate, hostID)
			if nil != err {
				updateErrMsg = append(updateErrMsg, fmt.Sprintf("%d行%v", index, err))
				continue
			}
			//prepare the log
			strHostID := fmt.Sprintf("%d", hostID)
			logObj := NewHostLog(req, common.BKDefaultOwnerID, strHostID, hostAddr, ObjAddr, hostLogFields)

			condInput[common.BKHostIDField] = hostID
			input["condition"] = condInput
			input["data"] = host
			isSuccess, message, _ := GetHttpResult(req, uHostURL, common.HTTPUpdate, input)
			innerIP := host[common.BKHostInnerIPField].(string)
			if !isSuccess {
				ret := fmt.Sprintf("%s更新失败%v;", innerIP, message)
				updateErrMsg = append(updateErrMsg, fmt.Sprintf("%d行%v", index, ret))
				continue
			}
			logContent, _ := logObj.GetHostLog(strHostID, false)
			logConents = append(logConents, auditoplog.AuditLogExt{ID: hostID, Content: logContent, ExtKey: innerIP})

		} else {
			host[common.BKCloudIDField] = iSubArea
			host[common.CreateTimeField] = ts
			//补充未填写字段的默认值
			for key, val := range defaultFields {
				_, ok := host[key]
				if !ok {
					host[key] = val["default"]
				}
			}
			_, err := valid.ValidMap(host, common.ValidCreate, 0)

			if nil != err {
				errMsg = append(errMsg, fmt.Sprintf("%d行%v", index, err))
				continue
			}

			//prepare the log
			logObj := NewHostLog(req, common.BKDefaultOwnerID, "", hostAddr, ObjAddr, hostLogFields)

			isSuccess, message, retData := GetHttpResult(req, addHostURL, common.HTTPCreate, host)
			if !isSuccess {
				ret := fmt.Sprintf("%s新加失败%s;", host["InnerIP"].(string), message)
				errMsg = append(errMsg, fmt.Sprintf("%d行%v", index, ret))
				continue
			}

			retHost := retData.(map[string]interface{})
			hostID, _ := util.GetIntByInterface(retHost[common.BKHostIDField])
			addParams[common.BKHostIDField] = hostID
			innerIP := host[common.BKHostInnerIPField].(string)

			isSuccess, message, _ = GetHttpResult(req, addModulesURL, common.HTTPCreate, addParams)
			if !isSuccess {
				blog.Error("add hosthostconfig error, params:%v, error:%s", addParams, message)
				errMsg = append(errMsg, fmt.Sprintf("%d行%v", index, innerIP))
				continue
			}
			strHostID := fmt.Sprintf("%d", hostID)
			logContent, _ := logObj.GetHostLog(strHostID, false)

			logConents = append(logConents, auditoplog.AuditLogExt{ID: hostID, Content: logContent, ExtKey: innerIP})

		}

		succMsg = append(succMsg, fmt.Sprintf("%d", index))
	}

	if 0 < len(logConents) {
		logAPIClient := sourceAuditAPI.NewClient(auditAddr)
		_, err := logAPIClient.AuditHostsLog(logConents, "导入主机", ownerID, fmt.Sprintf("%d", appID), user, auditoplog.AuditOpTypeAdd)
		//addAuditLogs(req, logAdd, "新加主机", ownerID, appID, user, auditAddr)
		if nil != err {
			blog.Errorf("add audit log error %s", err.Error())
		}
	}

	if 0 < len(errMsg) || 0 < len(updateErrMsg) {
		return errors.New("导入主机出现错误"), succMsg, updateErrMsg, errMsg
	}

	return nil, succMsg, updateErrMsg, errMsg
}

//EnterIP 将机器导入到制定模块或者空闲机器， 已经存在机器，不操作
func EnterIP(req *restful.Request, ownerID string, appID, moduleID int, IP, osType, hostname, appName, setName, moduleName, hostAddr, ObjAddr, auditAddr string, errHandle errorHandle.DefaultCCErrorIf) error {

	user := sencecommon.GetUserFromHeader(req)

	addHostURL := hostAddr + "/host/v1/insts/"

	addParams := make(map[string]interface{})
	addParams[common.BKAppIDField] = appID
	addParams[common.BKModuleIDField] = []int{moduleID}
	addModulesURL := hostAddr + "/host/v1/meta/hosts/modules/"

	conds := map[string]interface{}{
		common.BKHostInnerIPField: IP,
		common.BKCloudIDField:     common.BKDefaultDirSubArea,
	}
	hostList, err := GetHostInfoByConds(req, hostAddr, conds)
	if nil != err {
		return errors.New("查询主机信息失败")
	}
	if len(hostList) > 0 {
		return nil
	}

	host := make(map[string]interface{})
	host[common.BKHostInnerIPField] = IP
	host[common.BKOSTypeField] = osType

	host["import_from"] = common.HostAddMethodAgent
	host[common.BKCloudIDField] = common.BKDefaultDirSubArea
	defaultFields := getHostFields(ownerID, ObjAddr)
	//补充未填写字段的默认值
	for key, val := range defaultFields {
		_, ok := host[key]
		if !ok {

			host[key] = val[common.BKDefaultField]
		}
	}

	isSuccess, message, retData := GetHttpResult(req, addHostURL, common.HTTPCreate, host)
	if !isSuccess {
		return errors.New(fmt.Sprintf("add host to cmdb error,error:%s", message))
	}

	retHost := retData.(map[string]interface{})
	hostID, _ := util.GetIntByInterface(retHost[common.BKHostIDField])
	addParams[common.BKHostIDField] = hostID

	isSuccess, message, _ = GetHttpResult(req, addModulesURL, common.HTTPCreate, addParams)
	if !isSuccess {
		blog.Error("enterip add hosthostconfig error, params:%v, error:%s", addParams, message)
		return errors.New(fmt.Sprintf("add hosthostconfig error,error:%s", message))
	}

	//prepare the log
	hostLogFields, _ := GetHostLogFields(req, ownerID, ObjAddr)
	logObj := NewHostLog(req, common.BKDefaultOwnerID, "", hostAddr, ObjAddr, hostLogFields)
	content, _ := logObj.GetHostLog(fmt.Sprintf("%d", hostID), false)
	logAPIClient := sourceAuditAPI.NewClient(auditAddr)
	logAPIClient.AuditHostLog(hostID, content, "enter IP HOST", IP, ownerID, fmt.Sprintf("%d", appID), user, auditoplog.AuditOpTypeAdd)
	logClient, err := NewHostModuleConfigLog(req, nil, hostAddr, ObjAddr, auditAddr)
	logClient.SetHostID([]int{hostID})
	logClient.SetDescPrefix("enter IP ")
	logClient.SaveLog(fmt.Sprintf("%d", appID), user)
	return nil

}
