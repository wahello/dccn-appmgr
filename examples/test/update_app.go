package main

import (
	"context"
	"os"

	//	"github.com/Ankr-network/dccn-common/protos"

	"log"
	"time"

	appmgr "github.com/Ankr-network/dccn-common/protos/appmgr/v1/grpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"

	//usermgr "github.com/Ankr-network/dccn-common/protos/usermgr/v1/grpc"
	common_proto "github.com/Ankr-network/dccn-common/protos/common"
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
		"authorization": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJleHAiOjE1NjMyMzg5NTEsImp0aSI6IjU1NzNhYjY3LTQ0YTUtNGY2Yi1iMjY2LTY3MzA1MjcyZWEzMSIsImlzcyI6ImFua3IuY29tIn0.-8NckBOtNjuhMC2B4CgYYtK9qmzFa2IGA0zjpfPDFgw",
	})

	ctx := metadata.NewOutgoingContext(context.Background(), md)
	//
	tokenContext, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	//
	appDeployment := common_proto.AppDeployment{}
	appDeployment.AppId = os.Args[1]
	log.Printf("APP ID: %s", appDeployment.AppId)
	appDeployment.AppName = "zilliqa_test2"
	appDeployment.ChartDetail = &common_proto.ChartDetail{
		ChartRepo: "stable",
		ChartName: "zilliqa",
		ChartVer:  "0.1.0",
	}
	var customValues []*common_proto.CustomValue
	customValues = append(customValues, &common_proto.CustomValue{
		Key: "seed_port"
		Value: "32138"
	})
	customValues = append(customValues, &common_proto.CustomValue{
		Key: "pubkey"
		Value: "034D4483544DB5BF71D4C91E9B920855B2577B8B54A0B30AC60EB81A5260C583A4"
	})
	customValues = append(customValues, &common_proto.CustomValue{
		Key: "prikey"
		Value: "F2BA7F7C9582D1CEE5423DE15017C386D63E6796730BAC9673A5B73E3CCF2984"
	})
	if _, err := appClient.UpdateApp(tokenContext, &appmgr.UpdateAppRequest{AppDeployment: &appDeployment}); err != nil {
		log.Fatal(err)
	} else {
		log.Println("update app successfully : appid   " + appDeployment.AppId)
	}

}
