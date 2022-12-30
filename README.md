# socketcluster-client-go
Refer examples for more details :
    
Overview
--------
This client provides following functionality

- Easy to setup and use
- Support for emitting and listening to remote events
- Pub/sub
- Authentication (JWT)
- Can be used for testing of all server side functions

To install use

```markdown
    go get github.com/mlmarius/socketcluster-client-go/scclient
```

Description
-----------
Create instance of `scclient` by passing url of socketcluster-server end-point 

```go
    //Create a client instance
    client := scclient.New("ws://192.168.100.11:8000/socketcluster/");
    
```
**Important Note** : Default url to socketcluster end-point is always *ws://somedomainname.com/socketcluster/*.

#### Registering basic listeners
 
Different functions are given as an argument to register listeners

```go
        package main
        
        import (
        	"github.com/mlmarius/socketcluster-client-go/scclient"
        	"text/scanner"
        	"os"
        	"fmt"
        )
        
        func onConnect(client scclient.Client) {
            fmt.Println("Connected to server")
        }
        
        func onDisconnect(client scclient.Client, err error) {
            fmt.Printf("Error: %s\n", err.Error())
        }
        
        func onConnectError(client scclient.Client, err error) {
            fmt.Printf("Error: %s\n", err.Error())
        }
        
        func onSetAuthentication(client scclient.Client, token string) {
            fmt.Println("Auth token received :", token)
        
        }
        
        func onAuthentication(client scclient.Client, isAuthenticated bool) {
            fmt.Println("Client authenticated :", isAuthenticated)
            go startCode(client)
        }  
            
        func main() {
        	var reader scanner.Scanner
        	client := scclient.New("ws://192.168.100.11:8000/socketcluster/");
        	client.SetBasicListener(onConnect, onConnectError, onDisconnect)
        	client.SetAuthenticationListener(onSetAuthentication, onAuthentication)
        	go client.Connect()
        
        	fmt.Println("Enter any key to terminate the program")
        	reader.Init(os.Stdin)
        	reader.Next()
        	// os.Exit(0)
        }
        
        func startCode(client scclient.Client) {
        	// start writing your code from here
        	// All emit, receive and publish events
        }
        
```

#### Connecting to server

- For connecting to server:

```go
    //This will send websocket handshake request to socketcluster-server
    client.Connect()
```

Emitting and listening to events
--------------------------------
#### Event emitter

- eventname is name of event and message can be String, boolean, int or structure

```go

    client.Emit(eventname,message);
        
    //  client.Emit("chat","This is a sample message")
```

- To send event with acknowledgement

```go

	client.EmitAck("chat","This is a sample message", func(eventName string, error interface{}, data interface{}) {
		if error == nil {
			fmt.Println("Got ack for emit event with data ", data, " and error ", error)
		}
	})
	
```

#### Event Listener

- For listening to events :

The object received can be String, Boolean, Long or GO structure.

```go
    // Receiver code without sending acknowledgement back
    client.On("chat", func(eventName string, data interface{}) {
		fmt.Println("Got data ", data, " for event ", eventName)
	})
    
```

- To send acknowledgement back to server

```go
    // Receiver code with ack
	client.OnAck("chat", func(eventName string, data interface{}, ack func(error interface{}, data interface{})) {
		fmt.Println("Got data ", data, " for event ", eventName)
		fmt.Println("Sending back ack for the event")
		ack("This is error", "This is data")
	}) 
        
```

Implementing Pub-Sub via channels
---------------------------------

#### Creating channel

- For creating and subscribing to channels:

```go
    // without acknowledgement
    client.Subscribe("mychannel")
    
    //with acknowledgement
    client.SubscribeAck("mychannel", func(channelName string, error interface{}, data interface{}) {
        if error == nil {
            fmt.Println("Subscribed to channel ", channelName, "successfully")
        }
    })
```


#### Publishing event on channel

- For publishing event :

```go

       // without acknowledgement
       client.Publish("mychannel", "This is a data to be published")

       
       // with acknowledgement
       client.PublishAck("mychannel", "This is a data to be published", func(channelName string, error interface{}, data interface{}) {
       		if error == nil {
       			fmt.Println("Data published successfully to channel ", channelName)
       		}
       	})
``` 
 
#### Listening to channel

- For listening to channel event :

```go
        client.OnChannel("mychannel", func(channelName string, data interface{}) {
        		fmt.Println("Got data ", data, " for channel ", channelName)
        })
    
``` 
     
#### Un-subscribing to channel

```go
         // without acknowledgement
        client.Unsubscribe("mychannel")
         
         // with acknowledgement
        client.UnsubscribeAck("mychannel", func(channelName string, error interface{}, data interface{}) {
            if error == nil {
                fmt.Println("Unsubscribed to channel ", channelName, "successfully")
            }
        })
```

#### Closing the connection with server
```go
    client.Disconnect()
```

#### Setting request headers
```go
	client.RequestHeader.Set("Accept-Encoding","gzip, deflate, sdch")
	client.RequestHeader.Set("Accept-Language","en-US,en;q=0.8")
	client.RequestHeader.Set("Pragma","no-cache")
	client.RequestHeader.Set("User-Agent","Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/49.0.2623.87 Safari/537.36")
	
```

#### Setting proxy server
- It can be set using connectionOptions by providing url to proxy server

```go
    client.ConnectionOptions = gowebsocket.ConnectionOptions {
       Proxy: gowebsocket.BuildProxy("http://example.com"),
    }
```

#### Setting data compression, ssl verification and subprotocols

- It can be set using connectionOptions inside socket 

```go
    client.ConnectionOptions = gowebsocket.ConnectionOptions {
        UseSSL:true,
        UseCompression:true,
        Subprotocols: [] string{"chat","superchat"},
    }
```

- ConnectionOptions needs to be applied before connecting to server