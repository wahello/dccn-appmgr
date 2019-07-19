package main

import (
	"bytes"
	"context"

	//	"github.com/Ankr-network/dccn-common/protos"

	"log"
	"time"

	appmgr "github.com/Ankr-network/dccn-common/protos/appmgr/v1/grpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	"k8s.io/helm/pkg/chartutil"
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
		"authorization": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJleHAiOjE1NjMyMzg5NTEsImp0aSI6IjU1NzNhYjY3LTQ0YTUtNGY2Yi1iMjY2LTY3MzA1MjcyZWEzMSIsImlzcyI6ImFua3IuY29tIn0.-8NckBOtNjuhMC2B4CgYYtK9qmzFa2IGA0zjpfPDFgw",
	})

	ctx := metadata.NewOutgoingContext(context.Background(), md)
	//
	tokenContext, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	//

	if rsp, err := appClient.DownloadChart(tokenContext, &appmgr.DownloadChartRequest{
		ChartVer:  "5.6.2",
		ChartName: "wordpress",
		ChartRepo: "stable",
	}); err != nil {
		log.Fatal(err)
	} else {
		loadedChart, err := chartutil.LoadArchive(bytes.NewReader(rsp.ChartFile))
		if err != nil {
			log.Panic(err.Error())
		}
		log.Printf(" chart download successfully \n\n %+v \n\n %+v ", loadedChart.Metadata.Version, loadedChart.Metadata.Name)
	}

}
