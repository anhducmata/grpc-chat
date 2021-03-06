package client

import (
	"bufio"
	"context"
	"fmt"
	"os"

	pb "github.com/cloudingcity/grpc-chat/proto"
	log "github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

func Connect(addr string, username, password string) {
	conn, err := grpc.Dial(addr, grpc.WithInsecure())
	if err != nil {
		log.Fatalln(err)
	}
	defer conn.Close()
	log.Infof("Connect to server: %s", addr)

	c := &Client{
		grpcClient: pb.NewChatClient(conn),
	}
	tkn, err := c.Connect(username, password)
	if err != nil {
		log.Fatalln(err)
	}
	if err := c.Stream(tkn, username); err != nil {
		log.Fatalln(err)
	}
}

type Client struct {
	grpcClient pb.ChatClient
}

func (c *Client) Connect(username, password string) (string, error) {
	req := &pb.ConnectRequest{
		Username: username,
		Password: password,
	}
	resp, err := c.grpcClient.Connect(context.Background(), req)
	if err != nil {
		return "", err
	}
	return resp.Token, nil
}

func (c *Client) Stream(token string, username string) error {
	md := metadata.Pairs("x-token", token)
	ctx := metadata.NewOutgoingContext(context.Background(), md)
	stream, err := c.grpcClient.Stream(ctx)
	if err != nil {
		return err
	}
	defer stream.CloseSend()

	// Send message
	go func() {
		sc := bufio.NewScanner(os.Stdin)
		for {
			if !sc.Scan() {
				log.Fatalln(sc.Err())
			}
			req := &pb.StreamRequest{
				Token:    token,
				Username: username,
				Message:  sc.Text(),
			}
			if err := stream.Send(req); err != nil {
				log.Fatalln(err)
			}
		}
	}()

	// Receive broadcast
	for {
		resp, err := stream.Recv()
		if err != nil {
			return err
		}
		fmt.Printf("[%s]: %s\n", resp.Username, resp.Message)
	}
}
