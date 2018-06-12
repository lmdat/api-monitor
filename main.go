package main

import (
	"fmt"

	"github.com/gomodule/redigo/redis"
	"github.com/googollee/go-socket.io"
	"github.com/kataras/iris"
)

func main() {
	// Web Server
	http := iris.New()

	// Mở Socket
	socket, err := socketio.NewServer(nil)
	if err != nil {
		panic(err)
	}

	// Mở connect đến Redis Server
	rConn, err2 := redis.Dial("tcp", ":6379")
	if err2 != nil {
		panic(err2)
	}
	defer rConn.Close()

	// Tạo Pub/Sub Connection
	rPubSubConn := &redis.PubSubConn{Conn: rConn}
	defer rPubSubConn.Close()

	// Subscribe channel "CALL-REST-API"
	err3 := rPubSubConn.Subscribe("CALL-REST-API")
	if err3 != nil {
		panic(err3)
	}

	// Nhận kết nối socket từ client
	socket.On("connection", func(so socketio.Socket) {
		http.Logger().Infof("Connected Id: %s", so.Id())

		so.Join("api-monitor")

		so.On("disconnection", func() {
			http.Logger().Infof("Disconnection From Id: %s", so.Id())
		})

	})

	socket.On("error", func(so socketio.Socket, err error) {
		http.Logger().Errorf("Error: %v", err)
	})

	// Nhận message từ Redis sau đó broadcast đến cho tất cả client đang kết nối
	go deliverMessage(rPubSubConn, socket)

	// Run Web server
	// Xử lý socket
	http.Any("/socket.io/", iris.FromStd(socket))

	// Xử lý file publish/index.html
	http.StaticWeb("/", "./public")

	http.Run(iris.Addr(":8000"))
}

func deliverMessage(conn *redis.PubSubConn, socket *socketio.Server) {
	for {
		switch v := conn.Receive().(type) {
		case redis.Message:
			socket.BroadcastTo("api-monitor", "CALL_REST_API", string(v.Data))
			fmt.Printf("%s: message: %s\n", v.Channel, v.Data)
		case redis.Subscription:
			fmt.Printf("%s: %s %d\n", v.Channel, v.Kind, v.Count)
		case error:
			panic(v)
		}
	}
}
