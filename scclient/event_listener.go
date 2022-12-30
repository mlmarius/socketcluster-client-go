package scclient

import (
	"context"
	"errors"
	"github.com/go-logr/logr"
	"sync"
	"time"
)

type Empty struct{}

type EmitAckFunc func(eventName string, error interface{}, data interface{})
type ackListener struct {
	eventName     string
	ack           EmitAckFunc
	cancelTimeout context.CancelFunc
}

func (l ackListener) Ack(error interface{}, data interface{}) {
	l.ack(l.eventName, error, data)
}

type Listener struct {
	emitAckListener *sync.Map
	onListener      map[string]func(eventName string, data interface{})
	onAckListener   map[string]func(eventName string, data interface{}, ack func(error interface{}, data interface{}) error)
	logger          logr.Logger
	ctx             context.Context
}

func Init(ctx context.Context, logger logr.Logger) Listener {
	return Listener{
		emitAckListener: &sync.Map{},
		onListener:      make(map[string]func(eventName string, data interface{})),
		onAckListener:   make(map[string]func(eventName string, data interface{}, ack func(error interface{}, data interface{}) error)),
		logger:          logger,
		ctx:             ctx,
	}
}

// putEmitAck registers a callback that gets called when we receive back a reply from the WS server
func (l *Listener) putEmitAck(id int, eventName string, timeout time.Duration, ack EmitAckFunc) {
	ctx, cf := context.WithTimeout(l.ctx, timeout)
	ackObj := ackListener{
		eventName:     eventName,
		ack:           ack,
		cancelTimeout: cf,
	}

	l.emitAckListener.Store(id, ackObj)

	go func() {
		// continues when
		//	- global context is canceled
		//	- timeout for this emit expires
		//	- acknowledgement is received
		<-ctx.Done()
		obj, loaded := l.emitAckListener.LoadAndDelete(id)

		if !loaded {
			// the callback has already been fulfilled and removed from the map. nothing left to do
			return
		}

		ackObj, _ := obj.(ackListener)
		l.logger.Error(errors.New("Ack timed out"), "Procedure call failed", "event", eventName)
		ackObj.Ack(errors.New("request timed out or canceled"), nil)
	}()
}

func (l *Listener) handleEmitAck(id int, error interface{}, data interface{}) {
	obj, loaded := l.emitAckListener.LoadAndDelete(id)
	if !loaded {
		l.logger.Info("Ack function not found for rid :: ", "id", id)
		return
	}

	ackObj, _ := obj.(ackListener)
	l.logger.Info("Ack received for event :: ", "eventName", ackObj.eventName)
	ackObj.cancelTimeout()
	ackObj.Ack(error, data)
}

func (l *Listener) putOnListener(eventName string, onListener func(eventName string, data interface{})) {
	l.onListener[eventName] = onListener
}

func (l *Listener) handleOnListener(eventName string, data interface{}) {
	on := l.onListener[eventName]
	if on != nil {
		on(eventName, data)
	}
}

func (l *Listener) putOnAckListener(eventName string, onAckListener func(eventName string, data interface{}, ack func(error interface{}, data interface{}) error)) {
	l.onAckListener[eventName] = onAckListener
}

func (l *Listener) handleOnAckListener(eventName string, data interface{}, ack func(error interface{}, data interface{}) error) {
	onAck := l.onAckListener[eventName]
	if onAck != nil {
		onAck(eventName, data, ack)
	}
}

func (l *Listener) hasEventAck(eventName string) bool {
	return l.onAckListener[eventName] != nil
}
