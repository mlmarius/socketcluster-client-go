package scclient

import "github.com/go-logr/logr"

type Empty struct{}

type Listener struct {
	emitAckListener map[int][]interface{}
	onListener      map[string]func(eventName string, data interface{})
	onAckListener   map[string]func(eventName string, data interface{}, ack func(error interface{}, data interface{}))
	logger          logr.Logger
}

func Init(logger logr.Logger) Listener {
	return Listener{
		emitAckListener: make(map[int][]interface{}),
		onListener:      make(map[string]func(eventName string, data interface{})),
		onAckListener:   make(map[string]func(eventName string, data interface{}, ack func(error interface{}, data interface{}))),
		logger:          logger,
	}
}

func (listener *Listener) putEmitAck(id int, eventName string, ack func(eventName string, error interface{}, data interface{})) {
	listener.emitAckListener[id] = []interface{}{eventName, ack}
}

func (listener *Listener) handleEmitAck(id int, error interface{}, data interface{}) {
	ackObject := listener.emitAckListener[id]
	if ackObject != nil {
		eventName := ackObject[0].(string)
		listener.logger.Info("Ack received for event :: ", eventName)
		ack := ackObject[1].(func(eventName string, error interface{}, data interface{}))
		ack(eventName, error, data)
	} else {
		listener.logger.Info("Ack function not found for rid :: ", id)
	}
}

func (listener *Listener) putOnListener(eventName string, onListener func(eventName string, data interface{})) {
	listener.onListener[eventName] = onListener
}

func (listener *Listener) handleOnListener(eventName string, data interface{}) {
	on := listener.onListener[eventName]
	if on != nil {
		on(eventName, data)
	}
}

func (listener *Listener) putOnAckListener(eventName string, onAckListener func(eventName string, data interface{}, ack func(error interface{}, data interface{}))) {
	listener.onAckListener[eventName] = onAckListener
}

func (listener *Listener) handleOnAckListener(eventName string, data interface{}, ack func(error interface{}, data interface{})) {
	onAck := listener.onAckListener[eventName]
	if onAck != nil {
		onAck(eventName, data, ack)
	}
}

func (listener *Listener) hasEventAck(eventName string) bool {
	return listener.onAckListener[eventName] != nil
}
