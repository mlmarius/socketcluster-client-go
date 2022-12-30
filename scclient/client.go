package scclient

import (
	"context"
	"github.com/go-logr/logr"
	"github.com/mlmarius/gowebsocket"
	"github.com/mlmarius/socketcluster-client-go/scclient/models"
	"github.com/mlmarius/socketcluster-client-go/scclient/parser"
	"github.com/mlmarius/socketcluster-client-go/scclient/utils"
	_ "golang.org/x/net/websocket"
	"net/http"
	"time"
	_ "time"
)

func handlePing(message string) (bool, string) {
	if message == "" {
		return true, ""
	}
	if message == "#1" {
		return true, "#2"
	}
	return false, ""
}

type Client struct {
	authToken           *string
	url                 string
	counter             utils.AtomicCounter
	socket              gowebsocket.Socket
	onConnect           func(client Client)
	onConnectError      func(client Client, err error)
	onDisconnect        func(client Client, err error)
	onSetAuthentication func(client Client, token string)
	onAuthentication    func(client Client, isAuthenticated bool)
	ConnectionOptions   gowebsocket.ConnectionOptions
	RequestHeader       http.Header
	logger              logr.Logger
	defTimeout          time.Duration
	ctx                 context.Context
	cf                  context.CancelFunc
	Listener
}

func New(ctx context.Context, url string, logger logr.Logger, defTimeout time.Duration) Client {
	innerCtx, cf := context.WithCancel(ctx)
	return Client{
		url:        url,
		counter:    utils.AtomicCounter{Counter: 0},
		logger:     logger,
		Listener:   Init(innerCtx, logger),
		defTimeout: defTimeout,
		ctx:        innerCtx,
		cf:         cf,
	}
}

func (client *Client) IsConnected() bool {
	return client.socket.IsConnected
}

func (client *Client) GetLogger() logr.Logger {
	return client.logger
}

func (client *Client) SetAuthToken(token string) {
	client.authToken = &token
}

func (client *Client) GetAuthToken() *string {
	return client.authToken
}

func (client *Client) SetBasicListener(onConnect func(client Client), onConnectError func(client Client, err error), onDisconnect func(client Client, err error)) {
	client.onConnect = onConnect
	client.onConnectError = onConnectError
	client.onDisconnect = onDisconnect
}

func (client *Client) SetAuthenticationListener(onSetAuthentication func(client Client, token string), onAuthentication func(client Client, isAuthenticated bool)) {
	client.onSetAuthentication = onSetAuthentication
	client.onAuthentication = onAuthentication
}

func (client *Client) registerCallbacks() {

	client.socket.OnConnected = func(socket gowebsocket.Socket) {
		client.counter.Reset()
		if err := client.sendHandshake(); err != nil {
			client.logger.Error(err, "Handshake failed")
			client.Disconnect()
		}
		if client.onConnect != nil {
			client.onConnect(*client)
		}
	}
	client.socket.OnConnectError = func(err error, socket gowebsocket.Socket) {
		client.cf()
		if err != nil {
			if client.onConnectError != nil {
				client.onConnectError(*client, err)
			}
		}
	}
	client.socket.OnTextMessage = func(message string, socket gowebsocket.Socket) {
		client.logger.Info("inbound message", "value", message)

		if isPing, reply := handlePing(message); isPing {
			if err := client.socket.SendText(reply); err != nil {
				client.logger.Error(err, "Could not send ping reply")
			}
			return
		}

		var messageObject = utils.DeserializeDataFromString(message)
		data, rid, cid, eventname, err := parser.GetMessageDetails(messageObject)

		parseresult := parser.Parse(rid, cid, eventname)

		switch parseresult {
		case parser.ISAUTHENTICATED:
			isAuthenticated := utils.GetIsAuthenticated(messageObject)
			if client.onAuthentication != nil {
				client.onAuthentication(*client, isAuthenticated)
			}
		case parser.SETTOKEN:
			client.logger.Info("Set token event received")
			token := utils.GetAuthToken(messageObject)
			if client.onSetAuthentication != nil {
				client.onSetAuthentication(*client, token)
			}

		case parser.REMOVETOKEN:
			client.logger.Info("Remove token event received")
			client.authToken = nil
		case parser.EVENT:
			client.logger.Info("Received data for event :: ", "eventName", eventname)
			if client.hasEventAck(eventname.(string)) {
				client.handleOnAckListener(eventname.(string), data, client.ack(cid))
			} else {
				client.handleOnListener(eventname.(string), data)
			}
		case parser.ACKRECEIVE:
			client.handleEmitAck(rid, err, data)
		case parser.PUBLISH:
			channel := models.GetChannelObject(data)
			client.logger.Info("Publish event received for channel :: ", "channel", channel.Channel)
			client.handleOnListener(channel.Channel, channel.Data)
		}

	}
	client.socket.OnDisconnected = func(err error, socket gowebsocket.Socket) {
		client.cf()
		if client.onDisconnect != nil {
			client.onDisconnect(*client, err)
		}
		return
	}

}

func (client *Client) Connect() context.Context {
	socketLogger := client.logger.WithName("socket")
	client.socket = gowebsocket.New(client.url, socketLogger)
	client.registerCallbacks()
	// Connect
	client.socket.ConnectionOptions = client.ConnectionOptions
	client.socket.RequestHeader = client.RequestHeader
	client.socket.Connect(client.ctx)
	return client.ctx
}

func (client *Client) sendHandshake() error {
	handshake := utils.SerializeDataIntoString(models.GetHandshakeObject(client.authToken, int(client.counter.IncrementAndGet())))
	return client.socket.SendText(handshake)
}

func (client *Client) ack(cid int) func(error interface{}, data interface{}) error {
	return func(error interface{}, data interface{}) error {
		ackObject := models.GetReceiveEventObject(data, error, cid)
		ackData := utils.SerializeDataIntoString(ackObject)
		return client.socket.SendText(ackData)
	}
}

func (client *Client) Emit(eventName string, data interface{}) error {
	emitObject := models.GetEmitEventObject(eventName, data, int(client.counter.IncrementAndGet()))
	emitData := utils.SerializeDataIntoString(emitObject)
	return client.socket.SendText(emitData)
}

func (client *Client) EmitAck(eventName string, data interface{}, timeout time.Duration, ack EmitAckFunc) error {
	id := int(client.counter.IncrementAndGet())
	emitObject := models.GetEmitEventObject(eventName, data, id)
	emitData := utils.SerializeDataIntoString(emitObject)
	client.putEmitAck(id, eventName, timeout, ack)
	return client.socket.SendText(emitData)
}

func (client *Client) Subscribe(channelName string) error {
	subscribeObject := models.GetSubscribeEventObject(channelName, int(client.counter.IncrementAndGet()))
	subscribeData := utils.SerializeDataIntoString(subscribeObject)
	return client.socket.SendText(subscribeData)
}

func (client *Client) SubscribeAck(channelName string, timeout time.Duration, ack func(eventName string, error interface{}, data interface{})) error {
	id := int(client.counter.IncrementAndGet())
	subscribeObject := models.GetSubscribeEventObject(channelName, id)
	subscribeData := utils.SerializeDataIntoString(subscribeObject)
	client.putEmitAck(id, channelName, timeout, ack)
	return client.socket.SendText(subscribeData)
}

func (client *Client) Unsubscribe(channelName string) error {
	unsubscribeObject := models.GetUnsubscribeEventObject(channelName, int(client.counter.IncrementAndGet()))
	unsubscribeData := utils.SerializeDataIntoString(unsubscribeObject)
	return client.socket.SendText(unsubscribeData)
}

func (client *Client) UnsubscribeAck(channelName string, timeout time.Duration, ack func(eventName string, error interface{}, data interface{})) error {
	id := int(client.counter.IncrementAndGet())
	unsubscribeObject := models.GetUnsubscribeEventObject(channelName, id)
	unsubscribeData := utils.SerializeDataIntoString(unsubscribeObject)
	client.putEmitAck(id, channelName, timeout, ack)
	return client.socket.SendText(unsubscribeData)
}

func (client *Client) Publish(channelName string, data interface{}) error {
	publishObject := models.GetPublishEventObject(channelName, data, int(client.counter.IncrementAndGet()))
	publishData := utils.SerializeDataIntoString(publishObject)
	return client.socket.SendText(publishData)
}

func (client *Client) PublishAck(channelName string, data interface{}, timeout time.Duration, ack func(eventName string, error interface{}, data interface{})) error {
	id := int(client.counter.IncrementAndGet())
	publishObject := models.GetPublishEventObject(channelName, data, id)
	publishData := utils.SerializeDataIntoString(publishObject)
	client.putEmitAck(id, channelName, timeout, ack)
	return client.socket.SendText(publishData)
}

func (client *Client) OnChannel(eventName string, ack func(eventName string, data interface{})) {
	client.putOnListener(eventName, ack)
}

func (client *Client) On(eventName string, ack func(eventName string, data interface{})) {
	client.putOnListener(eventName, ack)
}

func (client *Client) OnAck(eventName string, ack func(eventName string, data interface{}, ack func(error interface{}, data interface{}) error)) {
	client.putOnAckListener(eventName, ack)
}

func (client *Client) Disconnect() error {
	client.cf()
	return client.socket.Close()
}
