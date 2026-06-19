package main

import (
	"context"
	"fmt"
	"net"
	"os"
	"os/signal"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	"github.com/menta2k/protoc-gen-redact/v3/redact/v3"

	"github.com/menta2k/protoc-gen-redact/v3/examples/user/pb"
)

func RunServer() (*grpc.Server, <-chan struct{}) {
	ready := make(chan struct{})
	// new grpc server
	srv := grpc.NewServer()
	// register our Redacted Chat Server
	pb.RegisterRedactedChatServer(srv,
		NewChatServer(),
		redact.Wrapper(func(ctx context.Context) bool {
			md, ok := metadata.FromIncomingContext(ctx)
			return ok && md["key"] != nil
		}),
	)
	// run server
	go func() {
		// tcp listener
		lis, err := net.Listen("tcp", ":50051")
		if err != nil {
			panic(err)
		}
		fmt.Println("listening")
		ready <- struct{}{}
		if err := srv.Serve(lis); err != nil {
			panic(err)
		}
	}()
	return srv, ready
}

func RunClient(ctx context.Context) (*grpc.ClientConn, error) {
	// new grpc connection
	conn, err := grpc.DialContext(ctx, ":50051", grpc.WithInsecure())
	if err != nil {
		return conn, err
	}
	// our chat client
	cli := pb.NewChatClient(conn)
	return conn, TestClient(ctx, cli)
}

func TestClient(ctx context.Context, cli pb.ChatClient) error {
	dummy := &pb.User{
		Username: "one",
		Password: "password",
		Email:    "one@eno.com",
		Name:     "One",
		Home: &pb.User_Location{
			Lat: 23.32,
			Lng: 32.42,
		},
	}

	// Test chat server & client
	got, err := cli.GetUser(ctx, &pb.GetUserRequest{Username: dummy.Username})
	if c := status.Code(err); err == nil || c != codes.NotFound {
		return fmt.Errorf("GetUser() want NotFound, got %v", c)
	}
	fmt.Println("get user", got)

	got, err = cli.AddUser(ctx, dummy)
	if c := status.Code(err); err == nil || c != codes.PermissionDenied {
		return fmt.Errorf("AddUser() want PermissionDenied, got %v", c)
	}
	fmt.Println("add user", got)

	// new bypass context
	newCtx := metadata.NewOutgoingContext(ctx, map[string][]string{"key": {"value"}})

	// bypass AddUser redaction
	got, err = cli.AddUser(newCtx, dummy)
	if err != nil {
		return err
	}
	fmt.Println("bypass add user", got)

	// normal get
	got, err = cli.GetUser(ctx, &pb.GetUserRequest{Username: dummy.Username})
	if err != nil || got == nil {
		return err
	}
	if got.Password != "REDACTED" || got.Password == dummy.Password {
		return fmt.Errorf("password should be redacted")
	}
	if got.Email != "r*d@ct*d" || got.Email == dummy.Email {
		return fmt.Errorf("email should also be redacted")
	}
	fmt.Println("get user", got)

	// bypassed get
	got, err = cli.GetUser(newCtx, &pb.GetUserRequest{Username: dummy.Username})
	if err != nil || got == nil {
		return err
	}
	if got.Password == "REDACTED" || got.Password != dummy.Password {
		return fmt.Errorf("password be bypassed")
	}
	if got.Email == "r*d@ct*d" || got.Email != dummy.Email {
		return fmt.Errorf("email be bypassed")
	}
	fmt.Println("bypass get user", got)

	return nil
}

func main() {
	done := make(chan os.Signal, 1)
	signal.Notify(done, os.Interrupt, os.Kill)
	defer signal.Stop(done)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// run our chat server
	srv, ready := RunServer()
	// wait till srv is ready
	<-ready
	// run our chat client to test server
	conn, err := RunClient(ctx)
	if err != nil {
		panic(err)
	}

	cancel()
	select {
	case <-done:
		break
	case <-ctx.Done():
		break
	}
	// close server and client connections
	if err := conn.Close(); err != nil {
		panic(err)
	}
	srv.GracefulStop()
	fmt.Println("done")
}
