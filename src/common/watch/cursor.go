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

package watch

import (
	"bytes"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"strconv"

	"configcenter/src/common"
	"configcenter/src/common/blog"
	"configcenter/src/storage/stream/types"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// this cursor means there is no event occurs.
// we send this cursor to our the watcher and if we
// received a NoEventCursor, then we need to fetch event
// from the head cursor
var NoEventCursor string

func init() {
	no := Cursor{
		Type:        NoEvent,
		ClusterTime: types.TimeStamp{Sec: 1, Nano: 1},
		Oid:         "5ea6d3f394c1f5d986e9bd86",
		Oper:        types.OperType("noEvent"),
	}
	cursor, err := no.Encode()
	if err != nil {
		panic("initial NoEventCursor failed")
	}
	// cursor should be:
	// MQ0xDTVlYTZkM2YzOTRjMWY1ZDk4NmU5YmQ4Ng1ub0V2ZW50DTENMQ==
	NoEventCursor = cursor
}

type CursorType string

const (
	NoEvent                 CursorType = "no_event"
	UnknownType             CursorType = "unknown"
	Host                    CursorType = "host"
	ModuleHostRelation      CursorType = "host_relation"
	Biz                     CursorType = "biz"
	Set                     CursorType = "set"
	Module                  CursorType = "module"
	ObjectBase              CursorType = "object_instance"
	Process                 CursorType = "process"
	ProcessInstanceRelation CursorType = "process_instance_relation"
	// a mixed event type, which contains host, host relation, process events etc.
	// and finally converted to host identifier event.
	HostIdentifier CursorType = "host_identifier"
	// MainlineInstance specified for mainline instance event watch, filtered from object instance events
	MainlineInstance CursorType = "mainline_instance"
)

func (ct CursorType) ToInt() int {
	switch ct {
	case NoEvent:
		return 1
	case Host:
		return 2
	case ModuleHostRelation:
		return 3
	case Biz:
		return 4
	case Set:
		return 5
	case Module:
		return 6
	case ObjectBase:
		return 8
	case Process:
		return 9
	case ProcessInstanceRelation:
		return 10
	case HostIdentifier:
		return 11
	case MainlineInstance:
		return 12
	default:
		return -1
	}
}

func (ct *CursorType) ParseInt(typ int) {
	switch typ {
	case 1:
		*ct = NoEvent
	case 2:
		*ct = Host
	case 3:
		*ct = ModuleHostRelation
	case 4:
		*ct = Biz
	case 5:
		*ct = Set
	case 6:
		*ct = Module
	case 8:
		*ct = ObjectBase
	case 9:
		*ct = Process
	case 10:
		*ct = ProcessInstanceRelation
	case 11:
		*ct = HostIdentifier
	case 12:
		*ct = MainlineInstance
	default:
		*ct = UnknownType
	}
}

// ListCursorTypes returns all support CursorTypes.
func ListCursorTypes() []CursorType {
	return []CursorType{Host, ModuleHostRelation, Biz, Set, Module, ObjectBase, Process,
		ProcessInstanceRelation, HostIdentifier, MainlineInstance}
}

// ListEventCallbackCursorTypes returns all support CursorTypes for event callback.
func ListEventCallbackCursorTypes() []CursorType {
	return []CursorType{Host, ModuleHostRelation, Biz, Set, Module, ObjectBase, Process, ProcessInstanceRelation}
}

// Cursor is a self-defined token which is corresponding to the mongodb's resume token.
// cursor has a unique and 1:1 relationship with mongodb's resume token.
type Cursor struct {
	Type        CursorType
	ClusterTime types.TimeStamp
	// a random hex string to avoid the caller to generated a self-defined cursor.
	Oid  string
	Oper types.OperType
	// UniqKey is an optional key which is used to ensure that a cursor is unique among a same resources(
	// as is Type field).
	// This key is used when the upper fields can not describe a unique cursor. such as the common object instance
	// event, all these instance event all have a same cursor type, as is object instance. but the instance id is
	// unique which can be used as a unique key, and is REENTRANT when a retry operation happens which is
	// **VERY IMPORTANT**.
	UniqKey string
}

const cursorVersion = "1"

func (c Cursor) Encode() (string, error) {
	if c.Type == "" {
		return "", errors.New("unsupported type")
	}

	if c.ClusterTime.Sec == 0 {
		return "", errors.New("invalid cluster time sec")
	}

	if c.Oid == "" {
		return "", errors.New("invalid oid")
	}

	if c.Oper == "" {
		return "", errors.New("unsupported operation type")
	}

	sec := strconv.FormatUint(uint64(c.ClusterTime.Sec), 10)
	nano := strconv.FormatUint(uint64(c.ClusterTime.Nano), 10)
	pool := bytes.Buffer{}
	// version field.
	pool.WriteString(cursorVersion)
	pool.WriteByte('\r')

	// type filed.
	if c.Type.ToInt() < 0 {
		return "", errors.New("unsupported cursor type")
	}

	pool.WriteString(strconv.Itoa(c.Type.ToInt()))
	pool.WriteByte('\r')

	// oid field.
	pool.WriteString(c.Oid)
	pool.WriteByte('\r')

	// operation type field
	pool.WriteString(string(c.Oper))
	pool.WriteByte('\r')

	// cluster time sec field.
	pool.WriteString(sec)
	pool.WriteByte('\r')

	// cluster time nano field
	pool.WriteString(nano)
	pool.WriteByte('\r')

	// unique key field
	pool.WriteString(c.UniqKey)

	return base64.StdEncoding.EncodeToString(pool.Bytes()), nil
}

func (c *Cursor) Decode(cur string) error {
	byt, err := base64.StdEncoding.DecodeString(cur)
	if err != nil {
		return fmt.Errorf("decode cursor, but base64 decode failed, err: %v", err)
	}

	elements := make([]string, 0)
	pool := bytes.NewBuffer(byt)

	ele := make([]byte, 0)
	for {
		b, err := pool.ReadByte()
		if err != nil {
			if err != io.EOF {
				return err
			}
			// to the end
			elements = append(elements, string(ele))
			break
		}
		if b == '\r' {
			elements = append(elements, string(ele))
			ele = ele[:0]
		} else {
			ele = append(ele, b)
		}
	}

	// at least have 6 fields, UniqKey is an optional field.
	if len(elements) < 6 {
		return errors.New("invalid cursor string")
	}

	if elements[0] != cursorVersion {
		return fmt.Errorf("decode cursor, but got invalid cursor version: %s", elements[0])
	}

	typ, err := strconv.Atoi(elements[1])
	if err != nil {
		return fmt.Errorf("got invalid type: %s", elements[1])
	}
	cursorType := CursorType("")
	cursorType.ParseInt(typ)
	c.Type = cursorType

	_, err = primitive.ObjectIDFromHex(elements[2])
	if err != nil {
		return fmt.Errorf("got invalid oid: %s, err: %v", elements[2], err)
	}
	c.Oid = elements[2]

	if elements[3] == "" {
		return fmt.Errorf("decode cursor, but got empty operation type")
	}
	c.Oper = types.OperType(elements[3])

	sec, err := strconv.ParseUint(elements[4], 10, 64)
	if err != nil {
		return fmt.Errorf("got invalid sec field %s, err: %v", elements[4], err)
	}
	c.ClusterTime.Sec = uint32(sec)

	nano, err := strconv.ParseUint(elements[5], 10, 64)
	if err != nil {
		return fmt.Errorf("got invalid nano field %s, err: %v", elements[5], err)
	}
	c.ClusterTime.Nano = uint32(nano)

	// cause unique key is an optional key.
	if len(elements) >= 7 {
		c.UniqKey = elements[6]
	}

	return nil
}

func GetEventCursor(coll string, e *types.Event, instID int64) (string, error) {
	curType := UnknownType
	switch coll {
	case common.BKTableNameBaseHost:
		curType = Host
	case common.BKTableNameModuleHostConfig:
		curType = ModuleHostRelation
	case common.BKTableNameBaseApp:
		curType = Biz
	case common.BKTableNameBaseSet:
		curType = Set
	case common.BKTableNameBaseModule:
		curType = Module
	case common.BKTableNameBaseInst:
		curType = ObjectBase
	case common.BKTableNameMainlineInstance:
		curType = MainlineInstance
	case common.BKTableNameBaseProcess:
		curType = Process
	case common.BKTableNameProcessInstanceRelation:
		curType = ProcessInstanceRelation
	default:
		blog.Errorf("unsupported cursor type collection: %s, oid: %s", e.ID())
		return "", fmt.Errorf("unsupported cursor type collection: %s", coll)
	}

	hCursor := &Cursor{
		Type:        curType,
		ClusterTime: e.ClusterTime,
		Oid:         e.Oid,
		Oper:        e.OperationType,
	}

	if curType == ObjectBase || curType == MainlineInstance {
		if instID <= 0 {
			return "", errors.New("invalid instance id")
		}

		// add unique key for common object instance.
		hCursor.UniqKey = strconv.FormatInt(instID, 10)
	}

	hCursorEncode, err := hCursor.Encode()
	if err != nil {
		blog.Errorf("encode node cursor failed, err: %v, oid: %s", err, e.Oid)
		return "", err
	}

	return hCursorEncode, nil
}
