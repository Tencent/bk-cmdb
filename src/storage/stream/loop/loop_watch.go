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

package loop

import (
	"context"
	"time"

	"configcenter/src/apimachinery/discovery"
	"configcenter/src/common/blog"
	"configcenter/src/storage/stream/event"
	"configcenter/src/storage/stream/types"
)

func NewLoopWatch(streamW *event.Event, isMaster discovery.ServiceManageInterface) (*LoopsWatch, error) {
	loops := &LoopsWatch{
		streamWatch: streamW,
		isMaster:    isMaster,
	}

	return loops, nil
}

type LoopsWatch struct {
	streamWatch *event.Event
	isMaster    discovery.ServiceManageInterface
}

// WithOne allows users to watch events one by one.
func (lw *LoopsWatch) WithOne(opts *types.LoopOneOptions) error {
	if err := opts.Validate(); err != nil {
		blog.Errorf("run loop watch, but option is invalid, err: %v", err)
		return err
	}

	startToken, err := opts.TokenHandler.GetStartWatchToken(context.Background())
	if err != nil {
		blog.Errorf("%s job, run loop watch %s, but get start token failed, err: %v", opts.Name,
			opts.WatchOpt.Collection, err)
		return err
	}

	// update the start token.
	if len(startToken) != 0 {
		opts.WatchOpt.StartAfterToken = &types.EventToken{Data: startToken}
	}

	var cancel func()
	var cancelCtx context.Context
	cancelCtx, cancel = context.WithCancel(context.Background())

	watcher, err := lw.streamWatch.Watch(cancelCtx, opts.WatchOpt)
	if err != nil {
		blog.Errorf("%s job, run loop, but watch failed, err: %v", opts.Name, err)
		cancel()
		return err
	}

	retrySignal := make(chan struct{})
	retryObserver := &retryHandler{
		retryCounter: 0,
		maxRetryCnt:  opts.RetryOptions.MaxRetryCount,
	}

	handler := func(newCancelCtx context.Context, newWatcher *types.Watcher, newRetrySignal chan struct{}) {
		lw.tryLoopWithOne(newCancelCtx, newWatcher, newRetrySignal, retryObserver, opts)
	}

	// start watch retry signal
	go lw.watchRetry(cancel, retrySignal, &opts.LoopOptions, handler)

	// start loop with events one by one.
	go lw.tryLoopWithOne(cancelCtx, watcher, retrySignal, retryObserver, opts)

	return nil
}

// WithBatch allows users to watch events with batch.
func (lw *LoopsWatch) WithBatch(opts *types.LoopBatchOptions) error {
	if err := opts.Validate(); err != nil {
		blog.Errorf("run loop watch batch, but option is invalid, err: %v", err)
		return err
	}

	startToken, err := opts.TokenHandler.GetStartWatchToken(context.Background())
	if err != nil {
		blog.Errorf("%s job, run loop watch batch %s, but get start token failed, err: %v", opts.Name,
			opts.WatchOpt.Collection, err)
		return err
	}

	// update the start token.
	if len(startToken) != 0 {
		opts.WatchOpt.StartAfterToken = &types.EventToken{Data: startToken}
	}

	var cancel func()
	var cancelCtx context.Context
	cancelCtx, cancel = context.WithCancel(context.Background())

	watcher, err := lw.streamWatch.Watch(cancelCtx, opts.WatchOpt)
	if err != nil {
		blog.Errorf("%s job, run loop, but watch failed, err: %v", opts.Name, err)
		cancel()
		return err
	}

	retrySignal := make(chan struct{})
	retryObserver := &retryHandler{
		retryCounter: 0,
		maxRetryCnt:  opts.RetryOptions.MaxRetryCount,
	}

	handler := func(newCancelCtx context.Context, newWatcher *types.Watcher, newRetrySignal chan struct{}) {
		lw.tryLoopWithBatch(newCancelCtx, newWatcher, newRetrySignal, retryObserver, opts)
	}

	// start watch retry signal
	go lw.watchRetry(cancel, retrySignal, &opts.LoopOptions, handler)

	// start loop with events one by one.
	go lw.tryLoopWithBatch(cancelCtx, watcher, retrySignal, retryObserver, opts)

	return nil
}

type handlerFunc func(ctxWithCancel context.Context, watcher *types.Watcher, retrySignal chan struct{})

// watchRetry watch the loop jobs to check if a new watch is needed. if yes, then re-watch again.
func (lw *LoopsWatch) watchRetry(cancel context.CancelFunc,
	retrySignal chan struct{},
	opts *types.LoopOptions,
	doHandler handlerFunc) {

	for {
		// initialize a new context
		ctx := context.Background()

		select {
		case <-opts.StopNotifier:
			cancel()
			blog.Warnf("received stop %s loop watch job notify, stopping now.", opts.Name)
			return

		// wait for another retry
		case <-retrySignal:
			// wait for a well and then do the retry work.
			time.Sleep(opts.RetryOptions.RetryDuration)

			// initialize a new retry signal, exit loop immediately for next usage.
			retrySignal = make(chan struct{})

			// cancel the former watch
			cancel()

			// use the last token to resume so that we can start again from where we stopped.
			lastToken, err := opts.TokenHandler.GetStartWatchToken(ctx)
			if err != nil {
				blog.Errorf("%s job, run loop watch, but get last event token failed, err: %v", opts.Name, err)
				// notify retry signal, exit loop
				close(retrySignal)
				continue
			}

			blog.Errorf("%s job, the former watch loop: %s failed, start retry again from token: %s.",
				opts.Name, opts.WatchOpt.Collection, lastToken)

			// set start after token if needed.
			if len(lastToken) != 0 {
				// we have already received the new event and handle it success,
				// so we need to use this token. otherwise, we should still use the w.watchOpt.StartAfterToken
				opts.WatchOpt.StartAfterToken = &types.EventToken{Data: lastToken}
			}

			var cancelCtx context.Context
			cancelCtx, cancel = context.WithCancel(ctx)

			watcher, err := lw.streamWatch.Watch(cancelCtx, opts.WatchOpt)
			if err != nil {
				cancel()
				// notify retry signal, exit loop
				close(retrySignal)
				blog.Errorf("%s job, run retry loop, but watch failed, err: %v", err)
				continue
			}

			// start handle loop jobs
			go doHandler(cancelCtx, watcher, retrySignal)

			blog.Warnf("%s job, retry loop %s from token: %s success.", opts.Name, opts.WatchOpt.Collection, lastToken)
		}
	}
}

// tryLoopWithBatch try handle event with batch
func (lw *LoopsWatch) tryLoopWithBatch(ctxWithCancel context.Context,
	watcher *types.Watcher,
	retrySignal chan struct{},
	retryObserver *retryHandler,
	opts *types.LoopBatchOptions) {

	ticker := time.NewTicker(50 * time.Millisecond)

	observer := &observer{
		isMaster:       lw.isMaster,
		previousStatus: lw.isMaster.IsMaster(),
	}

	for {

		reWatch, loop := observer.canLoop()
		if reWatch {
			// stop the tick to release resource.
			ticker.Stop()
			blog.Warnf("%s job, master status has changed, try to re-watch again, collection：%s", opts.Name,
				opts.WatchOpt.Collection)

			// trigger re-watch action now.
			close(retrySignal)

			// exit the for loop
			return
		}

		if !loop {
			blog.V(5).Infof("%s job, loop %s event, but not master, skip.", opts.Name, opts.WatchOpt.Collection)
			time.Sleep(5 * time.Second)
			continue
		}

		batchEvents := make([]*types.Event, 0)
		// get event with loop.
		for {
			select {
			case <-ctxWithCancel.Done():
				// stop the tick to release resource.
				ticker.Stop()

				blog.Warnf("%s job, received cancel loop watch %s signal, exit loop.", opts.Name,
					opts.WatchOpt.Collection)
				// exist the goroutine
				return

			case one := <-watcher.EventChan:
				batchEvents = append(batchEvents, one)

				if blog.V(4) {
					blog.Infof("%s job, received %s event, detail: %s, op-time: %s, rid: %s", opts.Name,
						opts.WatchOpt.Collection, one.String(), one.ClusterTime.String(), one.ID())
				}

				// calculate event count, try to get more event for a batch
				if len(batchEvents) < opts.BatchSize {
					// continue to get more events
					continue
				}

			case <-ticker.C:
				// handle with batch event.
				if len(batchEvents) == 0 {
					// ticks, but no events received, loop next round to get events.
					continue
				}

			case <-opts.StopNotifier:
				ticker.Stop()
				blog.Warnf("received stop %s loop watch job notify, stopping now.", opts.Name)
				return
			}

			// break the for loop to handle event for now.
			break
		}

		// for safety guarantee
		if len(batchEvents) == 0 {
			continue
		}

		first := batchEvents[0]

		blog.Infof("%s job, received %s batch %d events, first op-time: %s rid: %s.", opts.Name, opts.WatchOpt.Collection,
			len(batchEvents), first.ClusterTime.String(), first.ID())

		retry := opts.EventHandler.DoBatch(batchEvents)
		if retry {

			if retryObserver.canStillRetry() {
				blog.Warnf("%s job, received %s %d events in batch, but do batch failed, retry now, rid: %s", opts.Name,
					opts.WatchOpt.Collection, len(batchEvents), first.ID())
				// an error occurred, we need to retry it later.
				// tell the schedule to re-watch again.
				close(retrySignal)
				// exist this goroutine.
				return
			}

			blog.Warnf("%s job, collection %s batch watch retry exceed max count, skip, rid: %s.", opts.Name,
				opts.WatchOpt.Collection, first.ID())
			// save the event token now.
		}

		// reset retry counter so that the previous retry count will not affect the next event
		retryObserver.resetRetryCounter()

		last := batchEvents[len(batchEvents)-1]
		// update the last watched token for resume usage.
		if err := opts.TokenHandler.SetLastWatchToken(ctxWithCancel, last.Token.Data); err != nil {
			blog.Errorf("%s job, loop watch %s event, but set last token failed, err: %v, rid: %s, retry later.",
				opts.Name, opts.WatchOpt.Collection, err, first.ID())

			// retry later.
			close(retrySignal)
			// exist this goroutine
			return
		}
	}

}

// tryLoopWithOne try handle event one by one
func (lw *LoopsWatch) tryLoopWithOne(ctxWithCancel context.Context,
	watcher *types.Watcher,
	retrySignal chan struct{},
	retryObserver *retryHandler,
	opts *types.LoopOneOptions) {

	observer := &observer{
		isMaster:       lw.isMaster,
		previousStatus: lw.isMaster.IsMaster(),
	}

	for one := range watcher.EventChan {
		select {
		case <-ctxWithCancel.Done():
			blog.Warnf("%s job, received cancel loop watch %s signal, exit loop, exit loop", opts.Name,
				opts.WatchOpt.Collection)
			return

		case <-opts.StopNotifier:
			blog.Warnf("received stop %s loop watch job notify, stopping now.", opts.Name)
			return

		default:
		}

		reWatch, loop := observer.canLoop()
		if reWatch {
			blog.Warnf("%s job, master status has changed, try to re-watch %s again", opts.Name, opts.WatchOpt.Collection)
			// trigger re-watch action now.
			close(retrySignal)
			// exit the for loop
			return
		}

		if !loop {
			blog.Infof("%s job, received %s %s event, but not master, skip. details: %s, rid: %s",
				opts.Name, opts.WatchOpt.Collection, one.OperationType, one.String(), one.ID())
			continue
		}

		blog.Infof("%s job, received %s event, type: %s, op-time: %s rid: %s", opts.Name, opts.WatchOpt.Collection,
			one.OperationType, one.ClusterTime.String(), one.ID())

		if blog.V(4) {
			blog.Infof("%s job, event details: %s, oid: %s", opts.Name, one.String(), one.ID())
		}

		retry := lw.tryOne(one, opts)
		if retry {
			if retryObserver.canStillRetry() {
				blog.Warnf("%s job, retry watch %s later. rid: %s", opts.Name, opts.WatchOpt.Collection, one.ID())
				// an error occurred, we need to retry it later.
				// tell the schedule to re-watch again.
				close(retrySignal)
				// exist this goroutine.
				return
			}

			blog.Warnf("%s job, retry %s event exceed max count, skip, detail: %s, rid: %s", opts.Name,
				opts.WatchOpt.Collection, one.String(), one.ID())
			// save the event token now.
		}

		// reset retry counter so that the previous retry count will not affect the next event
		retryObserver.resetRetryCounter()

		// update the last watched token for resume usage.
		if err := opts.TokenHandler.SetLastWatchToken(ctxWithCancel, one.Token.Data); err != nil {
			blog.Errorf("%s job, loop watch %s event, but set last watched token failed, err: %v, rid: %s, "+
				"retry later.",
				opts.WatchOpt.Collection, err, one.ID())

			// retry later.
			close(retrySignal)
			// exist this goroutine
			return
		}
	}
}

// tryOne handle event one by one.
func (lw *LoopsWatch) tryOne(e *types.Event, opts *types.LoopOneOptions) (retry bool) {
	switch e.OperationType {
	case types.Insert:
		retry := opts.EventHandler.DoAdd(e)
		if retry {
			blog.Warnf("%s job, received %s %s event,  but do add job failed, retry now, rid: %s", opts.Name,
				opts.WatchOpt.Collection, e.OperationType, e.ID())

			return retry
		}

	case types.Update, types.Replace:
		retry := opts.EventHandler.DoUpdate(e)
		if retry {
			blog.Warnf("%s job, received %s %s event, but do update job failed, retry now, rid: %s", opts.Name,
				opts.WatchOpt.Collection, e.OperationType, e.ID())

			return retry
		}

	case types.Delete:
		retry := opts.EventHandler.DoDelete(e)
		if retry {
			blog.Warnf("%s job, received %s %s event, but do delete job failed, retry now, rid: %s", opts.Name,
				opts.WatchOpt.Collection, e.OperationType, e.ID())

			return retry
		}

	case types.Invalidate:
		blog.Errorf("%s job, watch %s event, received invalid operation type, doc: %s, rid: %s", opts.Name,
			opts.WatchOpt.Collection, e.DocBytes, e.ID())
		return false

	default:
		blog.Errorf("%s job, watch %s event, received unsupported operation type, doc: %s, rid: %s", opts.Name,
			opts.WatchOpt.Collection, e.DocBytes, e.ID())
		return false
	}

	return false
}

// describe the event retry policy
type retryHandler struct {
	// current retry count
	retryCounter int
	// a event's max retry count
	maxRetryCnt int
}

// check if this event can still retry
func (r *retryHandler) canStillRetry() bool {
	r.retryCounter += 1

	if r.retryCounter > r.maxRetryCnt {
		r.retryCounter = 0
		return false
	}

	return true
}

func (r *retryHandler) resetRetryCounter() {
	r.retryCounter = 0
}

type observer struct {
	isMaster       discovery.ServiceManageInterface
	previousStatus bool
}

// canLoop describe whether we can still loop the next event or next batch events.
// this is a master slave service. we should re-watch the event from the previous
// event token, only when we do this, we can loop the continuous events later which
// is no events is skipped or duplicated.
func (o *observer) canLoop() (reWatch bool, loop bool) {
	current := o.isMaster.IsMaster()

	if o.previousStatus == current {
		if !current {
			// not master, status not changed, and can not loop
			return false, false
		} else {
			// is master, status not changed, and can loop
			return false, true
		}
	}

	blog.Infof("loop watch, is master status changed from %v to %v.", o.previousStatus, current)

	// update status
	o.previousStatus = current

	// status already changed, and can not continue loop, need to re-watch again.
	return true, false
}
