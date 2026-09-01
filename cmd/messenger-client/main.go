package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	pb "highloadgram/api"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
)

func main() {
	username := flag.String("username", "", "client username")
	peer := flag.String("peer", "", "message recipient")
	addr := flag.String("addr", "localhost:50051", "gRPC server address")
	flag.Parse()

	if *username == "" || *peer == "" {
		log.Fatal("username and peer are required")
	}

	conn, err := grpc.NewClient(
		*addr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()

	client := pb.NewMessengerClient(conn)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	stream, err := client.Subscribe(ctx, &pb.SubscribeRequest{
		Username: *username,
	})
	if err != nil {
		log.Fatal(err)
	}

	go func() {
		for {
			msg, err := stream.Recv()
			if err != nil {
				if status.Code(err) != codes.Canceled {
					log.Printf("receive: %v", err)
				}
				return
			}

			fmt.Printf("%s: %s\n", msg.From, msg.Text)
		}
	}()

	fmt.Printf("connected as %s, peer: %s\n", *username, *peer)

	scanner := bufio.NewScanner(os.Stdin)

	for scanner.Scan() {
		text := strings.TrimSpace(scanner.Text())

		if text == "" {
			continue
		}

		sendCtx, sendCancel := context.WithTimeout(ctx, 5*time.Second)

		_, err := client.SendMessage(
			sendCtx,
			&pb.SendMessageRequest{
				From: *username,
				To:   *peer,
				Text: text,
			},
		)

		sendCancel()

		if err != nil {
			fmt.Printf("send error: %v\n", err)
		}
	}

	if err := scanner.Err(); err != nil {
		log.Printf("stdin: %v", err)
	}
}
