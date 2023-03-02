package main

import (
	"context"
	"log"
	"net"
	"os"
	"sync"

	"github.com/mohdjishin/chat-app-gRPC/protopb"
	"google.golang.org/grpc"
	glog "google.golang.org/grpc/grpclog"
)

var grpcLog glog.LoggerV2

type Connection struct {
	stream protopb.Broadcast_CreateStreamServer
	id     string
	active bool
	error  chan error
}

type Server struct {
	Connection []*Connection
	protopb.UnimplementedBroadcastServer
}

func init() {

	grpcLog = glog.NewLoggerV2(os.Stdout, os.Stdout, os.Stdout)
}

func (s *Server) CreateStream(pconn *protopb.Connect, stream protopb.Broadcast_CreateStreamServer) error {

	conn := &Connection{
		stream: stream,
		id:     pconn.User.Id,
		active: true,
		error:  make(chan error),
	}

	s.Connection = append(s.Connection, conn)
	return <-conn.error
}

func (s *Server) BroadcastMessage(ctx context.Context, msg *protopb.Message) (*protopb.Close, error) {

	wait := sync.WaitGroup{}
	done := make(chan int)
	for _, conn := range s.Connection {
		wait.Add(1)

		go func(msg *protopb.Message, conn *Connection) {
			defer wait.Done()
			if conn.active {
				err := conn.stream.Send(msg)
				grpcLog.Info("Message sent to ", conn.stream)
				if err != nil {
					grpcLog.Errorf("Error while sending message to %s  - Error %s", conn.stream, err)
					conn.active = false
					conn.error <- err
				}

			}
		}(msg, conn)

	}

	go func() {

		wait.Wait()
		close(done)

	}()

	<-done
	return &protopb.Close{}, nil
}
func main() {

	var connections []*Connection

	// server := &Server{connections}

	grpcServer := grpc.NewServer()
	listener, err := net.Listen("tcp", ":8080")
	if err != nil {
		log.Fatalf("Error while creating sever %v", err)
	}

	grpcLog.Info("Server started at port 8080")

	protopb.RegisterBroadcastServer(grpcServer, &Server{connections, protopb.UnimplementedBroadcastServer{}})
	grpcServer.Serve(listener)
}
