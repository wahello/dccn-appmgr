package main

import (
	"context"

	//	"github.com/Ankr-network/dccn-common/protos"

	"log"
	"time"

	appmgr "github.com/Ankr-network/dccn-common/protos/appmgr/v1/grpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"

	//usermgr "github.com/Ankr-network/dccn-common/protos/usermgr/v1/grpc"
	common_proto "github.com/Ankr-network/dccn-common/protos/common"
)

var addr = "appmgr:50051"

func main() {

	log.SetFlags(log.LstdFlags | log.Llongfile)
	conn, err := grpc.Dial(addr, grpc.WithInsecure())
	if err != nil {
		log.Fatal(err.Error())
	}
	defer func(conn *grpc.ClientConn) {
		if err := conn.Close(); err != nil {
			log.Println(err.Error())
		}
	}(conn)

	appClient := appmgr.NewAppMgrClient(conn)

	md := metadata.New(map[string]string{
		"authorization": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJleHAiOjE1NjMyMzg5NTEsImp0aSI6IjU1NzNhYjY3LTQ0YTUtNGY2Yi1iMjY2LTY3MzA1MjcyZWEzMSIsImlzcyI6ImFua3IuY29tIn0.-8NckBOtNjuhMC2B4CgYYtK9qmzFa2IGA0zjpfPDFgw",
	})

	ctx := metadata.NewOutgoingContext(context.Background(), md)
	//
	tokenContext, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	//

	if res, err := appClient.AppList(tokenContext, &common_proto.Empty{}); err != nil {
		log.Fatal(err)
	} else {
		log.Printf("app list successfully : \n  %v ", res.AppReports)
	}

}
