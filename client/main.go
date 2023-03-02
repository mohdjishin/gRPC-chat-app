package main

import (
	"bufio"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"flag"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/mohdjishin/chat-app-gRPC/protopb"
	"google.golang.org/grpc"
)

var wait *sync.WaitGroup

func init() {
	wait = &sync.WaitGroup{}
}

func connect(user *protopb.User, client protopb.BroadcastClient) error {
	stream, err := client.CreateStream(context.Background(), &protopb.Connect{
		User:   user,
		Active: true,
	})
	if err != nil {
		return fmt.Errorf("error while creating stream: %s", err)
	}

	wait.Add(1)
	go func(str protopb.Broadcast_CreateStreamClient) {
		defer wait.Done()
		for {
			msg, err := str.Recv()
			if err != nil {
				fmt.Printf("error while receiving message: %s\n", err)
				break
			}
			fmt.Printf("Message received from %s - %s\n", msg.User.Id, msg.Content)
		}
	}(stream)
	return nil
}

func main() {
	timestamp := time.Now()
	done := make(chan int)

	name := flag.String("N", "Jishin", "Name of the user")
	id := sha256.Sum256([]byte(*name + timestamp.String() + *name))

	conn, err := grpc.Dial("localhost:8080", grpc.WithInsecure())
	if err != nil {
		fmt.Println("error while connecting to server:", err)
		return
	}
	defer conn.Close()

	client := protopb.NewBroadcastClient(conn)
	user := &protopb.User{
		Id:   hex.EncodeToString(id[:]),
		Name: *name,
	}
	if err := connect(user, client); err != nil {
		fmt.Println("error while connecting:", err)
		return
	}

	wait.Add(1)
	go func() {
		defer wait.Done()

		scanner := bufio.NewScanner(os.Stdin)
		for scanner.Scan() {
			msg := &protopb.Message{
				User:      user,
				Content:   scanner.Text(),
				Timestamp: timestamp.String(),
			}

			_, err := client.BroadcastMessage(context.Background(), msg)
			if err != nil {
				fmt.Println("error while sending message:", err)
				break
			}
		}
	}()

	go func() {
		wait.Wait()
		close(done)
	}()

	<-done
}
