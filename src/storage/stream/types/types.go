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

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"time"

	"github.com/tidwall/gjson"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/bsontype"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/x/bsonx"
)

type OperType string

const (
	// reference doc:
	// https://docs.mongodb.com/manual/reference/change-events/#change-events
	// Document operation type
	Insert  OperType = "insert"
	Delete  OperType = "delete"
	Replace OperType = "replace"
	Update  OperType = "update"

	// collection operation type.
	Drop   OperType = "drop"
	Rename OperType = "rename"

	// dropDatabase event occurs when a database is dropped.
	DropDatabase OperType = "dropDatabase"

	// For change streams opened up against a collection, a drop event, rename event,
	// or dropDatabase event that affects the watched collection leads to an invalidate event.
	Invalidate OperType = "invalidate"

	// Lister OperType is a self defined type, which is represent this operation comes from
	// a list watcher's find operations, it does not really come form the mongodb's change event.
	Lister OperType = "lister"
	// ListerDone OperType is a self defined type, which means that the list operation has already finished,
	// and the watch events starts. this OperType send only for once.
	// Note: it's only used in the ListWatch Operation.
	ListDone OperType = "listerDone"
)

type ListOptions struct {
	// Filter helps you filter out which kind of data's change event you want
	// to receive, such as the filter :
	// {"bk_obj_id":"biz"} means you can only receives the data that has this kv.
	// Note: the filter's key must be a exist document key filed in the collection's
	// document
	Filter map[string]interface{}

	// list the documents only with these fields.
	Fields []string

	// EventStruct is the point data struct that the event decoded into.
	// Note: must be a point value.
	EventStruct interface{}

	// Collection defines which collection you want you watch.
	Collection string

	// Step defines the list step when the client try to list all the data defines in the
	// namespace. default value is `DefaultListStep`, value range [200,2000]
	PageSize *int
}

func (opts *ListOptions) CheckSetDefault() error {
	if reflect.ValueOf(opts.EventStruct).Kind() != reflect.Ptr ||
		reflect.ValueOf(opts.EventStruct).IsNil() {
		return fmt.Errorf("invalid EventStruct field, must be a pointer and not nil")
	}

	if opts.PageSize != nil {
		if *opts.PageSize < 0 || *opts.PageSize > 2000 {
			return fmt.Errorf("invalid page size, range is [200,2000]")
		}
	} else {
		opts.PageSize = &defaultListPageSize
	}

	if len(opts.Collection) == 0 {
		return errors.New("invalid Namespace field, database and collection can not be empty")
	}
	return nil
}

type Options struct {
	// reference doc:
	// https://docs.mongodb.com/manual/reference/method/db.collection.watch/#change-stream-with-full-document-update-lookup
	// default value is true
	MajorityCommitted *bool

	// The maximum amount of time in milliseconds the server waits for new
	// data changes to report to the change stream cursor before returning
	// an empty batch.
	// default value is 1000ms
	MaxAwaitTime *time.Duration

	// OperationType describe which kind of operation you want to watch,
	// such as a "insert" operation or a "replace" operation.
	// If you don't set, it will means watch  all kinds of operations.
	OperationType *OperType

	// Filter helps you filter out which kind of data's change event you want
	// to receive, such as the filter :
	// {"bk_obj_id":"biz"} means you can only receives the data that has this kv.
	// Note: the filter's key must be a exist document key filed in the collection's
	// document
	Filter map[string]interface{}

	// CollectionFilter helps you filter out which kind of collection's change event you want to receive,
	// such as the filter : {"$regex":"^cc_ObjectBase"} means you can only receive events from collections
	// starts with the prefix cc_ObjectBase
	CollectionFilter interface{}

	// EventStruct is the point data struct that the event decoded into.
	// Note: must be a point value.
	EventStruct interface{}

	// Collection defines which collection you want you watch.
	Collection string

	// StartAfterToken describe where you want to watch the event.
	// Note: the returned event doesn't contains the token represented,
	// and will returns event just after this token.
	// If StartAfterToken and StartAtTime is set at the same time, then StartAfterToken is used only.
	StartAfterToken *EventToken

	// Ensures that this watch will provide events that occurred after this timestamp.
	StartAtTime *TimeStamp

	// WatchFatalErrorCallback the function to be called when watch failed with a fatal error
	// reset the resume token and set the start time for next watch in case it use the mistaken token again
	WatchFatalErrorCallback func(startAtTime TimeStamp) error `json:"-"`

	// Fields defines which fields will be returned along with the events
	// this is optional, if not set, all the fields will be returned.
	Fields []string
}

var defaultMaxAwaitTime = time.Second

// CheckSet check the legal of each option, and set the default value
func (opts *Options) CheckSetDefault() error {
	if reflect.ValueOf(opts.EventStruct).Kind() != reflect.Ptr ||
		reflect.ValueOf(opts.EventStruct).IsNil() {
		return fmt.Errorf("invalid EventStruct field, must be a pointer and not nil")
	}

	if opts.MajorityCommitted == nil {
		t := true
		opts.MajorityCommitted = &t
	}

	if opts.MaxAwaitTime == nil {
		opts.MaxAwaitTime = &defaultMaxAwaitTime
	}

	if len(opts.Collection) == 0 && opts.CollectionFilter == nil {
		return errors.New("invalid Namespace field, database and collection can not be empty")
	}
	return nil
}

type TimeStamp struct {
	// the most significant 32 bits are a time_t value (seconds since the Unix epoch)
	Sec uint32 `json:"sec" bson:"sec"`
	// the least significant 32 bits are an incrementing ordinal for operations within a given second.
	Nano uint32 `json:"nano" bson:"nano"`
}

func (t TimeStamp) String() string {
	return time.Unix(int64(t.Sec), int64(t.Nano)).Format("2006-01-02/15:04:05")
}

func (t TimeStamp) MarshalBSONValue() (bsontype.Type, []byte, error) {
	return bsonx.Time(time.Unix(int64(t.Sec), int64(t.Nano))).MarshalBSONValue()
}

type timeStampCopy struct {
	Sec  uint32 `json:"sec" bson:"sec"`
	Nano uint32 `json:"nano" bson:"nano"`
}

func (t *TimeStamp) UnmarshalBSONValue(typo bsontype.Type, raw []byte) error {
	if typo == bsontype.DateTime {
		timeStamp := bson.RawValue{Type: bsontype.DateTime, Value: raw}.Time()
		t.Sec = uint32(timeStamp.Unix())
		t.Nano = uint32(timeStamp.Nanosecond())
		return nil
	}

	tt := timeStampCopy{}
	err := bson.Unmarshal(raw, &tt)
	if err != nil {
		return err
	}

	t.Sec = tt.Sec
	t.Nano = tt.Nano
	return nil
}

type WatchOptions struct {
	Options
}

var defaultListPageSize = 1000

type ListWatchOptions struct {
	Options

	// Step defines the list step when the client try to list all the data defines in the
	// namespace. default value is `DefaultListStep`, value range [200,2000]
	PageSize *int
}

func (lw *ListWatchOptions) CheckSetDefault() error {
	if err := lw.Options.CheckSetDefault(); err != nil {
		return err
	}

	if lw.PageSize != nil {
		if *lw.PageSize < 0 || *lw.PageSize > 2000 {
			return fmt.Errorf("invalid page size, range is [200,2000]")
		}
	} else {
		lw.PageSize = &defaultListPageSize
	}

	return nil
}

const DefaultEventChanSize = 100

type Watcher struct {
	EventChan <-chan *Event
}

type Event struct {
	// Oid represent the unique document key filed "_id"
	Oid           string
	Document      interface{}
	DocBytes      []byte
	OperationType OperType
	Collection    string

	// The timestamp from the oplog entry associated with the event.
	ClusterTime TimeStamp

	// event token for resume after.
	Token EventToken

	// changed fields details in this event, describes which fields is updated or removed.
	ChangeDesc *ChangeDescription
}

type ChangeDescription struct {
	// updated details's value is the current value, not the previous value.
	UpdatedFields map[string]interface{}
	RemovedFields []string
}

func (e *Event) String() string {
	return fmt.Sprintf("oper: %s, oid: %s, doc: %s", e.OperationType, e.Oid, e.DocBytes)
}

// ID returns a event's  logical unique identity id in a collection
func (e *Event) ID() string {
	return fmt.Sprintf("%s-%d-%d", e.Oid, e.ClusterTime.Sec, e.ClusterTime.Nano)
}

// mongodb change stream token, which represent a event's identity.
type EventToken struct {
	// Hex value of document's _id
	Data string `bson:"_data"`
}

// reference:
// https://docs.mongodb.com/manual/reference/change-events/
type EventStream struct {
	Token         EventToken          `bson:"_id"`
	OperationType OperType            `bson:"operationType"`
	ClusterTime   primitive.Timestamp `bson:"clusterTime"`
	Namespace     Namespace           `bson:"ns"`
	DocumentKey   Key                 `bson:"documentKey"`
	UpdateDesc    UpdateDescription   `bson:"updateDescription"`
}

type Key struct {
	// the unique document id, as is "_id"
	ID primitive.ObjectID `bson:"_id"`
}

type Namespace struct {
	Database   string `bson:"db"`
	Collection string `bson:"coll"`
}

type UpdateDescription struct {
	// document's fields which is updated in a change stream
	UpdatedFields map[string]interface{} `json:"updatedFields" bson:"updatedFields"`
	// document's fields which is removed in a change stream
	RemovedFields []string `json:"removedFields" bson:"removedFields"`
}

// EventDetail event document detail and changed fields
type EventDetail struct {
	Detail        JsonString             `json:"detail"`
	UpdatedFields map[string]interface{} `json:"update_fields"`
	RemovedFields []string               `json:"deleted_fields"`
}

type JsonString string

func (j JsonString) MarshalJSON() ([]byte, error) {
	if j == "" {
		j = "{}"
	}
	return []byte(j), nil
}

func (j *JsonString) UnmarshalJSON(b []byte) error {
	*j = JsonString(b)
	return nil
}

// GetEventDetail get event document detail, returns EventDetail's detail field
func GetEventDetail(detailStr *string) *string {
	if detailStr == nil {
		return new(string)
	}
	detail := gjson.Get(*detailStr, "detail").Raw
	return &detail
}

type TokenHandler interface {
	SetLastWatchToken(ctx context.Context, token string) error
	GetStartWatchToken(ctx context.Context) (token string, err error)
}

type LoopOptions struct {
	// name of this loop watch
	Name         string
	WatchOpt     *WatchOptions
	TokenHandler TokenHandler
	RetryOptions *RetryOptions

	// StopNotifier is used when user need to stop loop events and release related resources.
	// It's a optional option. when it's not set(as is nil), then the loop will not exit forever.
	// Otherwise, user can use it to stop loop events.
	// When a user want to stop the loop, the only thing that a user need to do is to just
	// **close** this stop notifier channel.
	// Attention:
	// Close this notifier channel is the only way to stop loop correctly.
	// Do not send data to this channel.
	StopNotifier <-chan struct{}
}

type LoopOneOptions struct {
	LoopOptions
	EventHandler *OneHandler
}

func (lo *LoopOneOptions) Validate() error {
	if len(lo.Name) == 0 {
		return errors.New("loop watch should have a name")
	}

	if lo.TokenHandler == nil {
		return errors.New("token handler is nil")
	}

	if lo.EventHandler == nil {
		return errors.New("event handler is nil")
	}

	if lo.EventHandler.DoAdd == nil || lo.EventHandler.DoUpdate == nil || lo.EventHandler.DoDelete == nil {
		return errors.New("invalid event handler options with add, update or delete is nil")
	}

	if lo.RetryOptions != nil {
		if lo.RetryOptions.MaxRetryCount <= 0 {
			lo.RetryOptions.MaxRetryCount = defaultRetryCount
		}

		if lo.RetryOptions.RetryDuration == 0 {
			lo.RetryOptions.RetryDuration = defaultRetryDuration
		}

		if lo.RetryOptions.RetryDuration < 500*time.Millisecond {
			return errors.New("invalid retry duration, can not less than 500ms")
		}
	} else {
		lo.RetryOptions = &RetryOptions{
			MaxRetryCount: defaultRetryCount,
			RetryDuration: defaultRetryDuration,
		}
	}

	if lo.LoopOptions.StopNotifier == nil {
		// if not set, then set never stop loop as default
		lo.LoopOptions.StopNotifier = make(<-chan struct{})
	}

	return nil
}

type LoopBatchOptions struct {
	LoopOptions
	EventHandler *BatchHandler
	// describe how many events in a batch.
	BatchSize int
}

const (
	defaultBatchSize     = 200
	defaultRetryCount    = 10
	defaultRetryDuration = 1 * time.Second
)

func (lo *LoopBatchOptions) Validate() error {
	if len(lo.Name) == 0 {
		return errors.New("loop watch should have a name")
	}

	if lo.TokenHandler == nil {
		return errors.New("token handler is nil")
	}

	if lo.EventHandler == nil {
		return errors.New("event handler is nil")
	}

	if lo.EventHandler.DoBatch == nil {
		return errors.New("batch handler is nil")
	}

	if lo.BatchSize == 0 {
		lo.BatchSize = defaultBatchSize
	}

	if lo.RetryOptions != nil {
		if lo.RetryOptions.MaxRetryCount <= 0 {
			lo.RetryOptions.MaxRetryCount = defaultRetryCount
		}

		if lo.RetryOptions.RetryDuration == 0 {
			lo.RetryOptions.RetryDuration = defaultRetryDuration
		}

		if lo.RetryOptions.RetryDuration < 200*time.Millisecond {
			return errors.New("invalid retry duration, can not less than 200ms")
		}
	} else {
		lo.RetryOptions = &RetryOptions{
			MaxRetryCount: defaultRetryCount,
			RetryDuration: defaultRetryDuration,
		}
	}

	if lo.LoopOptions.StopNotifier == nil {
		// if not set, then set never stop loop as default
		lo.LoopOptions.StopNotifier = make(<-chan struct{})
	}

	return nil
}

type RetryOptions struct {
	// the maximum count to retry, when a event is handled failed.
	MaxRetryCount int

	// the duration between each retry.
	// default
	RetryDuration time.Duration
}

type OneHandler struct {
	// retry decide whether event(s) is required to retry after
	// a event is handled failed
	DoAdd    func(event *Event) (retry bool)
	DoUpdate func(event *Event) (retry bool)
	DoDelete func(event *Event) (retry bool)
}

type BatchHandler struct {
	// DoBatch means handle the event with batch,
	// when this is enabled, then DoAdd, DoUpdate, DoDelete will be ignored
	DoBatch func(es []*Event) (retry bool)
}
