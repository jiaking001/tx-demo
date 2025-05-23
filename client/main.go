package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"google.golang.org/grpc"
	user "tx-demo/user/proto"
)

func main() {
	conn, err := grpc.Dial(":50051", grpc.WithInsecure(), grpc.WithBlock())
	if err != nil {
		log.Fatalf("Failed to connect: %v", err)
	}
	defer conn.Close()

	client := user.NewUserServiceClient(conn)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	// 注册
	//registerReq := &user.RegisterRequest{
	//	Username: "John",
	//	Password: "password123",
	//	Like:     "sleep",
	//}
	//registerResp, err := client.Register(ctx, registerReq)
	//if err != nil {
	//	log.Fatalf("Failed to register: %v", err)
	//}
	//fmt.Printf("Register Response: %+v\n", registerResp)

	// 登录
	loginReq := &user.LoginRequest{
		Username: "John",
		Password: "password123",
	}
	loginResp, err := client.Login(ctx, loginReq)
	if err != nil {
		log.Fatalf("Failed to login: %v", err)
	}
	fmt.Printf("Login Response: %+v\n", loginResp)

	// 获取用户信息
	//userInfoResp, err := client.GetUserInfo(ctx, &emptypb.Empty{})
	//if err != nil {
	//	log.Fatalf("Failed to get user info: %v", err)
	//}
	//fmt.Printf("User Info Response: %+v\n", userInfoResp)
}
