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
	"configcenter/src/common/core/cc/api"
	"configcenter/src/common/util"
	scenecommon "configcenter/src/scene_server/common"
	"configcenter/src/scene_server/validator"
	sourceAuditAPI "configcenter/src/source_controller/api/auditlog"
	sourceAPI "configcenter/src/source_controller/api/object"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"

	simplejson "github.com/bitly/go-simplejson"
	restful "github.com/emicklei/go-restful"

	"time"
)

//AddHost return error info
func AddHost(req *restful.Request, ownerID string, appID int, hostInfos map[int]map[string]interface{}, moduleID int, cc *api.APIResource) (error, []string, []string, []string) {
	forward := &sourceAPI.ForwardParam{Header: req.Request.Header}
	user := scenecommon.GetUserFromHeader(req)

	hostAddr := cc.HostCtrl()
	ObjAddr := cc.ObjCtrl()
	auditAddr := cc.AuditCtrl()
	addHostURL := hostAddr + "/host/v1/insts/"
	uHostURL := ObjAddr + "/object/v1/insts/host"

	language := util.GetActionLanguage(req)
	errHandle := cc.Error.CreateDefaultCCErrorIf(language)
	langHandle := cc.Lang.CreateDefaultCCLanguageIf(language)

	addParams := make(map[string]interface{})
	addParams[common.BKAppIDField] = appID
	addParams[common.BKModuleIDField] = []int{moduleID}
	addModulesURL := hostAddr + "/host/v1/meta/hosts/modules/"

	allHostList, err := GetHostInfoByConds(req, hostAddr, nil, langHandle)
	if nil != err {
		return errors.New(langHandle.Language("host_search_fail")), nil, nil, nil
	}

	//get asst field
	objCli := sourceAPI.NewClient("")
	objCli.SetAddress(ObjAddr)
	asst := map[string]interface{}{}
	asst[common.BKOwnerIDField] = ownerID
	asst[common.BKObjIDField] = common.BKInnerObjIDHost
	searchData, _ := json.Marshal(asst)
	objCli.SetAddress(ObjAddr)
	asstDes, err := objCli.SearchMetaObjectAsst(&sourceAPI.ForwardParam{Header: req.Request.Header}, searchData)
	if nil != err {
		return errors.New("查询主机属性失败"), nil, nil, nil
	}

	hostMap := convertHostInfo(allHostList)
	input := make(map[string]interface{}, 2)     //更新主机数据
	condInput := make(map[string]interface{}, 1) //更新主机条件
	var errMsg, succMsg, updateErrMsg []string   //新加错误， 成功，  更新失败
	iSubArea := common.BKDefaultDirSubArea

	defaultFields := getHostFields(forward, ownerID, ObjAddr)
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
			errMsg = append(errMsg, langHandle.Languagef("host_import_innerip_empty", index))
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

				errMsg = append(errMsg, langHandle.Languagef("host_import_property_need_set", index, key))
				continue
			}
			notExistFields = append(notExistFields, key)
		}
		blog.Infof("no validate fields %v", notExistFields)

		valid := validator.NewValidMapWithKeyFields(common.BKDefaultOwnerID, common.BKInnerObjIDHost, ObjAddr, notExistFields, forward, errHandle)

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
				updateErrMsg = append(updateErrMsg, langHandle.Languagef("host_import_update_fail", index, innerIP, message))
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
				ip, _ := host["InnerIP"].(string)
				errMsg = append(errMsg, langHandle.Languagef("host_import_add_fail", index, ip, message))
				continue
			}

			retHost := retData.(map[string]interface{})
			hostID, _ := util.GetIntByInterface(retHost[common.BKHostIDField])

			//add host asst attr
			hostAsstData := scenecommon.ExtractDataFromAssociationField(int64(hostID), host, asstDes)
			err = scenecommon.CreateInstAssociation(ObjAddr, req, hostAsstData)
			if nil != err {
				blog.Error("add host asst attr error : %v", err)
				errMsg = append(errMsg, fmt.Sprintf("%d行%v", index, innerIP))
				continue
			}

			addParams[common.BKHostIDField] = hostID
			innerIP := host[common.BKHostInnerIPField].(string)

			isSuccess, message, _ = GetHttpResult(req, addModulesURL, common.HTTPCreate, addParams)
			if !isSuccess {
				blog.Error("add hosthostconfig error, params:%v, error:%s", addParams, message)
				errMsg = append(errMsg, langHandle.Languagef("host_import_add_host_module", index, innerIP))
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
		_, err := logAPIClient.AuditHostsLog(logConents, "import host", ownerID, fmt.Sprintf("%d", appID), user, auditoplog.AuditOpTypeAdd)
		//addAuditLogs(req, logAdd, "新加主机", ownerID, appID, user, auditAddr)
		if nil != err {
			blog.Errorf("add audit log error %s", err.Error())
		}
	}

	if 0 < len(errMsg) || 0 < len(updateErrMsg) {
		return errors.New(langHandle.Language("host_import_err")), succMsg, updateErrMsg, errMsg
	}

	return nil, succMsg, updateErrMsg, errMsg
}

// EnterIP 将机器导入到制定模块或者空闲机器， 已经存在机器，不操作
func EnterIP(req *restful.Request, ownerID string, appID, moduleID int, IP, osType, hostname, appName, setName, moduleName string, cc *api.APIResource) error {

	user := scenecommon.GetUserFromHeader(req)

	hostAddr := cc.HostCtrl()
	ObjAddr := cc.ObjCtrl()
	auditAddr := cc.AuditCtrl()

	language := util.GetActionLanguage(req)
	//errHandle := cc.Error.CreateDefaultCCErrorIf(language)
	langHandle := cc.Lang.CreateDefaultCCLanguageIf(language)

	addHostURL := hostAddr + "/host/v1/insts/"

	addParams := make(map[string]interface{})
	addParams[common.BKAppIDField] = appID
	addParams[common.BKModuleIDField] = []int{moduleID}
	addModulesURL := hostAddr + "/host/v1/meta/hosts/modules/"

	conds := map[string]interface{}{
		common.BKHostInnerIPField: IP,
		common.BKCloudIDField:     common.BKDefaultDirSubArea,
	}
	hostList, err := GetHostInfoByConds(req, hostAddr, conds, langHandle)
	if nil != err {
		return errors.New(langHandle.Language("host_search_fail")) // "查询主机信息失败")
	}
	if len(hostList) > 0 {
		return nil
	}

	host := make(map[string]interface{})
	host[common.BKHostInnerIPField] = IP
	host[common.BKOSTypeField] = osType

	host["import_from"] = common.HostAddMethodAgent
	host[common.BKCloudIDField] = common.BKDefaultDirSubArea
	forward := &sourceAPI.ForwardParam{Header: req.Request.Header}
	defaultFields := getHostFields(forward, ownerID, ObjAddr)
	//补充未填写字段的默认值
	for key, val := range defaultFields {
		_, ok := host[key]
		if !ok {

			host[key] = val[common.BKDefaultField]
		}
	}

	isSuccess, message, retData := GetHttpResult(req, addHostURL, common.HTTPCreate, host)
	if !isSuccess {
		return errors.New(langHandle.Languagef("host_agent_add_host_fail", message))
	}

	retHost := retData.(map[string]interface{})
	hostID, _ := util.GetIntByInterface(retHost[common.BKHostIDField])
	addParams[common.BKHostIDField] = hostID

	isSuccess, message, _ = GetHttpResult(req, addModulesURL, common.HTTPCreate, addParams)
	if !isSuccess {
		blog.Error("enterip add hosthostconfig error, params:%v, error:%s", addParams, message)
		return errors.New(langHandle.Languagef("host_agent_add_host_module_fail", message))
	}

	//prepare the log
	hostLogFields, _ := GetHostLogFields(req, ownerID, ObjAddr)
	logObj := NewHostLog(req, common.BKDefaultOwnerID, "", hostAddr, ObjAddr, hostLogFields)
	content, _ := logObj.GetHostLog(fmt.Sprintf("%d", hostID), false)
	logAPIClient := sourceAuditAPI.NewClient(auditAddr)
	logAPIClient.AuditHostLog(hostID, content, "enter IP HOST", IP, ownerID, fmt.Sprintf("%d", appID), user, auditoplog.AuditOpTypeAdd)
	logClient, err := NewHostModuleConfigLog(req, nil, hostAddr, ObjAddr, auditAddr)
	logClient.SetHostID([]int{hostID})
	logClient.SetDesc("host module change")
	logClient.SaveLog(fmt.Sprintf("%d", appID), user)
	return nil

}

// AddHostV22 add host
func AddHostV22(CC *api.APIResource, req *restful.Request, bodyData []byte, hostAddr, ObjAddr, auditAddr string) error {
	type inputStruct struct {
		Ips        []string `json:"ip_list"`
		ModuleName string   `json:"bk_module_name"`
		SetName    string   `json:"bk_set_name"`
		AppName    string   `json:"bk_biz_name"`
		BKCloudID  int      `json:"bk_cloud_id"`
		HostInfo   string   `json:"host_info"`
	}
	var data inputStruct
	error := json.Unmarshal([]byte(bodyData), &data)
	js, err := simplejson.NewJson([]byte(data.HostInfo))
	if nil != err {

	}
	hostsMap, error := js.Map()
	if nil != error {
		blog.Error(" api fail to unmarshal json, error information is %s, msg:%s", error.Error(), string(bodyData))
		return errors.New("api fail to unmarshal json")
	}

	blog.Errorf("zzzz =%v", hostsMap)
	hostData := reflect.ValueOf(hostsMap["data"])
	input := make(common.KvMap)
	input["ips"] = data.Ips
	input[common.BKModuleNameField] = data.ModuleName
	input[common.BKSetNameField] = data.SetName
	input[common.BKAppNameField] = data.AppName
	input[common.BKOwnerIDField] = common.BKDefaultOwnerID
	blog.Errorf("hostdata:%v", hostData)
	//hostdata存在 说明在业务或者在资源池
	if hostData.Len() > 0 {
		blog.Errorf("hostdata 信息存在,%v", hostData)
		var appID, moduleID, setID, hostID int
		var osType, hostname, selfModuleName string
		for _, item := range hostsMap["data"].([]interface{}) {
			blog.Error("item:%v", item)
			newAppID := item.(map[string]interface{})[common.BKAppIDField]
			newModuleID := item.(map[string]interface{})[common.BKModuleIDField]
			newSetID := item.(map[string]interface{})[common.BKSetIDField]
			appID, _ = util.GetIntByInterface(newAppID)
			moduleID, _ = util.GetIntByInterface(newModuleID)
			setID, _ = util.GetIntByInterface(newSetID)
			osType = item.(map[string]interface{})[common.BKOSNameField].(string)
			hostname = item.(map[string]interface{})[common.BKHostNameField].(string)
			newhostID := item.(map[string]interface{})[common.BKHostIDField]
			hostID, _ = util.GetIntByInterface(newhostID)
			selfModuleName = item.(map[string]interface{})[common.BKModuleNameField].(string)
		}
		blog.Errorf("hostname:%s,ostype:%s,appid:%d,setid:%d,muduleid:%d,modulename:%s", hostname, osType, appID, setID, moduleID, data.ModuleName)
		error := addHostToModule(CC, req, appID, hostID, moduleID, data.AppName, data.SetName, data.ModuleName, selfModuleName, hostAddr, ObjAddr, auditAddr)
		if nil != error {
			blog.Error("add host error, params:%v, error:%s", input, error)
			return errors.New("add host error")
		}
		return nil
	} else {
		//ip+platid主机不在当前供应商下 添加
		// hostdata没有找到 添加ip到资源池
		blog.Error("ip host not found")
		// paramJSON, _ := json.Marshal(input)
		delModulesURL := hostAddr + "/host/v1/host/add/module"
		isSuccess, errMsg, _ := GetHttpResult(req, delModulesURL, common.HTTPSelectPost, input)
		if !isSuccess {
			blog.Error("remove hosthostconfig error, params:%v, error:%s", input, errMsg)
			return errors.New("remove hosthostconfig error")
		}

	}
	return nil
}

// addHostToModule add host to module
func addHostToModule(CC *api.APIResource, req *restful.Request, appID, hostID, moduleID int, appName, setName, moduleName, selfModuleName string, hostAddr, ObjAddr, auditAddr string) error {
	//默认业务与主机业务一致 说明主机存在资源池
	language := util.GetActionLanguage(req)
	langHandle := CC.Lang.CreateDefaultCCLanguageIf(language)
	errHandle := CC.Error.CreateDefaultCCErrorIf(language)

	//get default app
	ownerAppID, err := GetDefaultAppID(req, common.BKDefaultOwnerID, common.BKAppIDField, ObjAddr, langHandle)
	blog.Errorf("ownerAppID===%d", ownerAppID)
	if err != nil {
		blog.Infof("ownerid %s 资源池未找到", ownerAppID)
		return errors.New("not found resource pool")
	}
	if 0 == ownerAppID {
		blog.Infof("ownerid %s 资源池未找到", ownerAppID)
		return errors.New("not found resource pool")
	}
	introAppID, _, moduleID, err := GetTopoIDByName(req, common.BKDefaultOwnerID, appName, setName, moduleName, ObjAddr, errHandle)
	if nil != err {
		blog.Error("get app  topology id by name error:%s, msg: applicationName:%s, setName:%s, moduleName:%s", err.Error(), appName, setName, moduleName)
		return errors.New("search appliaction module not foud ")
	}
	blog.Errorf("--->>> appid:%s,==moduleid:%s,dataid:%s", introAppID, moduleID, appID)
	//如果为0 说明输入的不存在 返回成功
	if 0 == introAppID || 0 == moduleID {
		return nil
	}
	user := scenecommon.GetUserFromHeader(req)
	logClient, err := NewHostModuleConfigLog(req, nil, hostAddr, ObjAddr, auditAddr)
	if 0 != ownerAppID && appID == ownerAppID {
		blog.Errorf("default app 一致")
		params := make(map[string]interface{})
		params[common.BKAppIDField] = appID
		params[common.BKHostIDField] = hostID
		delModulesURL := hostAddr + "/host/v1/meta/hosts/defaultmodules"
		isSuccess, _, _ := GetHttpResult(req, delModulesURL, common.HTTPDelete, params)
		if !isSuccess {
			blog.Error("remove modulehostconfig error, params:%v, error:%v", params, err)
			return errors.New("remove modulehostconfig error")
		}
		logClient.SetDesc("remove host from module")
		host := []int{hostID}
		logClient.SetHostID(host)
		logClient.SaveLog(fmt.Sprintf("%d", appID), user)
		blog.Errorf("remove ok")

		moduleHostConfigParams := make(map[string]interface{})
		moduleHostConfigParams[common.BKAppIDField] = introAppID
		moduleHostConfigParams[common.BKHostIDField] = hostID
		moduleHostConfigParams[common.BKModuleIDField] = []int{moduleID}
		addModulesURL := hostAddr + "/host/v1/meta/hosts/modules"

		isSuccess, errMsg, _ := GetHttpResult(req, addModulesURL, common.HTTPCreate, moduleHostConfigParams)
		if !isSuccess {
			blog.Error("add hosthostconfig error, params:%v, error:%s", moduleHostConfigParams, errMsg)
			return errors.New("add hosthostconfig error")
		}
		logClient.SetDesc("add host to module")
		logClient.SetHostID(host)
		logClient.SaveLog(fmt.Sprintf("%d", appID), user)
		return nil
	} else {
		if introAppID == appID { //传入的ID和所在的业务ID一致
			// IsExistHostIDInApp 判断主机是否在传入的业务中
			blog.Errorf("is exist host in app")
			moduleHostConfigParams := make(map[string]interface{})
			moduleHostConfigParams[common.BKAppIDField] = appID
			moduleHostConfigParams[common.BKHostIDField] = hostID
			//如果主机在空闲机 删除
			if selfModuleName == common.DefaultResModuleName {
				delModulesURL := hostAddr + "/host/v1/meta/hosts/modules"
				isSuccess, errMsg, _ := GetHttpResult(req, delModulesURL, common.HTTPDelete, moduleHostConfigParams)
				if !isSuccess {
					blog.Error("remove hosthostconfig error, params:%v, error:%s", moduleHostConfigParams, errMsg)
					return errors.New("remove hosthostconfig error")
				}
			}

			moduleHostConfigParams[common.BKModuleIDField] = []int{moduleID}
			addModulesURL := hostAddr + "/host/v1/meta/hosts/modules"

			isSuccess, errMsg, _ := GetHttpResult(req, addModulesURL, common.HTTPCreate, moduleHostConfigParams)
			if !isSuccess {
				blog.Error("add hosthostconfig error, params:%v, error:%s", moduleHostConfigParams, errMsg)
				return errors.New("add hosthostconfig error")
			}
			blog.Info("add host to module suceess")
			logClient.SetDesc("add host to module")
			host := []int{hostID}
			logClient.SetHostID(host)
			logClient.SaveLog(fmt.Sprintf("%d", appID), user)
			return nil
		}
		blog.Errorf("host in other app")
		//说明主机在其他业务中 返回失败
		return errors.New("host in other app")
	}
	return nil
}
