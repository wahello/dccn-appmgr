package handler

import (
	"bytes"
	"context"
	"io/ioutil"
	"os"
	"testing"
	"time"

	appmgr "github.com/Ankr-network/dccn-common/protos/appmgr/v1/grpc"
	common_proto "github.com/Ankr-network/dccn-common/protos/common"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	"k8s.io/helm/pkg/chartutil"
)

var addr = "appmgr:50051"

var testing_ns_id string
var testing_app_id string

func TestChartList(t *testing.T) {

	conn, err := grpc.Dial(addr, grpc.WithInsecure())
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
		"token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJleHAiOjE1NTUwMTUzMTgsImp0aSI6IjQ4NTQ5YjQxLWUzNjYtNGIxMi05NTc3LTU0M2Y5NTE5Y2JlZiIsImlzcyI6ImFua3IubmV0d29yayJ9.A0p3KyxIKZHAZb_buPgadKj3d40Rlw_hSpsFBrNLjuw",
	})
	ctx := metadata.NewOutgoingContext(context.Background(), md)
	tokenContext, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	if rsp, err := appClient.ChartList(tokenContext, &appmgr.ChartListRequest{
		ChartRepo: "stable",
	}); err != nil {
		t.Error(err)
	} else {
		t.Logf(" chart list successfully  \n %+v  \n ", rsp.Charts)
	}

}

func TestChartDetail(t *testing.T) {

	conn, err := grpc.Dial(addr, grpc.WithInsecure())
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
		"token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJleHAiOjE1NTUwMTUzMTgsImp0aSI6IjQ4NTQ5YjQxLWUzNjYtNGIxMi05NTc3LTU0M2Y5NTE5Y2JlZiIsImlzcyI6ImFua3IubmV0d29yayJ9.A0p3KyxIKZHAZb_buPgadKj3d40Rlw_hSpsFBrNLjuw",
	})
	ctx := metadata.NewOutgoingContext(context.Background(), md)
	tokenContext, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	if rsp, err := appClient.ChartDetail(tokenContext, &appmgr.ChartDetailRequest{
		Chart: &common_proto.Chart{
			ChartName: "wordpress",
			ChartRepo: "stable"},
		ShowVersion: "5.6.2",
	}); err != nil {
		t.Error(err)
	} else {
		t.Logf(" chart list successfully  \n %+v \n\n %+v \n\n %+v  \n ", rsp.ChartVersionDetails, rsp.ReadmeMd, rsp.ValuesYaml)
	}

}

func TestDownloadChart(t *testing.T) {

	conn, err := grpc.Dial(addr, grpc.WithInsecure())
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
		"token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJleHAiOjE1NTUwMTUzMTgsImp0aSI6IjQ4NTQ5YjQxLWUzNjYtNGIxMi05NTc3LTU0M2Y5NTE5Y2JlZiIsImlzcyI6ImFua3IubmV0d29yayJ9.A0p3KyxIKZHAZb_buPgadKj3d40Rlw_hSpsFBrNLjuw",
	})
	ctx := metadata.NewOutgoingContext(context.Background(), md)
	tokenContext, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

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
		dest, err := os.Getwd()
		if err != nil {
			t.Error(err)
		}
		name, err := chartutil.Save(loadedChart, dest)
		if err == nil {
			t.Logf("Successfully download chart and saved it to: %s\n", name)
		} else {
			t.Error(err)
		}
	}
}

func TestCreateNamespace(t *testing.T) {

	conn, err := grpc.Dial(addr, grpc.WithInsecure())
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
		"token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJleHAiOjE1NTUwMTUzMTgsImp0aSI6IjQ4NTQ5YjQxLWUzNjYtNGIxMi05NTc3LTU0M2Y5NTE5Y2JlZiIsImlzcyI6ImFua3IubmV0d29yayJ9.A0p3KyxIKZHAZb_buPgadKj3d40Rlw_hSpsFBrNLjuw",
	})
	ctx := metadata.NewOutgoingContext(context.Background(), md)
	tokenContext, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	if rsp, err := appClient.CreateNamespace(tokenContext,
		&appmgr.CreateNamespaceRequest{Namespace: &common_proto.Namespace{
			NsName:         "test_ns",
			NsCpuLimit:     1000,
			NsMemLimit:     2000,
			NsStorageLimit: 50000,
		}}); err != nil {
		t.Error(err)
	} else {
		t.Log("create ns successfully : nsid   \n  " + rsp.NsId)
		testing_ns_id = rsp.NsId
		time.Sleep(5 * time.Second)
	}
}

func TestNamespaceList(t *testing.T) {

	conn, err := grpc.Dial(addr, grpc.WithInsecure())
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
		"token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJleHAiOjE1NTUwMTUzMTgsImp0aSI6IjQ4NTQ5YjQxLWUzNjYtNGIxMi05NTc3LTU0M2Y5NTE5Y2JlZiIsImlzcyI6ImFua3IubmV0d29yayJ9.A0p3KyxIKZHAZb_buPgadKj3d40Rlw_hSpsFBrNLjuw",
	})
	ctx := metadata.NewOutgoingContext(context.Background(), md)
	tokenContext, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	if rsp, err := appClient.NamespaceList(tokenContext, &common_proto.Empty{}); err != nil {
		t.Error(err)
	} else {
		t.Logf("namespace list successfully: \n %+v  \n ", rsp.NamespaceReports)
		time.Sleep(2 * time.Second)
	}

}

func TestCreateApp(t *testing.T) {

	conn, err := grpc.Dial(addr, grpc.WithInsecure())
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
		"token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJleHAiOjE1NTUwMTUzMTgsImp0aSI6IjQ4NTQ5YjQxLWUzNjYtNGIxMi05NTc3LTU0M2Y5NTE5Y2JlZiIsImlzcyI6ImFua3IubmV0d29yayJ9.A0p3KyxIKZHAZb_buPgadKj3d40Rlw_hSpsFBrNLjuw",
	})
	ctx := metadata.NewOutgoingContext(context.Background(), md)
	tokenContext, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	app := common_proto.App{}
	app.AppName = "wordpress_test"
	app.ChartDetail = &common_proto.ChartDetail{
		ChartRepo: "stable",
		ChartName: "wordpress",
		ChartVer:  "5.6.2",
	}
	app.NamespaceData = &common_proto.App_NsId{NsId: testing_ns_id}
	if rsp, err := appClient.CreateApp(tokenContext, &appmgr.CreateAppRequest{App: &app}); err != nil {
		t.Error(err)
	} else {
		t.Logf("create app successfully : appid %s  \n ", rsp.AppId)
		testing_app_id = rsp.AppId
		time.Sleep(5 * time.Second)
	}
}

func TestAppList(t *testing.T) {

	conn, err := grpc.Dial(addr, grpc.WithInsecure())
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
		"token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJleHAiOjE1NTUwMTUzMTgsImp0aSI6IjQ4NTQ5YjQxLWUzNjYtNGIxMi05NTc3LTU0M2Y5NTE5Y2JlZiIsImlzcyI6ImFua3IubmV0d29yayJ9.A0p3KyxIKZHAZb_buPgadKj3d40Rlw_hSpsFBrNLjuw",
	})
	ctx := metadata.NewOutgoingContext(context.Background(), md)
	tokenContext, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	if res, err := appClient.AppList(tokenContext, &common_proto.Empty{}); err != nil {
		t.Error(err)
	} else {
		t.Logf("app list successfully : \n  %v  \n ", res.AppReports)
		time.Sleep(2 * time.Second)
	}
}

func TestUpdateNamespace(t *testing.T) {

	conn, err := grpc.Dial(addr, grpc.WithInsecure())
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
		"token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJleHAiOjE1NTUwMTUzMTgsImp0aSI6IjQ4NTQ5YjQxLWUzNjYtNGIxMi05NTc3LTU0M2Y5NTE5Y2JlZiIsImlzcyI6ImFua3IubmV0d29yayJ9.A0p3KyxIKZHAZb_buPgadKj3d40Rlw_hSpsFBrNLjuw",
	})
	ctx := metadata.NewOutgoingContext(context.Background(), md)
	tokenContext, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	if _, err := appClient.UpdateNamespace(tokenContext,
		&appmgr.UpdateNamespaceRequest{Namespace: &common_proto.Namespace{
			NsId:           testing_ns_id,
			NsCpuLimit:     2000,
			NsMemLimit:     4000,
			NsStorageLimit: 100000,
		}}); err != nil {
		t.Error(err)
	} else {
		t.Log("update namespace successfully  \n ")
		time.Sleep(5 * time.Second)
	}
}

func TestUploadChart(t *testing.T) {

	conn, err := grpc.Dial(addr, grpc.WithInsecure())
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
		"token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJleHAiOjE1NTUwMTUzMTgsImp0aSI6IjQ4NTQ5YjQxLWUzNjYtNGIxMi05NTc3LTU0M2Y5NTE5Y2JlZiIsImlzcyI6ImFua3IubmV0d29yayJ9.A0p3KyxIKZHAZb_buPgadKj3d40Rlw_hSpsFBrNLjuw",
	})
	ctx := metadata.NewOutgoingContext(context.Background(), md)
	tokenContext, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	file, err := ioutil.ReadFile("wordpress-5.6.2.tgz")
	if err != nil {
		t.Error(err)
	}
	if _, err := appClient.UploadChart(tokenContext, &appmgr.UploadChartRequest{
		ChartFile: file,
		ChartName: "wordpress",
		ChartVer:  "5.6.3",
		ChartRepo: "stable"}); err != nil {
		t.Error(err)
	} else {
		t.Log("upload chart successfully")
		time.Sleep(2 * time.Second)
	}
}
func TestUpdateApp(t *testing.T) {

	conn, err := grpc.Dial(addr, grpc.WithInsecure())
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
		"token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJleHAiOjE1NTUwMTUzMTgsImp0aSI6IjQ4NTQ5YjQxLWUzNjYtNGIxMi05NTc3LTU0M2Y5NTE5Y2JlZiIsImlzcyI6ImFua3IubmV0d29yayJ9.A0p3KyxIKZHAZb_buPgadKj3d40Rlw_hSpsFBrNLjuw",
	})
	ctx := metadata.NewOutgoingContext(context.Background(), md)
	tokenContext, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	appDeployment := common_proto.AppDeployment{}
	appDeployment.AppId = testing_app_id
	t.Logf("APP ID: %s", appDeployment.AppId)
	appDeployment.AppName = "wordpress_test1"
	appDeployment.ChartDetail = &common_proto.ChartDetail{
		ChartRepo: "stable",
		ChartName: "wordpress",
		ChartVer:  "5.6.3",
	}

	if _, err := appClient.UpdateApp(tokenContext, &appmgr.UpdateAppRequest{AppDeployment: &appDeployment}); err != nil {
		t.Error(err)
	} else {
		t.Logf("update app successfully : appid %s \n ", appDeployment.AppId)
		time.Sleep(2 * time.Second)
	}
}

func TestDeleteChart(t *testing.T) {

	conn, err := grpc.Dial(addr, grpc.WithInsecure())
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
		"token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJleHAiOjE1NTUwMTUzMTgsImp0aSI6IjQ4NTQ5YjQxLWUzNjYtNGIxMi05NTc3LTU0M2Y5NTE5Y2JlZiIsImlzcyI6ImFua3IubmV0d29yayJ9.A0p3KyxIKZHAZb_buPgadKj3d40Rlw_hSpsFBrNLjuw",
	})
	ctx := metadata.NewOutgoingContext(context.Background(), md)
	tokenContext, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	if _, err := appClient.DeleteChart(tokenContext, &appmgr.DeleteChartRequest{
		ChartVer:  "5.6.3",
		ChartName: "wordpress",
		ChartRepo: "stable",
	}); err != nil {
		t.Error(err)
	} else {
		t.Log("delete chart successfully \n")
		time.Sleep(2 * time.Second)
	}
}

func TestAppDetail(t *testing.T) {

	conn, err := grpc.Dial(addr, grpc.WithInsecure())
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
		"token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJleHAiOjE1NTUwMTUzMTgsImp0aSI6IjQ4NTQ5YjQxLWUzNjYtNGIxMi05NTc3LTU0M2Y5NTE5Y2JlZiIsImlzcyI6ImFua3IubmV0d29yayJ9.A0p3KyxIKZHAZb_buPgadKj3d40Rlw_hSpsFBrNLjuw",
	})
	ctx := metadata.NewOutgoingContext(context.Background(), md)
	tokenContext, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	if res, err := appClient.AppDetail(tokenContext, &appmgr.AppID{AppId: testing_app_id}); err != nil {
		t.Error(err)
	} else {
		t.Logf("app id %s detail successfully : \n  %v \n", testing_app_id, res.AppReport)
		time.Sleep(2 * time.Second)
	}
}

func TestCancelApp(t *testing.T) {

	conn, err := grpc.Dial(addr, grpc.WithInsecure())
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
		"token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJleHAiOjE1NTUwMTUzMTgsImp0aSI6IjQ4NTQ5YjQxLWUzNjYtNGIxMi05NTc3LTU0M2Y5NTE5Y2JlZiIsImlzcyI6ImFua3IubmV0d29yayJ9.A0p3KyxIKZHAZb_buPgadKj3d40Rlw_hSpsFBrNLjuw",
	})
	ctx := metadata.NewOutgoingContext(context.Background(), md)
	tokenContext, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	if _, err := appClient.CancelApp(tokenContext,
		&appmgr.AppID{AppId: testing_app_id}); err != nil {
		t.Error(err)
	} else {
		t.Logf("cancel app successfully : appid %s \n" + testing_app_id)
		time.Sleep(2 * time.Second)
	}
}

func TestCancelNamespace(t *testing.T) {

	conn, err := grpc.Dial(addr, grpc.WithInsecure())
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
		"token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJleHAiOjE1NTUwMTUzMTgsImp0aSI6IjQ4NTQ5YjQxLWUzNjYtNGIxMi05NTc3LTU0M2Y5NTE5Y2JlZiIsImlzcyI6ImFua3IubmV0d29yayJ9.A0p3KyxIKZHAZb_buPgadKj3d40Rlw_hSpsFBrNLjuw",
	})
	ctx := metadata.NewOutgoingContext(context.Background(), md)
	tokenContext, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	if _, err := appClient.DeleteNamespace(tokenContext,
		&appmgr.DeleteNamespaceRequest{NsId: testing_ns_id}); err != nil {
		t.Error(err)
	} else {
		t.Log("delete ns successfully\n")
		time.Sleep(2 * time.Second)
	}
}
