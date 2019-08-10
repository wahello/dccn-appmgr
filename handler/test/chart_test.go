package test

import (
	"bytes"
	"context"
	"io/ioutil"
	"os"
	"testing"
	"time"
	appmgr "github.com/Ankr-network/dccn-common/protos/appmgr/v1/grpc"
	common_proto "github.com/Ankr-network/dccn-common/protos/common"
	usermgr "github.com/Ankr-network/dccn-common/protos/usermgr/v1/grpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	"k8s.io/helm/pkg/chartutil"
)

var chart_test_addr = "appmgr:50051"
var chart_test_token string

func TestChartUserLogin(t *testing.T) {

	conn, err := grpc.Dial("usermgr:50051", grpc.WithInsecure())
	if err != nil {
		t.Error(err)
	}
	defer func(conn *grpc.ClientConn) {
		if err := conn.Close(); err != nil {
			t.Error(err)
		}
	}(conn)
	userClient := usermgr.NewUserMgrClient(conn)
	ctx, cancel := context.WithTimeout(context.Background(),
		60*time.Second)
	defer cancel()

	if rsp, err := userClient.Login(ctx, &usermgr.LoginRequest{
		Email:    "test12345@mailinator.com",
		Password: "test12345",
	}); err != nil {
		t.Error(err)
	} else {
		t.Logf(" login successfully  \n %+v  \n ", rsp.AuthenticationResult)
		chart_test_token = rsp.AuthenticationResult.AccessToken
	}

}

func TestChartList(t *testing.T) {

	conn, err := grpc.Dial(chart_test_addr, grpc.WithInsecure())
	if err != nil {
		t.Error(err)
	}
	defer func(conn *grpc.ClientConn) {
		if err := conn.Close(); err != nil {
			t.Error(err)
		}
	}(conn)
	appClient := appmgr.NewAppMgrClient(conn)

	// valid token
	md := metadata.New(map[string]string{
		"authorization": chart_test_token,
	})
	ctx := metadata.NewOutgoingContext(context.Background(), md)
	tokenContext, cancel := context.WithTimeout(ctx, 10*time.Second)

	// invalid token
	md_invalid := metadata.New(map[string]string{
		"authorization": "",
	})
	ctx_invalid := metadata.NewOutgoingContext(context.Background(), md_invalid)
	tokenContext_invalid, _ := context.WithTimeout(ctx_invalid, 10*time.Second)

	defer cancel()

	// case 1: invalid access token
	if _, err := appClient.ChartList(tokenContext_invalid, &appmgr.ChartListRequest{
		ChartRepo: "stable",
	}); err != nil {
		t.Error(err)
	} else {
		t.Logf("can list stable chart successfully for invalid access token \n")
	}

	// case 2: correct inputs
	if rsp, err := appClient.ChartList(tokenContext, &appmgr.ChartListRequest{
		ChartRepo: "stable",
	}); err != nil || len(rsp.Charts) < 0 {
		t.Error(err)
	} else {
		t.Logf(" chart list successfully  \n %+v  \n ", rsp.Charts)
	}

}

func TestChartDetail(t *testing.T) {

	conn, err := grpc.Dial(chart_test_addr, grpc.WithInsecure())
	if err != nil {
		t.Error(err)
	}
	defer func(conn *grpc.ClientConn) {
		if err := conn.Close(); err != nil {
			t.Error(err)
		}
	}(conn)
	appClient := appmgr.NewAppMgrClient(conn)

	// valid access token
	md := metadata.New(map[string]string{
		"authorization": chart_test_token,
	})
	ctx := metadata.NewOutgoingContext(context.Background(), md)
	tokenContext, cancel := context.WithTimeout(ctx, 10*time.Second)

	// invalid access token
	md_invalid := metadata.New(map[string]string{
		"authorization": "",
	})
	ctx_invalid := metadata.NewOutgoingContext(context.Background(), md_invalid)
	tokenContext_invalid, _ := context.WithTimeout(ctx_invalid, 10*time.Second)

	defer cancel()
	// case 1: invalid access token for stable chart
	if _, err := appClient.ChartDetail(tokenContext_invalid, &appmgr.ChartDetailRequest{
		Chart: &common_proto.Chart{
			ChartName: "wordpress",
			ChartRepo: "stable"},
		ShowVersion: "5.6.2",
	}); err != nil {
		t.Error(err)
	} else {
		t.Logf("can list stable chart successfully for invalid access token \n ")
	}

	// case 2: correct inputs
	if rsp, err := appClient.ChartDetail(tokenContext, &appmgr.ChartDetailRequest{
		Chart: &common_proto.Chart{
			ChartName: "wordpress",
			ChartRepo: "stable"},
		ShowVersion: "5.6.2",
	}); err != nil {
		t.Error(err)
	} else {
		t.Logf(" list chart details successfully  \n %+v \n\n %+v \n\n %+v  \n ", rsp.ChartVersionDetails, rsp.ReadmeMd, rsp.ValuesYaml)
	}
}

func TestDownloadChart(t *testing.T) {

	conn, err := grpc.Dial(chart_test_addr, grpc.WithInsecure())
	if err != nil {
		t.Error(err)
	}
	defer func(conn *grpc.ClientConn) {
		if err := conn.Close(); err != nil {
			t.Error(err)
		}
	}(conn)
	appClient := appmgr.NewAppMgrClient(conn)

	// valid access token
	md := metadata.New(map[string]string{
		"authorization": chart_test_token,
	})
	ctx := metadata.NewOutgoingContext(context.Background(), md)
	tokenContext, cancel := context.WithTimeout(ctx, 10*time.Second)

	// invalid access token
	md_invalid := metadata.New(map[string]string{
		"authorization": "",
	})
	ctx_invalid := metadata.NewOutgoingContext(context.Background(), md_invalid)
	tokenContext_invalid, _ := context.WithTimeout(ctx_invalid, 10*time.Second)

	defer cancel()

	// case 1: invalid access token
	if _, err := appClient.DownloadChart(tokenContext_invalid, &appmgr.DownloadChartRequest{
		ChartVer:  "5.6.2",
		ChartName: "wordpress",
		ChartRepo: "stable",
	}); err != nil {
		t.Error(err)
	}else{
		t.Logf("can download stable chart successfully for invalid access token \n ")
	}

	// case 2: correct inputs
	if rsp, err := appClient.DownloadChart(tokenContext, &appmgr.DownloadChartRequest{
		ChartVer:  "5.6.2",
		ChartName: "wordpress",
		ChartRepo: "stable",
	}); err != nil {
		t.Error(err)
	} else {
		loadedChart, err := chartutil.LoadArchive(bytes.NewReader(rsp.ChartFile))
		if err != nil {
			t.Error(err)
		}
		t.Logf(" chart download successfully \n\n %+v \n\n %+v ", loadedChart.Metadata.Version, loadedChart.Metadata.Name)
	}
}


func TestSaveAsChart(t *testing.T) {
	conn, err := grpc.Dial(chart_test_addr, grpc.WithInsecure())
	if err != nil {
		t.Error(err)
	}
	defer func(conn *grpc.ClientConn) {
		if err := conn.Close(); err != nil {
			t.Error(err)
		}
	}(conn)
	appClient := appmgr.NewAppMgrClient(conn)

	md := metadata.New(map[string]string{
		"authorization": chart_test_token,
	})
	ctx := metadata.NewOutgoingContext(context.Background(), md)
	tokenContext, cancel := context.WithTimeout(ctx, 10*time.Second)

	defer cancel()

	// download a chart for chart_saveas test
	if rsp, _ := appClient.DownloadChart(tokenContext, &appmgr.DownloadChartRequest{
		ChartVer:  "5.6.2",
		ChartName: "wordpress",
		ChartRepo: "stable",
	}); err != nil {
		t.Error(err)
	} else {
		loadedChart, err := chartutil.LoadArchive(bytes.NewReader(rsp.ChartFile))
		if err != nil {
			t.Error(err)
		}
		t.Logf(" chart download successfully for chart_saveas test \n")
		dest, err := os.Getwd()
		if err != nil {
			t.Error(err)
		}
		name, err := chartutil.Save(loadedChart, dest)
		if err == nil {
			t.Logf("Successfully saved the chart to: %s\n", name)
		} else {
			t.Error(err)
		}
	}
}

func TestUploadChart(t *testing.T) {

	conn, err := grpc.Dial(chart_test_addr, grpc.WithInsecure())
	if err != nil {
		t.Error(err)
	}
	defer func(conn *grpc.ClientConn) {
		if err := conn.Close(); err != nil {
			t.Error(err)
		}
	}(conn)
	appClient := appmgr.NewAppMgrClient(conn)

	// valid access token
	md := metadata.New(map[string]string{
		"authorization": chart_test_token,
	})
	ctx := metadata.NewOutgoingContext(context.Background(), md)
	tokenContext, cancel := context.WithTimeout(ctx, 10*time.Second)

	defer cancel()

	// download a chart for upload
	if _, err := appClient.DownloadChart(tokenContext, &appmgr.DownloadChartRequest{
		ChartVer:  "5.6.2",
		ChartName: "wordpress",
		ChartRepo: "stable",
	}); err != nil {
		t.Error(err)
	} else {
		t.Logf(" chart download successfully for chart_upload test \n")
		file, err := ioutil.ReadFile("wordpress-5.6.2.tgz")
		if err != nil {
			t.Error(err)
		}
		if _, err := appClient.UploadChart(tokenContext, &appmgr.UploadChartRequest{
			ChartFile: file,
			ChartName: "chart_upload_test",
			ChartVer:  "9.9.9",
			ChartRepo: "user"}); err != nil {
			t.Error(err)
		} else {
			t.Log("upload chart successfully")
			time.Sleep(2 * time.Second)
		}
	}

	// delete the chart
	appClient.DeleteChart(tokenContext, &appmgr.DeleteChartRequest{
		ChartVer:  "9.9.9",
		ChartName: "chart_upload_test",
		ChartRepo: "user",})
}


func TestDeleteChart(t *testing.T) {

	conn, err := grpc.Dial(chart_test_addr, grpc.WithInsecure())
	if err != nil {
		t.Error(err)
	}
	defer func(conn *grpc.ClientConn) {
		if err := conn.Close(); err != nil {
			t.Error(err)
		}
	}(conn)
	appClient := appmgr.NewAppMgrClient(conn)
	md := metadata.New(map[string]string{
		"authorization": chart_test_token,
	})
	ctx := metadata.NewOutgoingContext(context.Background(), md)
	tokenContext, cancel := context.WithTimeout(ctx, 20*time.Second)
	defer cancel()

	// create chart for chart_delete test
	if _, err := appClient.DownloadChart(tokenContext, &appmgr.DownloadChartRequest{
		ChartVer:  "5.6.2",
		ChartName: "wordpress",
		ChartRepo: "stable",
	}); err != nil {
		t.Error(err)
	} else {
		t.Logf(" chart download successfully for chart_delete test \n")
		file, err := ioutil.ReadFile("wordpress-5.6.2.tgz")
		if err != nil {
			t.Error(err)
		}
		if _, err := appClient.UploadChart(tokenContext, &appmgr.UploadChartRequest{
			ChartFile: file,
			ChartName: "chart_delete_test",
			ChartVer:  "8.8.8",
			ChartRepo: "user"}); err != nil {
			t.Error(err)
		}
	}

	// chart delete test
	if _, err := appClient.DeleteChart(tokenContext, &appmgr.DeleteChartRequest{
		ChartVer:  "8.8.8",
		ChartName: "chart_delete_test",
		ChartRepo: "user",
	}); err != nil {
		t.Error(err)
	} else {
		// wait for chart status changed
		time.Sleep(5 * time.Second)
		ChartList, _ := appClient.ChartList(tokenContext, &appmgr.ChartListRequest{
			ChartRepo: "user",
		})
		t.Log(ChartList.Charts)
		for i := 0; i < len(ChartList.Charts); i++ {
			if ChartList.Charts[i].ChartName == "chart_delete_test" {
				t.Error(err)
			}
		}
		t.Log("delete chart successfully \n")
		time.Sleep(2 * time.Second)
	}
}