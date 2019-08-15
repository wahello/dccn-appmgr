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
	app := common_proto.App{}
	app.AppName = "zilliqa_test"
	app.ChartDetail = &common_proto.ChartDetail{
		ChartRepo: "stable",
		ChartName: "zilliqa",
		ChartVer:  "0.1.0",
	}
	app.NamespaceData = &common_proto.App_Namespace{
		Namespace: &common_proto.Namespace{
			NsName:         "test_ns1",
			NsCpuLimit:     1000,
			NsMemLimit:     2000,
			NsStorageLimit: 50000,
		},
	}

	var customValues []*common_proto.CustomValue
	customValues = append(customValues, &common_proto.CustomValue{
		Key:   "seed_port",
		Value: "32137",
	})
	customValues = append(customValues, &common_proto.CustomValue{
		Key:   "pubkey",
		Value: "03AB2115FA0FF77359B38FD14883B16412C7BF652EAD2C218C4B4F56F1ADFB3B89",
	})
	customValues = append(customValues, &common_proto.CustomValue{
		Key:   "prikey",
		Value: "8F0FA8FD1849DA7A2B6B02404B60D3BC3D723603C4CAEDEE086B57F28B90798C",
	})
	app.CustomValues = customValues

	if rsp, err := appClient.CreateApp(tokenContext, &appmgr.CreateAppRequest{App: &app}); err != nil {
		log.Fatal(err)
	} else {
		log.Println("create app successfully : appid   " + rsp.AppId)
	}

}
