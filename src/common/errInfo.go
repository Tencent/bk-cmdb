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

package common

// CC error number defined in this file
// Errno name is composed of the following format CCErr[XXX]
const (
	// the system code

	// CCSystemBusy the system is busy
	CCSystemBusy         = -1
	CCSystemUnknownError = -2
	CCSuccess            = 0
	CCSuccessStr         = "success"
	CCNoPermission       = 9900403

	// common error code 1199XXX

	// CCErrCommJSONUnmarshalFailed JSON deserialization failed
	CCErrCommJSONUnmarshalFailed = 1199000

	// CCErrCommJSONMarshalFailed JSON serialization failed
	CCErrCommJSONMarshalFailed = 1199001

	// CCErrCommHTTPDoRequestFailed the HTTP Request failed
	CCErrCommHTTPDoRequestFailed = 1199002

	// CCErrCommHTTPInputInvalid the input parameter is invalid, and the parameter here refers to the URL or Query parameter
	CCErrCommHTTPInputInvalid = 1199003

	// CCErrCommHTTPReadBodyFailed unable to read HTTP request body data
	CCErrCommHTTPReadBodyFailed = 1199004

	// CCErrCommHTTPBodyEmpty  HTTP request body data is not set
	CCErrCommHTTPBodyEmpty = 1199005

	// CCErrCommParamsInvalid parameter validation in the body is not pass
	CCErrCommParamsInvalid = 1199006

	// CCErrCommParamsNeedString  the parameter must be of type string
	CCErrCommParamsNeedString = 1199007

	// CCErrCommParamsLostField the parameter not specified
	CCErrCommParamsLostField = 1199008

	// CCErrCommParamsNeedInt the parameter must be of type int
	CCErrCommParamsNeedInt = 1199009

	// CCErrCommParamsNeedSet the parameter unassigned
	CCErrCommParamsNeedSet = 1199010

	// CCErrCommParamsIsInvalid the parameter is invalid or nonexistent
	CCErrCommParamsIsInvalid = 1199011

	// CCErrCommParseDataFailed failed to read data from data field
	CCErrCommParseDataFailed = 1199013

	// CCErrCommDuplicateItem duplicate data
	CCErrCommDuplicateItem = 1199014

	// CCErrCommOverLimit data length exceeds limit
	CCErrCommOverLimit = 1199015

	// CCErrFieldRegValidFailed regular verification failed
	CCErrFieldRegValidFailed = 1199016

	// CCErrCommDBSelectFailed database query failed
	CCErrCommDBSelectFailed = 1199017

	// CCErrCommDBInsertFailed database cannot add data
	CCErrCommDBInsertFailed = 1199018

	// CCErrCommNotFound the goal does not exist
	CCErrCommNotFound = 1199019

	// CCErrCommDBUpdateFailed database cannot update data
	CCErrCommDBUpdateFailed = 1199020

	// CCErrCommDBDeleteFailed database cannot delete data
	CCErrCommDBDeleteFailed = 1199021

	// CCErrCommRelyOnServerAddressFailed dependent service did not start
	CCErrCommRelyOnServerAddressFailed = 1199022

	// CCErrCommExcelTemplateFailed unable to generate and download
	CCErrCommExcelTemplateFailed = 1199023

	// CCErrCommParamsNeedTimeZone the parameter must be time zone type
	CCErrCommParamsNeedTimeZone = 1199024

	// CCErrCommParamsNeedBool the parameter must be bool type
	CCErrCommParamsNeedBool = 1199025

	// CCErrCommConfMissItem  missing configuration item
	CCErrCommConfMissItem = 1199026

	// CCErrCommNotAuthItem failed to get authorization information
	CCErrCommNotAuthItem = 1199027

	// CCErrCommNotAuthItem field validate failed
	CCErrCommFieldNotValid = 1199028

	// CCErrCommReplyDataFormatError Return data format error
	CCErrCommReplyDataFormatError = 1199029

	// CCErrCommReplyDataFormatError Return data format error
	CCErrCommPostInputParseError = 1199030

	// CCErrCommResourceInitFailed %s init failed
	CCErrCommResourceInitFailed = 1199031

	// CCErrCommParams should be string
	CCErrCommParamsShouldBeString = 1199032

	// CCErrCommSearchPropertyFailed get object property fields error
	CCErrCommSearchPropertyFailed = 1199033

	// CCErrCommParamsShouldBeEnum set enum
	CCErrCommParamsShouldBeEnum = 1199034

	// CCErrCommXXExceedLimit  xx exceed limit number
	CCErrCommXXExceedLimit = 1199035

	CCErrProxyRequestFailed      = 1199036
	CCErrRewriteRequestUriFailed = 1199037

	// CCErrCommLogicDataNil   need data %s is null
	CCErrCommInstDataNil = 1199038
	// CCErrCommInstFieldNotFound  %s field does not exist in %s
	CCErrCommInstFieldNotFound = 1199039
	// CCErrCommInstFieldConvertFail  convert %s  field %s to %s error %s
	CCErrCommInstFieldConvertFail = 1199040
	// CCErrCommUtilFail  handle %s error %s
	CCErrCommUtilHandleFail = 1199041
	// CCErrCommParamsNeedFloat the parameter must be float type
	CCErrCommParamsNeedFloat = 1199042
	// CCErrCommFieldNotValidFail  valid data error, %s
	CCErrCommFieldNotValidFail = 1199043

	CCErrCommNotAllSuccess = 1199044
	// parse auth attribute in api server rest filter failed.
	CCErrCommParseAuthAttributeFailed = 1199045

	// authorize request to auth center failed
	CCErrCommCheckAuthorizeFailed = 1199046

	// auth failed, do not have permission.
	CCErrCommAuthNotHavePermission = 1199047

	CCErrCommAuthorizeFailed             = 1199048
	CCErrCommRegistResourceToIAMFailed   = 1199049
	CCErrCommUnRegistResourceToIAMFailed = 1199050
	CCErrCommInappropriateVisitToIAM     = 1199051

	CCErrCommGetMultipleObject                = 1199052
	CCErrCommAuthCenterIsNotEnabled           = 1199053
	CCErrCommOperateBuiltInItemForbidden      = 1199054
	CCErrCommRemoveRecordHasChildrenForbidden = 1199055
	CCErrCommRemoveReferencedRecordForbidden  = 1199056
	CCErrCommParseBizIDFromMetadataInDBFailed = 1199057

	CCErrCommGenerateRecordIDFailed   = 1199058
	CCErrCommPageLimitIsExceeded      = 1199059
	CCErrCommUnexpectedParameterField = 1199060

	CCErrCommParseDBFailed                     = 1199061
	CCErrCommGetBusinessDefaultSetModuleFailed = 1199062

	CCErrCommParametersCountNotEnough         = 1199063
	CCErrCommFuncCalledWithInappropriateParam = 1199064

	// CCErrCommStartTransactionFailed start transaction failed
	CCErrCommStartTransactionFailed = 1199065
	// CCErrCommCommitTransactionFailed commit transaction failed
	CCErrCommCommitTransactionFailed = 1199066
	// CCErrCommAbortTransactionFailed abort transaction failed
	CCErrCommAbortTransactionFailed = 1199067

	CCErrCommListAuthorizedResourcedFromIAMFailed = 1199068
	CCErrParseAttrOptionEnumFailed                = 1199069

	// CCErrCommParamsNotSupportXXErr 参数%s的值%s 无效
	CCErrCommParamsValueInvalidError = 1199070

	// 构造DB查询条件失败
	CCErrConstructDBFilterFailed = 1199071
	CCErrGetNoAuthSkipURLFailed  = 1199072

	// CCErrCommValExceedMaxFailed %s field exceeds maximum value %v
	CCErrCommValExceedMaxFailed          = 1199073
	CCErrCommGlobalCCErrorNotInitialized = 1199074

	CCErrCommForbiddenOperateMainlineInstanceWithCommonAPI = 1199075
	CCErrTopoUpdateBuiltInCloudForbidden                   = 1199076

	// CCErrTopoModuleNotFoundError module [%s] does not exist in the business topology
	CCErrCommTopoModuleNotFoundError = 1199078
	// CCErrBizNotFoundError business [%s] does not exist
	CCErrCommBizNotFoundError      = 1199079
	CCErrParseAttrOptionListFailed = 1199080
	// one argument: maxValue
	CCErrExceedMaxOperationRecordsAtOnce = 1199081

	CCErrCommListAuthorizedResourceFromIAMFailed             = 1199082
	CCErrCommModifyFieldForbidden                            = 1199083
	CCErrCommForbiddenOperateInnerModelInstanceWithCommonAPI = 1199084
	CCErrCommUnexpectedFieldType                             = 1199085

	CCErrCommGetBusinessIDByHostIDFailed = 1199086

	// CCErrCommOPInProgressErr have the same task[%s] in progress
	CCErrCommOPInProgressErr = 1199087
	// CCErrCommRedisOPErr operate redis error.
	CCErrCommRedisOPErr = 1199088

	// CCErrArrayLengthWrong the length of the array is wrong
	CCErrArrayLengthWrong = 1199089

	//CCIllegalRegularExpression the regular expression's type assertion failed
	CCIllegalRegularExpression = 1199090

	// too many requests
	CCErrTooManyRequestErr = 1199997

	// unknown or unrecognized error
	CCErrorUnknownOrUnrecognizedError = 1199998

	// CCErrCommInternalServerError %s Internal Server Error
	CCErrCommInternalServerError = 1199999

	// apiserver 1100XXX
	CCErrAPIGetAuthorizedAppListFromAuthFailed = 1100001
	CCErrAPIGetUserResourceAuthStatusFailed    = 1100002
	CCErrAPINoObjectInstancesIsFound           = 1100003

	// toposerver 1101XXX
	// CCErrTopoInstCreateFailed unable to create the instance
	CCErrTopoInstCreateFailed = 1101000

	// CCErrTopoInstDeleteFailed unable to delete the instance
	CCErrTopoInstDeleteFailed = 1101001

	// CCErrTopoInstUpdateFailed unable to update the instance
	CCErrTopoInstUpdateFailed = 1101002

	// CCErrTopoInstSelectFailed unable to search the instance
	CCErrTopoInstSelectFailed = 1101003

	// CCErrTopoModuleCreateFailed unable to create a module
	CCErrTopoModuleCreateFailed = 1101004

	// CCErrTopoModuleDeleteFailed unable to delete a module
	CCErrTopoModuleDeleteFailed = 1101005

	// CCErrTopoModuleUpdateFailed unable to update a module
	CCErrTopoModuleUpdateFailed = 1101006

	// CCErrTopoModuleSelectFailed unable to select a module
	CCErrTopoModuleSelectFailed = 1101007

	// CCErrTopoSetCreateFailed unable to create a set
	CCErrTopoSetCreateFailed = 1101008

	// CCErrTopoSetDeleteFailed unable to delete a set
	CCErrTopoSetDeleteFailed = 1101009

	// CCErrTopoSetUpdateFailed unable to update a set
	CCErrTopoSetUpdateFailed = 1101010

	// CCErrTopoSetSelectFailed unable to select a set
	CCErrTopoSetSelectFailed = 1101011

	// CCErrTopoInstHasHostChild include hosts
	CCErrTopoInstHasHostChild = 1101012

	// CCErrTopoObjectCreateFailed unable to create a object
	CCErrTopoObjectCreateFailed = 1101013

	// CCErrTopoObjectDeleteFailed unable to delete a object
	CCErrTopoObjectDeleteFailed = 1101014

	// CCErrTopoObjectUpdateFailed unable to update a object
	CCErrTopoObjectUpdateFailed = 1101015

	// CCErrTopoObjectSelectFailed unable to select a object
	CCErrTopoObjectSelectFailed = 1101016

	// CCErrTopoObjectAttributeCreateFailed unable to create a object attribute
	CCErrTopoObjectAttributeCreateFailed = 1101017

	// CCErrTopoObjectAttributeDeleteFailed unable to delete a object attribute
	CCErrTopoObjectAttributeDeleteFailed = 1101018

	// CCErrTopoObjectAttributeUpdateFailed unable to update a object attribute
	CCErrTopoObjectAttributeUpdateFailed = 1101019

	// CCErrTopoObjectAttributeSelectFailed unable to select a object attribute
	CCErrTopoObjectAttributeSelectFailed = 1101020

	// CCErrTopoObjectClassificationCreateFailed unable to create a object classification
	CCErrTopoObjectClassificationCreateFailed = 1101021

	// CCErrTopoObjectClassificationDeleteFailed unbale to delete a object classification
	CCErrTopoObjectClassificationDeleteFailed = 1101022

	// CCErrTopoObjectClassificationUpdateFailed unable to update a object classification
	CCErrTopoObjectClassificationUpdateFailed = 1101023

	// CCErrTopoObjectClassificationSelectFailed unable to select a object classification
	CCErrTopoObjectClassificationSelectFailed = 1101024

	// CCErrTopoObjectGroupCreateFailed unable to create object group
	CCErrTopoObjectGroupCreateFailed = 1101025

	// CCErrTopoObjectGroupDeleteFailed unable to delete a object group
	CCErrTopoObjectGroupDeleteFailed = 1101026

	// CCErrTopoObjectGroupUpdateFailed unable to update a object group
	CCErrTopoObjectGroupUpdateFailed = 1101027

	// CCErrTopoObjectGroupSelectFailed unable to select a object group
	CCErrTopoObjectGroupSelectFailed = 1101028

	// CCErrTopoObjectClassificationHasObject the object classification can't be deleted under classification
	CCErrTopoObjectClassificationHasObject = 1101029

	// CCErrTopoHasHostCheckFailed cannot detect if host information is included
	CCErrTopoHasHostCheckFailed = 1101030

	// CCErrTopoHasHost include host
	CCErrTopoHasHost = 1101030

	// CCErrTopoGetCloudErrStrFailed get cloud error
	CCErrTopoGetCloudErrStrFailed = 1101031
	// CCErrTopoCloudNotFound   cloud area not found
	CCErrTopoCloudNotFound = 1101032

	// CCErrTopoGetAppFailed search app err %s
	CCErrTopoGetAppFailed = 1101033
	// CCErrTopoGetModuleFailed search  module err %s
	CCErrTopoGetModuleFailed = 1101034
	// CCErrTopoBizTopoOverLevel the mainline topo level over limit
	CCErrTopoBizTopoLevelOverLimit = 1101035
	// CCErrTopoInstHasBeenAssociation the instance has been associated
	CCErrTopoInstHasBeenAssociation = 1101036
	// it is forbidden to delete , that has some insts
	CCErrTopoObjectHasSomeInstsForbiddenToDelete = 1101037
	// the associations %s->%s already exist
	CCErrTopoAssociationAlreadyExist = 1101038
	// the source association object does not exist
	CCErrTopoAssociationSourceObjectNotExist = 1101039
	// the destination association object does not exist
	CCErrTopoAssociationDestinationObjectNotExist = 1101040
	// invalid object association id, should be int64
	CCErrTopoInvalidObjectAssociationID = 1101041
	// got multiple object association with one association id
	CCErrTopoGotMultipleAssociationInstance = 1101042
	// association with a object has multiple instance, can not be deleted.
	CCErrTopoAssociationHasAlreadyBeenInstantiated = 1101043
	// get association kind with id failed.
	CCErrTopoGetAssociationKindFailed = 1101044
	// create object association missing object kind id, src object id or destination object id.
	CCErrorTopoAssociationMissingParameters = 1101045
	// the given association id does not exist.
	CCErrorTopoObjectAssociationNotExist = 1101046
	// update object association, but update fields that can not be updated.
	CCErrorTopoObjectAssociationUpdateForbiddenFields = 1101047
	// mainline object association do not exist
	CCErrorTopoMainlineObjectAssociationNotExist = 1101048
	// CCErrorTopoImportAssociation  import association error
	CCErrorTopoImportAssociation = 1101049
	// got multiple association kind with a id
	CCErrorTopoGetMultipleAssocKindInstWithOneID = 1101050
	// delete a pre-defined association kind.
	CCErrorTopoDeletePredefinedAssociationKind = 1101051
	// create new instance for a new association, but association map is 1:1
	CCErrorTopoCreateMultipleInstancesForOneToOneAssociation = 1101052
	// the object has associate to another object, or has been associated by another one.
	CCErrorTopoObjectHasAlreadyAssociated = 1101053
	// update a pre-defined association, it's forbidden.
	CCErrorTopoUpdatePredefinedAssociation = 1101054
	// can not delete a pre-defined association.
	CCErrorTopoDeletePredefinedAssociation = 1101055
	// association do not exist.
	CCErrorTopoAssociationDoNotExist = 1101056
	// create model's instance batch, but instance's data missing field bk_inst_name
	CCErrorTopoObjectInstanceMissingInstanceNameField = 1101057
	// object instance's bk_inst_name filed is not string
	CCErrorTopoInvalidObjectInstanceNameFieldValue = 1101058
	// create model's instance patch, but instance's name is duplicate.
	CCErrorTopoMultipleObjectInstanceName = 1101059

	CCErrorTopoAssociationKindHasBeenUsed                     = 1101060
	CCErrorTopoCreateMultipleInstancesForOneToManyAssociation = 1101061
	CCErrTopoAppDeleteFailed                                  = 1101131
	CCErrTopoAppUpdateFailed                                  = 1101132
	CCErrTopoAppSearchFailed                                  = 1101133
	CCErrTopoAppCreateFailed                                  = 1101134
	CCErrTopoForbiddenToDeleteModelFailed                     = 1101135
	CCErrTopoMainlineCreatFailed                              = 1101137
	CCErrTopoMainlineDeleteFailed                             = 1101138
	CCErrTopoMainlineSelectFailed                             = 1101139
	CCErrTopoTopoSelectFailed                                 = 1101140
	CCErrTopoDeleteMainLineObjectAndInstNameRepeat            = 1101149
	CCErrHostNotAllowedToMutiBiz                              = 1101150
	CCErrTopoGraphicsSearchFailed                             = 1101151
	CCErrTopoGraphicsUpdateFailed                             = 1101152
	CCErrTopoObjectUniqueCreateFailed                         = 1101160
	CCErrTopoObjectUniqueUpdateFailed                         = 1101161
	CCErrTopoObjectUniqueDeleteFailed                         = 1101162
	CCErrTopoObjectUniqueSearchFailed                         = 1101163
	CCErrTopoObjectPropertyNotFound                           = 1101164
	CCErrTopoObjectPropertyUsedByUnique                       = 1101165
	CCErrTopoObjectUniqueKeyKindInvalid                       = 1101166
	CCErrTopoObjectUniquePresetCouldNotDelOrEdit              = 1101167
	CCErrTopoObjectUniqueCanNotHasMultipleMustCheck           = 1101168
	CCErrTopoObjectUniqueShouldHaveMoreThanOne                = 1101069
	// association kind has been apply to object
	CCErrorTopoAssKindHasApplyToObject = 1101070
	// pre definition association kind can not be delete
	CCErrorTopoPreAssKindCanNotBeDelete = 1101071
	CCErrorTopoAsstKindIsNotExist       = 1101072
	CCErrorAsstInstIsNotExist           = 1101073
	CCErrorInstToAsstIsNotExist         = 1101074
	CCErrorInstHasAsst                  = 1101075
	CCErrTopoCreateAssocKindFailed      = 1101076
	CCErrTopoUpdateAssoKindFailed       = 1101077
	CCErrTopoDeleteAssoKindFailed       = 1101078
	CCErrTopoModuleIDNotfoundFailed     = 1101080
	CCErrTopoBkAppNotAllowedDelete      = 1101081
	// CCErrorTopoAssociationKindMainlineUnavailable can't use bk_mainline in this case
	CCErrorTopoAssociationKindMainlineUnavailable = 1101082
	// CCErrorTopoAssociationKindInconsistent means AssociationKind parameter Inconsistent with caller method
	CCErrorTopoAssociationKindInconsistent = 1101083
	// CCErrorTopoModelStopped means model have been stopped to use
	CCErrorTopoModelStopped = 1101084
	// mainline's object unique can not be updated, deleted or create new rules.
	CCErrorTopoMainlineObjectCanNotBeChanged   = 1101085
	CCErrorTopoGetAuthorizedBusinessListFailed = 1101086
	CCErrTopoArchiveBusinessHasHost            = 1101087

	CCErrorTopoFullTextFindErr              = 1101088
	CCErrorTopoFullTextClientNotInitialized = 1101089

	CCErrorTopoUpdateModuleFromTplServiceCategoryForbidden = 1101090
	CCErrorTopoUpdateModuleFromTplNameForbidden            = 1101091
	CCErrTopoCanNotAddRequiredAttributeForMainlineModel    = 1101092
	CCErrorTopoObjectInstanceObjIDFieldConflictWithURL     = 1101093
	CCErrTopoImportMainlineForbidden                       = 1101094

	CCErrorTopoSyncModuleTaskFailed    = 1101095
	CCErrorTopoSyncModuleTaskIsRunning = 1101096

	CCErrorTopoForbiddenOperateModuleOnSetInitializedByTemplate = 1101097
	CCErrorTopoForbiddenDeleteBuiltInSetModule                  = 1101098
	CCErrorTopoModuleNameDuplicated                             = 1101099

	CCErrorTopoPathParamPaserFailed                = 1101100
	CCErrorTopoSearchModelAttriFailedPleaseRefresh = 1101101
	CCErrorTopoOnlyResourceDirNameCanBeUpdated     = 1101102
	CCErrorTopoOperateReourceDirFailNotExist       = 1101103
	CCErrorTopoResourceDirIdleModuleCanNotRemove   = 1101104
	CCErrorTopoResourceDirUsedInCloudSync          = 1101105

	CCErrorModelNotFound = 1101106

	CCErrorAttributeNameDuplicated = 1101107
	CCErrorSetNameDuplicated       = 1101108

	// CCErrorCheckRegularFailed check regular expression failed
	CCErrorCheckRegularFailed   = 1101109
	CCErrorParamExceedMaxLength = 1101110

	// object controller 1102XXX

	// CCErrObjectPropertyGroupInsertFailed failed to save the property group
	CCErrObjectPropertyGroupInsertFailed = 1102000
	// CCErrObjectPropertyGroupDeleteFailed failed to delete the property group
	CCErrObjectPropertyGroupDeleteFailed = 1102001
	// CCErrObjectPropertyGroupSelectFailed failed to select the property group
	CCErrObjectPropertyGroupSelectFailed = 1102002
	// CCErrObjectPropertyGroupUpdateFailed failed to update the filed
	CCErrObjectPropertyGroupUpdateFailed = 1102003

	CCErrObjectCreateInstFailed       = 1102004
	CCErrObjectUpdateInstFailed       = 1102005
	CCErrObjectDeleteInstFailed       = 1102006
	CCErrObjectSelectInstFailed       = 1102007
	CCErrObjectSelectIdentifierFailed = 1102008

	// CCErrObjectDBOpErrno failed to operation database
	CCErrObjectDBOpErrno = 1102020

	// event_server 1103XXX
	// CCErrEventSubscribeInsertFailed failed to save the Subscribe
	CCErrEventSubscribeInsertFailed = 1103000

	// CCErrEventSubscribeDeleteFailed failed to delete the Subscribe
	CCErrEventSubscribeDeleteFailed = 1103001

	// CCErrEventSubscribeSelectFailed failed to select the Subscribe
	CCErrEventSubscribeSelectFailed = 1103002

	// CCErrEventSubscribeUpdateFailed failed to update the filed
	CCErrEventSubscribeUpdateFailed = 1103003

	// CCErrEventSubscribePingFailed failed to ping the target
	CCErrEventSubscribePingFailed = 1103004
	// CCErrEventSubscribePingFailed failed to telnet the target
	CCErrEventSubscribeTelnetFailed = 1103005
	// CCErrEventOperateSuccessBUtSentEventFailed failed to sent event
	CCErrEventPushEventFailed = 1103006

	CCErrEventChainNodeNotExist = 1103007
	CCErrEventDetailNotExist    = 1103008

	// host 1104XXX
	CCErrHostModuleRelationAddFailed = 1104000

	// migrate 1105XXX
	//  CCErrCommMigrateFailed failed to migrate
	CCErrCommMigrateFailed        = 1105000
	CCErrCommInitAuthCenterFailed = 1105001

	// host controller 1106XXX
	CCErrHostSelectInst                  = 1106000
	CCErrHostCreateInst                  = 1106002
	CCErrHostGetSnapshot                 = 1106003
	CCErrHostTransferModule              = 1106004
	CCErrDelDefaultModuleHostConfig      = 1106005
	CCErrGetModule                       = 1106006
	CCErrDelOriginHostModuelRelationship = 1106007
	CCErrGetOriginHostModuelRelationship = 1106008
	CCErrTransferHostFromPool            = 1106009
	CCErrAlreadyAssign                   = 1106010
	CCErrNotBelongToIdleModule           = 1106011
	CCErrTransfer2ResourcePool           = 1106012
	CCErrCreateUserCustom                = 1106013
	CCErrHostFavouriteQueryFail          = 1106014
	CCErrHostFavouriteCreateFail         = 1106015
	CCErrHostFavouriteUpdateFail         = 1106016
	CCErrHostFavouriteDeleteFail         = 1106017
	CCErrHostFavouriteDupFail            = 1106018
	CCErrHostGetSnapshotChannelEmpty     = 1106019
	CCErrHostGetSnapshotChannelClose     = 1106020
	CCErrCloudCreateSyncTaskFail         = 1106021
	CCErrCloudConfirmHistoryAddFail      = 1106022
	CCErrCloudSyncHistorySearchFail      = 1106023
	CCErrHostGetSnapshotBatch            = 1106024

	// process controller 1107XXX
	CCErrProcDeleteProc2Module   = 1107001
	CCErrProcCreateProc2Module   = 1107002
	CCErrProcSelectProc2Module   = 1107003
	CCErrProcCreateProcConf      = 1107004
	CCErrProcDeleteProcConf      = 1107005
	CCErrProcGetProcConf         = 1107006
	CCErrProcUpdateProcConf      = 1107007
	CCErrProcCreateInstanceModel = 1107008
	CCErrProcGetInstanceModel    = 1107009
	CCErrProcDeleteInstanceModel = 1107010
	CCErrProcDeleteProc2Template = 1107011
	CCErrProcCreateProc2Template = 1107012
	CCErrProcSelectProc2Template = 1107013

	// process server 1108XXX
	CCErrProcSearchDetailFaile          = 1108001
	CCErrProcBindToMoudleFaile          = 1108002
	CCErrProcUnBindToMoudleFaile        = 1108003
	CCErrProcSelectBindToMoudleFaile    = 1108004
	CCErrProcUpdateProcessFailed        = 1108005
	CCErrProcSearchProcessFailed        = 1108006
	CCErrProcDeleteProcessFailed        = 1108007
	CCErrProcCreateProcessFailed        = 1108008
	CCErrProcFieldValidFaile            = 1108009
	CCErrProcGetByApplicationIDFail     = 1108010
	CCErrProcGetByIP                    = 1108011
	CCErrProcOperateFaile               = 1108012
	CCErrProcBindWithModule             = 1108013
	CCErrProcDeleteTemplateFail         = 1108014
	CCErrProcUpdateTemplateFail         = 1108015
	CCErrProcSearchTemplateFail         = 1108016
	CCErrProcBindToTemplateFailed       = 1108017
	CCErrProcUnBindToTemplateFailed     = 1108018
	CCErrProcSelectBindToTemplateFailed = 1108019
	CCErrProcQueryTaskInfoFail          = 1108020
	CCErrProcQueryTaskWaitOPFail        = 1108021
	CCErrProcQueryTaskOPErrFail         = 1108022
	CCErrProcCreateTemplateFail         = 1108023

	CCErrProcGetServiceInstancesFailed                    = 1108024
	CCErrProcCreateServiceInstancesFailed                 = 1108025
	CCErrProcDeleteServiceInstancesFailed                 = 1108026
	CCErrProcGetProcessTemplatesFailed                    = 1108027
	CCErrProcGetProcessInstanceFailed                     = 1108028
	CCErrProcGetProcessInstanceRelationFailed             = 1108029
	CCErrProcDeleteServiceTemplateFailed                  = 1108030
	CCErrProcCreateProcessTemplateFailed                  = 1108031
	CCErrProcUpdateProcessTemplateFailed                  = 1108032
	CCErrProcGetProcessTemplateFailed                     = 1108033
	CCErrProcGetDefaultServiceCategoryFailed              = 1108034
	CCErrProcEditProcessInstanceCreateByTemplateForbidden = 1108035
	CCErrProcServiceTemplateAndCategoryNotCoincide        = 1108036
	CCErrProcModuleNotBindWithTemplate                    = 1108037
	CCErrCreateServiceInstanceWithWrongHTTPMethod         = 1108038
	CCErrCreateRawProcessInstanceOnTemplateInstance       = 1108039
	CCErrProcRemoveTemplateBindingOnModule                = 1108040
	CCErrProcReconstructServiceInstanceNameFailed         = 1108041

	CCErrProcUnbindModuleServiceTemplateDisabled = 1108042
	CCErrProcGetServiceCategoryFailed            = 1108043

	CCErrHostTransferFinalModuleConflict = 1108044

	CCErrSyncServiceInstanceByTemplateFailed = 1108045

	// audit log 1109XXX
	CCErrAuditSaveLogFailed      = 1109001
	CCErrAuditTakeSnapshotFailed = 1109002
	CCErrAuditSelectFailed       = 1109003
	CCErrAuditSelectTimeout      = 1109004

	// host server
	CCErrHostGetFail               = 1110001
	CCErrHostUpdateFail            = 1110002
	CCErrHostUpdateFieldFail       = 1110003
	CCErrHostCreateFail            = 1110004
	CCErrHostModifyFail            = 1110005
	CCErrHostDeleteFail            = 1110006
	CCErrHostFiledValdFail         = 1110007
	CCErrHostNotFound              = 1110008
	CCErrHostLength                = 1110009
	CCErrHostDetailFail            = 1111011
	CCErrHostSnap                  = 1110011
	CCErrHostFeildValidFail        = 1110012
	CCErrHostFavCreateFail         = 1110013
	CCErrHostEmptyFavName          = 1110014
	CCErrHostFavUpdateFail         = 1110015
	CCErrHostFavDeleteFail         = 1110016
	CCErrHostFavGetFail            = 1110017
	CCErrHostHisCreateFail         = 1110018
	CCErrHostHisGetFail            = 1110019
	CCErrHostCustomCreateFail      = 1110020
	CCErrHostCustomGetFail         = 1110021
	CCErrHostCustomGetDefaultFail  = 1110022
	CCErrHostNotINAPP              = 1110023
	CCErrHostNotINAPPFail          = 1110024
	CCErrHostDELResourcePool       = 1110025
	CCErrHostAddRelationFail       = 1110026
	CCErrHostMoveResourcePoolFail  = 1110027
	CCErrHostEditRelationPoolFail  = 1110028
	CCErrAddHostToModule           = 1110029
	CCErrAddHostToModuleFailStr    = 1110030
	CCErrTransferHostToArchivedApp = 1110031

	// hostserver api machinery new error code
	CCErrAddUserCustomQueryFailed       = 1110040
	CCErrUpdateUserCustomQueryFailed    = 1110041
	CCErrDeleteUserCustomQueryFailed    = 1110042
	CCErrSearchUserCustomQueryFailed    = 1110043
	CCErrGetUserCustomQueryDetailFailed = 1110044
	CCErrHostModuleConfigFailed         = 1110045
	CCErrHostGetSetFailed               = 1110046
	CCErrHostGetAPPFail                 = 1110047
	CCErrHostAPPNotFoundFail            = 1110048
	CCErrHostGetModuleFail              = 1110049
	CCErrHostAgentStatusFail            = 1110050
	// CCErrHostNotResourceFail The resource pool was not found"
	CCErrHostNotResourceFail = 1110051
	// CCErrHostBelongResourceFail The host is already in the resource pool
	CCErrHostBelongResourceFail = 1110052
	// CCErrHostGetResourceFail failed to get resource pool information, error message: %s
	CCErrHostGetResourceFail = 1110053
	// CCErrHostModuleNotExist get %s module not found
	CCErrHostModuleNotExist = 1110054
	// CCErrDeleteHostFromBusiness Delete the host under the business
	CCErrDeleteHostFromBusiness = 1110055
	// CCErrHostModuleConfigNotMatch hostID[%#v] not belong to business
	CCErrHostModuleConfigNotMatch = 1110056
	// CCErrHostModuleIDNotFoundORHasMultipleInnerModuleIDFailed Module does not exist or there are multiple built-in modules
	CCErrHostModuleIDNotFoundORHasMultipleInnerModuleIDFailed = 1110057
	CCErrHostSearchNeedObjectInstIDErr                        = 1110058
	CCErrHostSetNotExist                                      = 1110059
	CCErrHostSetNotBelongBusinessErr                          = 1110060
	CCErrHostModuleNotBelongBusinessErr                       = 1110061
	CCErrHostModuleNotBelongSetErr                            = 1110062
	CCErrHostPlatCloudNameIsrequired                          = 1110063
	CCErrHostPlatCloudNameAlreadyExist                        = 1110064
	CCErrHostFindManyCloudAreaAddHostCountFieldFail           = 1110065
	CCErrDeleteDefaultCloudAreaFail                           = 1110066
	CCErrHostFindManyCloudAreaAddSyncTaskIDsFieldFail         = 1110067

	// web 1111XXX
	CCErrWebFileNoFound                 = 1111001
	CCErrWebFileSaveFail                = 1111002
	CCErrWebOpenFileFail                = 1111003
	CCErrWebFileContentEmpty            = 1111004
	CCErrWebFileContentFail             = 1111005
	CCErrWebGetHostFail                 = 1111006
	CCErrWebCreateEXCELFail             = 1111007
	CCErrWebGetObjectFail               = 1111008
	CCErrWebGetAddNetDeviceResultFail   = 1111009
	CCErrWebGetAddNetPropertyResultFail = 1111010
	CCErrWebGetNetDeviceFail            = 1111011
	CCErrWebGetNetPropertyFail          = 1111012
	CCErrWebNeedFillinUsernamePasswd    = 1111013
	CCErrWebUsernamePasswdWrong         = 1111014
	CCErrWebNoUsernamePasswd            = 1111015
	CCErrWebUserinfoFormatWrong         = 1111016
	CCErrWebUnknownLoginVersion         = 1111017
	CCErrWebGetUsernameMapFail          = 1111018
	CCErrWebHostCheckFail               = 1111019

	// datacollection 1112xxx
	CCErrCollectNetDeviceCreateFail            = 1112000
	CCErrCollectNetDeviceGetFail               = 1112001
	CCErrCollectNetDeviceDeleteFail            = 1112002
	CCErrCollectObjIDNotNetDevice              = 1112003
	CCErrCollectNetPropertyCreateFail          = 1112004
	CCErrCollectNetPropertyGetFail             = 1112005
	CCErrCollectNetPropertyDeleteFail          = 1112006
	CCErrCollectNetDeviceObjPropertyNotExist   = 1112007
	CCErrCollectDeviceNotExist                 = 1112008
	CCErrCollectPeriodFormatFail               = 1112009
	CCErrCollectNetDeviceHasPropertyDeleteFail = 1112010
	CCErrCollectNetCollectorSearchFail         = 1112011
	CCErrCollectNetCollectorUpdateFail         = 1112012
	CCErrCollectNetCollectorDiscoverFail       = 1112013
	CCErrCollectNetReportSearchFail            = 1112014
	CCErrCollectNetReportConfirmFail           = 1112015
	CCErrCollectNetHistorySearchFail           = 1112016
	CCErrCollectNetDeviceUpdateFail            = 1112017
	CCErrCollectNetPropertyUpdateFail          = 1112018

	// coreservice 1113xxx
	// CCErrorModelAttributeGroupHasSomeAttributes the group has some attributes
	CCErrCoreServiceModelAttributeGroupHasSomeAttributes = 1113001

	// CCErrCoreServiceHostNotBelongBusiness hostID [%#v] does not belong of  businessID [%d]
	CCErrCoreServiceHostNotBelongBusiness = 1113002
	// CCErrCoreServiceHostNotExist hostID [%#v] does not exist
	CCErrCoreServiceHostNotExist = 1113003
	// ModuleID [%#v] has not belong of  businessID [%d]
	CCErrCoreServiceHasModuleNotBelongBusiness = 1113004
	// CCErrCoreServiceModuleContainDefaultModuleErr  translate host to multiple module not contain default module
	CCErrCoreServiceModuleContainDefaultModuleErr = 1113005
	// CCErrCoreServiceBusinessNotExist Business [%#v] does not exist
	CCErrCoreServiceBusinessNotExist = 1113006
	// CCErrCoreServiceDefaultModuleNotExist Business [%#v] default module does not exist
	CCErrCoreServiceDefaultModuleNotExist = 1113007
	// CCErrCoreServiceModuleNotDefaultModuleErr   businessID [%d] of moduleID[%d] not default module
	CCErrCoreServiceModuleNotDefaultModuleErr = 1113008
	// CCErrCoreServiceTransferHostModuleErr   transfer module host config error. error detail in return data
	CCErrCoreServiceTransferHostModuleErr = 1113009
	// CCErrCoreServiceEventPushEventFailed failed to sent event
	CCErrCoreServiceEventPushEventFailed = 1113010

	// 禁止释放(转移到空闲机/故障机/资源池)已关联到服务实例的主机
	CCErrCoreServiceForbiddenReleaseHostReferencedByServiceInstance = 1113011

	CCErrHostRemoveFromDefaultModuleFailed                                    = 1113012
	CCErrCoreServiceTransferToDefaultModuleUseWrongMethod                     = 1113013
	CCErrCoreServiceModuleWithoutServiceTemplateCouldNotCreateServiceInstance = 1113014
	CCErrCoreServiceModuleNotFound                                            = 1113015
	CCErrCoreServiceInstanceAlreadyExist                                      = 1113016
	CCErrCoreServiceServiceCategoryNameDuplicated                             = 1113017
	CCErrCoreServiceModuleAndServiceInstanceTemplateNotCoincide               = 1113018
	CCErrCoreServiceProcessNameDuplicated                                     = 1113019
	CCErrCoreServiceFuncNameDuplicated                                        = 1113020
	CCErrCoreServiceModuleNotBoundWithTemplate                                = 1113021
	CCErrCoreServiceShouldNotRemoveProcessCreateByTemplate                    = 1113022
	// CCErrCoreServiceDeleteMultpleObjectIDRecordErr 删除多个模型中的%s数据
	CCErrCoreServiceDeleteMultpleObjectIDRecordErr = 1113023
	// CCErrCoreServiceDeleteMultipleObjectIDRecordErr 不允许删除在唯一校验中的字段
	CCErrCoreServiceNotAllowUniqueAttr = 1113024
	// CCErrCoreServiceNotUpdatePredefinedAttrErr 修改不允许修改的属性的描述
	CCErrCoreServiceNotUpdatePredefinedAttrErr = 1113025
	// CCErrCoreServiceNotAllowAddRequiredFieldErr 模型[%s]不允许新加必填字段
	CCErrCoreServiceNotAllowAddRequiredFieldErr = 1113026
	// CCErrCoreServiceNotAllowAddRequiredFieldErr 模型[%s]不允许修改必填字段
	CCErrCoreServiceNotAllowChangeRequiredFieldErr = 1113027
	// CCErrCoreServiceNotAllowAddFieldErr 模型[%s]不允许新加字段
	CCErrCoreServiceNotAllowAddFieldErr = 1113028
	// CCErrCoreServiceNotAllowDeleteErr 模型【%s】不允许删除
	CCErrCoreServiceNotAllowDeleteErr = 1113029
	// CCErrCoreServiceModelHasInstanceErr 模型下有示例数据
	CCErrCoreServiceModelHasInstanceErr = 1113030
	// CCErrCoreServiceModelHasAssociationErr 模型与其他模型有关联关系
	CCErrCoreServiceModelHasAssociationErr           = 1113031
	CCErrCoreServiceOnlyNodeServiceCategoryAvailable = 1113032
	// SearchTopoTreeScanTooManyData means hit too many data, we return directly.
	SearchTopoTreeScanTooManyData = 1113033

	// CCERrrCoreServiceUniqueRuleExist 模型唯一校验规则已经存在
	CCERrrCoreServiceSameUniqueCheckRuleExist = 1113050
	CCERrrCoreServiceConcurrent               = 1113051

	// CCErrCoreServiceResourceDirectoryNotExistErr 资源池目录不存在
	CCErrCoreServiceResourceDirectoryNotExistErr = 1113033
	// CCErrCoreServiceHostNotUnderAnyResourceDirectory 主机不在任意资源池目录下
	CCErrCoreServiceHostNotUnderAnyResourceDirectory = 11130034

	// synchronize data core service  11139xx
	CCErrCoreServiceSyncError = 1113900
	// CCErrCoreServiceSyncDataClassifyNotExistError %s type data synchronization, data of the same type %s does not exist
	CCErrCoreServiceSyncDataClassifyNotExistError = 1113901

	// synchronize_server 1114xxx

	CCErrSynchronizeError = 1113903

	// operation_server 1116xxx
	CCErrOperationBizModuleHostAmountFail = 1116001
	CCErrOperationNewAddStatisticFail     = 1116002
	CCErrOperationChartAlreadyExist       = 1116003
	CCErrOperationDeleteChartFail         = 1116004
	CCErrOperationSearchChartFail         = 1116005
	CCErrOperationUpdateChartFail         = 1116006
	CCErrOperationGetChartDataFail        = 1116007
	CCErrOperationUpdateChartPositionFail = 1116008

	// task_server 1117xxx
	// CCErrTaskNotFound task not found
	CCErrTaskNotFound = 1117001
	// CCErrTaskSubTaskNotFound sub task not found
	CCErrTaskSubTaskNotFound = 1117002
	// CCErrTaskStatusNotAllowChangeTo task not allow status change to xx
	CCErrTaskStatusNotAllowChangeTo = 1117003
	// CCErrTaskErrResponseEmtpyFail error response empty
	CCErrTaskErrResponseEmtpyFail = 1117004
	CCErrTaskLockedTaskFail       = 1117005
	CCErrTaskUnLockedTaskFail     = 1117006
	CCErrTaskListTaskFail         = 1117007

	// cloud_server 1118xxx
	// CCErrCloudVendorNotSupport cloud vendor not support
	CCErrCloudVendorNotSupport                = 1118001
	CCErrCloudAccountNameAlreadyExist         = 1118002
	CCErrCloudValidAccountParamFail           = 1118003
	CCErrCloudAccountIDNoExistFail            = 1118004
	CCErrCloudSyncTaskNameAlreadyExist        = 1118005
	CCErrCloudValidSyncTaskParamFail          = 1118006
	CCErrCloudVpcIDIsRequired                 = 1118007
	CCErrCloudVendorInterfaceCalledFailed     = 1118008
	CCErrCloudAccountSecretIDAlreadyExist     = 1118009
	CCErrCloudTaskAlreadyExistInAccount       = 1118010
	CCErrCloudAccoutIDSecretWrong             = 1118011
	CCErrCloudHttpRequestTimeout              = 1118012
	CCErrCloudVpcGetFail                      = 1118013
	CCErrCloudRegionGetFail                   = 1118014
	CCErrCloudSyncDirNoChosen                 = 1118015
	CCErrCloudSyncDirNoExist                  = 1118016
	CCErrCloudIDNoProvided                    = 1118017
	CCErrCloudIDNoExist                       = 1118018
	CCErrDefaultCloudIDProvided               = 1118019
	CCErrCloudAccountCreateFail               = 1118020
	CCErrGetCloudAccountConfBatchFailed       = 1118021
	CCErrDeleteDestroyedHostRelatedFailed     = 1118022
	CCErrCloudAccountDeletedFailedForSyncTask = 1118023

	/** TODO: 以下错误码需要改造 **/

	// json
	CCErrCommJsonDecode    = 3001
	CCErrCommJsonDecodeStr = "json decode failed!"
	CCErrCommJsonEncode    = 3002
	CCErrCommJsonEncodeStr = "json encode failed!"

	JsonMarshalErr    = 9000
	JsonMarshalErrStr = "json marshal error"
)
