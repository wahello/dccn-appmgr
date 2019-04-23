package main

import (
	"context"
	"io/ioutil"

	//	"github.com/Ankr-network/dccn-common/protos"

	"log"
	"time"

	appmgr "github.com/Ankr-network/dccn-common/protos/appmgr/v1/grpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	//usermgr "github.com/Ankr-network/dccn-common/protos/usermgr/v1/grpc"
	//	apiCommon "github.com/Ankr-network/dccn-hub/app-dccn-api/examples/common"
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
		"token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJleHAiOjE1NTUwMTUzMTgsImp0aSI6IjQ4NTQ5YjQxLWUzNjYtNGIxMi05NTc3LTU0M2Y5NTE5Y2JlZiIsImlzcyI6ImFua3IubmV0d29yayJ9.A0p3KyxIKZHAZb_buPgadKj3d40Rlw_hSpsFBrNLjuw",
	})

	ctx := metadata.NewOutgoingContext(context.Background(), md)
	//
	tokenContext, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	//
	file, err := ioutil.ReadFile("values.yaml")
	if err != nil {
		log.Panic(err.Error())
	}
	if rsp, err := appClient.SaveChart(tokenContext, &appmgr.SaveChartRequest{
		ValueFile: file,
		ChartName: "wordpress",
		ChartVer:  "5.6.0",
		ChartRepo: "stable",
		SaveName:  "wordpress_nodeport",
		SaveVer:   "5.6.1",
		SaveRepo:  "stable"}); err != nil {
		log.Fatal(err)
	} else {
		log.Println("save chart successfully    ")
	}

}
