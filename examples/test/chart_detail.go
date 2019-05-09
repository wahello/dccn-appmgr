package main

import (
	"context"

	//	"github.com/Ankr-network/dccn-common/protos"

	"log"
	"time"

	appmgr "github.com/Ankr-network/dccn-common/protos/appmgr/v1/grpc"
	common_protos "github.com/Ankr-network/dccn-common/protos/common"
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

	if rsp, err := appClient.ChartDetail(tokenContext, &appmgr.ChartDetailRequest{
		Chart: &common_protos.Chart{
			ChartName: "wordpress",
			ChartRepo: "stable"},
		ShowVersion: "5.6.0",
	}); err != nil {
		log.Fatal(err)
	} else {
		log.Printf(" chart list successfully  \n %+v \n\n %+v \n\n %+v ", rsp.ChartVersionDetails, rsp.ReadmeMd, rsp.ValuesYaml)
	}

}
