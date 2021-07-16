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
	"configcenter/src/ac/extensions"
	"configcenter/src/apimachinery"
	"configcenter/src/scene_server/topo_server/logics/model"
	"configcenter/src/scene_server/topo_server/logics/operation"
)

// Logics provides management interface for operations of model and instance and related resources like association
type Logics interface {
	ClassificationOperation() model.ClassificationOperationInterface
	UniqueOperation() model.UniqueOperationInterface
	ObjectOperation() model.ObjectOperationInterface
	AuditOperation() operation.AuditOperationInterface
	IdentifierOperation() operation.IdentifierOperationInterface
}

type logics struct {
	classification model.ClassificationOperationInterface
	unique         model.UniqueOperationInterface
	object         model.ObjectOperationInterface
	audit          operation.AuditOperationInterface
	identifier     operation.IdentifierOperationInterface
}

// New create a logics manager
func New(client apimachinery.ClientSetInterface, authManager *extensions.AuthManager) Logics {
	classificationOperation := model.NewClassificationOperation(client, authManager)
	uniqueOperation := model.NewUniqueOperation(client, authManager)
	objectOperation := model.NewObjectOperation(client, authManager)
	AuditOperation := operation.NewAuditOperation(client)
	IdentifierOperation := operation.NewIdentifier(client)

	return &logics{
		classification: classificationOperation,
		unique:         uniqueOperation,
		object:         objectOperation,
		audit:          AuditOperation,
		identifier:     IdentifierOperation,
	}
}

func (c *logics) ClassificationOperation() model.ClassificationOperationInterface {
	return c.classification
}

func (c *logics) UniqueOperation() model.UniqueOperationInterface {
	return c.unique
}

func (c *logics) ObjectOperation() model.ObjectOperationInterface {
	return c.object
}

func (c *logics) AuditOperation() operation.AuditOperationInterface {
	return c.audit
}

func (c *logics) IdentifierOperation() operation.IdentifierOperationInterface {
	return c.identifier
}
