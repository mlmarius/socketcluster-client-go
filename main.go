package main

import (
	"context"
	"errors"
	"fmt"
	"github.com/go-logr/zapr"
	"github.com/mlmarius/socketcluster-client-go/scclient"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"os"
	"text/scanner"
	"time"
)

func onConnect(_ scclient.Client) {
	fmt.Println("Connected to server")
}

func onDisconnect(client scclient.Client, err error) {
	client.GetLogger().Error(err, "Disconnected")
}

func onConnectError(_ scclient.Client, _ error) {
	//client.GetLogger().Error(err, "Error on connection")
	//os.Exit(1)
}

func onSetAuthentication(client scclient.Client, token string) {
	client.GetLogger().Info("Authentication token received", "token", token)
	client.SetAuthToken(token)
}

func onAuthentication(client scclient.Client, isAuthenticated bool) {
	client.GetLogger().Info("onAuthentication ...")
	if isAuthenticated {
		client.GetLogger().Info("client already authenticated ... starting ...")
		client.GetLogger().Info("Client already authenticated")
		start(client)
	} else {
		client.GetLogger().Info("Client logging in")
		err := client.EmitAck("login", struct{}{}, time.Second*10, func(evt string, err interface{}, data interface{}) {
			if err != nil {
				switch v := err.(type) {
				case map[string]interface{}:
					if errString, ok := v["message"].(string); ok {
						client.GetLogger().Error(errors.New(errString), "Login failed")
					} else {
						client.GetLogger().Error(errors.New("could not deserialize login error"), "Login failed")
					}
				default:
					client.GetLogger().Error(errors.New("unknown error type"), "Login failed")
				}
				_ = client.Disconnect()
			} else {
				client.GetLogger().Info("login success", "event", evt, "err", err, "data", data)
				client.GetLogger().Info("starting after login success ... ")
				start(client)
			}
		})
		if err != nil {
			client.GetLogger().Error(err, "Could not send login request")
			_ = client.Disconnect()
		}
	}
}

func getLogger() *zap.Logger {
	config := zap.NewProductionEncoderConfig()
	config.EncodeTime = zapcore.ISO8601TimeEncoder
	consoleEncoder := zapcore.NewConsoleEncoder(config)
	defaultLogLevel := zapcore.DebugLevel
	core := zapcore.NewCore(consoleEncoder, zapcore.AddSync(os.Stdout), defaultLogLevel)
	logger := zap.New(
		core,
		zap.AddCaller(),
		zap.AddStacktrace(zapcore.ErrorLevel),
	)
	return logger
}

func main() {
	var reader scanner.Scanner

	zlogger := getLogger()
	defer func() {
		_ = zlogger.Sync() // flushes buffer, if any
	}()
	logger := zapr.NewLoggerWithOptions(zlogger)

	ctx, cf := context.WithCancel(context.Background())

	go func() {
		for {
			client := scclient.New(ctx, "ws://localhost:8080/socketcluster/", logger.WithName("client"), time.Second*60)
			client.SetBasicListener(onConnect, onConnectError, onDisconnect)
			client.SetAuthenticationListener(onSetAuthentication, onAuthentication)

			logger.Info("attempting connection to server")
			clientCtx := client.Connect()
			<-clientCtx.Done()
			_ = client.Disconnect()
			time.Sleep(time.Second * 5)
		}
	}()

	//fmt.Println("Enter any key to terminate the program")
	reader.Init(os.Stdin)
	reader.Next()
	cf()
	// os.Exit(0)
}

func start(client scclient.Client) {
	// start writing your code from here
	client.GetLogger().Info("started ...")
	start := time.Now()
	_ = client.EmitAck("test", struct{}{}, time.Second, func(eventName string, error interface{}, data interface{}) {
		dt := time.Now().Sub(start)
		client.GetLogger().Info("test procedure finalized", "time passed", dt, "data", data, "error", error)
	})
}
