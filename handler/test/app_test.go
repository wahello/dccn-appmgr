package test

import (
	"context"
	"testing"
	"time"
	appmgr "github.com/Ankr-network/dccn-common/protos/appmgr/v1/grpc"
	common_proto "github.com/Ankr-network/dccn-common/protos/common"
	usermgr "github.com/Ankr-network/dccn-common/protos/usermgr/v1/grpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

var app_test_addr = "appmgr:50051"
var token string
var testing_app_id string
var testing_ns_id string

func TestUserLogin(t *testing.T) {
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
		t.Logf(" login successfully \n %+v  \n ", rsp.AuthenticationResult)
		token = rsp.AuthenticationResult.AccessToken
	}
}

func TestCreateApp(t *testing.T) {
	conn, err := grpc.Dial(app_test_addr, grpc.WithInsecure())
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
		"authorization": token,
	})
	ctx := metadata.NewOutgoingContext(context.Background(), md)
	tokenContext, cancel := context.WithTimeout(ctx, 300 * time.Second)

	defer cancel()

	// create namespace for app_create
	rsp, err := appClient.CreateNamespace(tokenContext,
		&appmgr.CreateNamespaceRequest{Namespace: &common_proto.Namespace{
			NsName:         "app_create_1",
			NsCpuLimit:     1000,
			NsMemLimit:     2000,
			NsStorageLimit: 50000,
		}})

	t.Log("create ns successfully for app_create \n" )
	testing_ns_id = rsp.NsId
	// wait for status changed
	time.Sleep(20 * time.Second)

	// case 1: correct inputs with created namespace
	app_1 := common_proto.App{}
	app_1.AppName = "app_create_1"
	app_1.ChartDetail = &common_proto.ChartDetail{
		ChartRepo: "stable",
		ChartName: "wordpress",
		ChartVer:  "5.6.2",
	}
	app_1.NamespaceData = &common_proto.App_NsId{NsId: testing_ns_id}
	if rsp, err := appClient.CreateApp(tokenContext, &appmgr.CreateAppRequest{App: &app_1}); err != nil {
		t.Error(err)
	} else {
		t.Logf("case 1 pass: correct inputs: create app successfully : app_id %s \n ", rsp.AppId)
		testing_app_id = rsp.AppId
	}
	t.Log(testing_app_id)

	// case 2: invalid namespace
	app_2 := common_proto.App{}
	app_2.AppName = "app_create_2"
	app_2.ChartDetail = &common_proto.ChartDetail{
		ChartRepo: "stable",
		ChartName: "wordpress",
		ChartVer:  "5.6.2",
	}
	app_2.NamespaceData = &common_proto.App_NsId{NsId: ""}
	if _, err := appClient.CreateApp(tokenContext, &appmgr.CreateAppRequest{App: &app_2}); err == nil {
		t.Error(err)
	} else {
		t.Logf("case 2 pass: invalid namespace: cannot create app successfully for an invalid namespace")
	}

	// case 3: invalid chart
	app_3 := common_proto.App{}
	app_3.AppName = "app_create_3"
	app_3.ChartDetail = &common_proto.ChartDetail{
		ChartRepo: "",
		ChartName: "",
		ChartVer:  "",
	}
	app_3.NamespaceData = &common_proto.App_NsId{NsId: testing_ns_id}
	if _, err := appClient.CreateApp(tokenContext, &appmgr.CreateAppRequest{App: &app_3}); err == nil {
		t.Error(err)
	} else {
		t.Logf("case 3 pass: invalid chart: cannot create app successfully for an invalid chart")
	}

	// case 4: empty app name
	app_4 := common_proto.App{}
	app_4.AppName = ""
	app_4.ChartDetail = &common_proto.ChartDetail{
		ChartRepo: "stable",
		ChartName: "wordpress",
		ChartVer:  "5.6.2",
	}
	app_4.NamespaceData = &common_proto.App_NsId{NsId: testing_ns_id}
	if _, err := appClient.CreateApp(tokenContext, &appmgr.CreateAppRequest{App: &app_4}); err == nil {
		t.Error(err)
	} else {
		t.Logf("case 4 pass: empty app name: cannot create app successfully for an empty app name")
	}

	// wait for status changed
	time.Sleep(30 * time.Second)

	// delete the app created
	appClient.CancelApp(tokenContext, &appmgr.AppID{AppId: testing_app_id})
	time.Sleep(15 * time.Second)

	// delete the namespace created
	appClient.DeleteNamespace(tokenContext,&appmgr.DeleteNamespaceRequest{NsId: testing_ns_id})
	time.Sleep(10 * time.Second)
}

func TestAppList(t *testing.T) {
	conn, err := grpc.Dial(app_test_addr, grpc.WithInsecure())
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
		"authorization": token,
	})
	ctx := metadata.NewOutgoingContext(context.Background(), md)
	tokenContext, cancel := context.WithTimeout(ctx, 20 * time.Second)

	defer cancel()

	// case 1: correct inputs
	if res, err := appClient.AppList(tokenContext, &common_proto.Empty{}); err != nil || len(res.AppReports) < 0 {
		t.Error(err)
	} else {
		t.Logf("case 1: correct inputs: app list successfully : \n  %v  \n ", res.AppReports)
		time.Sleep(2 * time.Second)
	}
}


func TestAppDetail(t *testing.T) {

	conn, err := grpc.Dial(app_test_addr, grpc.WithInsecure())
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
		"authorization": token,
	})
	ctx := metadata.NewOutgoingContext(context.Background(), md)
	tokenContext, cancel := context.WithTimeout(ctx, 100 * time.Second)
	defer cancel()

	// create app for app_detail test
	rsp, err := appClient.CreateNamespace(tokenContext,
		&appmgr.CreateNamespaceRequest{Namespace: &common_proto.Namespace{
			NsName:         "app_create_test_1",
			NsCpuLimit:     1000,
			NsMemLimit:     2000,
			NsStorageLimit: 50000,
		}})

	t.Log("create ns successfully for app_detail \n" )
	testing_ns_id = rsp.NsId
	// wait for status changed
	time.Sleep(15 * time.Second)

	app := common_proto.App{}
	app.AppName = "app_detail_test"
	app.ChartDetail = &common_proto.ChartDetail{
		ChartRepo: "stable",
		ChartName: "wordpress",
		ChartVer:  "5.6.2",
	}
	app.NamespaceData = &common_proto.App_NsId{NsId: testing_ns_id}
	if rsp, err := appClient.CreateApp(tokenContext, &appmgr.CreateAppRequest{App: &app}); err != nil {
		t.Error(err)
	} else {
		t.Logf("create app for app_detail test successfully : appid %s \n ", rsp.AppId)
		testing_app_id = rsp.AppId
		// wait for status changed
		time.Sleep(15 * time.Second)
	}

	if _, err := appClient.AppDetail(tokenContext, &appmgr.AppID{AppId: testing_app_id}); err != nil {
		t.Error(err)
	} else {
		// t.Logf("app id %s detail successfully : \n  %v \n", testing_app_id, res.AppReport)
		time.Sleep(2 * time.Second)
	}

	// delete the app created
	appClient.CancelApp(tokenContext, &appmgr.AppID{AppId: testing_app_id})
	time.Sleep(10 * time.Second)

	// delete the namespace created
	appClient.DeleteNamespace(tokenContext,&appmgr.DeleteNamespaceRequest{NsId: testing_ns_id})
}

func TestCancelApp(t *testing.T) {

	conn, err := grpc.Dial(app_test_addr, grpc.WithInsecure())
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
		"authorization": token,
	})
	ctx := metadata.NewOutgoingContext(context.Background(), md)
	tokenContext, cancel := context.WithTimeout(ctx, 80 * time.Second)
	defer cancel()

	// create app for app_cancel test
	rsp, err := appClient.CreateNamespace(tokenContext,
		&appmgr.CreateNamespaceRequest{Namespace: &common_proto.Namespace{
			NsName:         "app_cancel_test",
			NsCpuLimit:     1000,
			NsMemLimit:     2000,
			NsStorageLimit: 50000,
		}})

	t.Log("create ns successfully for app_cancel \n" )
	testing_ns_id = rsp.NsId
	// wait for status changed
	time.Sleep(15 * time.Second)

	app := common_proto.App{}
	app.AppName = "app_cancel_test"
	app.ChartDetail = &common_proto.ChartDetail{
		ChartRepo: "stable",
		ChartName: "wordpress",
		ChartVer:  "5.6.2",
	}
	app.NamespaceData = &common_proto.App_NsId{NsId: testing_ns_id}
	if rsp, err := appClient.CreateApp(tokenContext, &appmgr.CreateAppRequest{App: &app}); err != nil {
		t.Error(err)
	} else {
		t.Logf("create app for app_cancel_test successfully : appid %s \n ", rsp.AppId)
		testing_app_id = rsp.AppId
		// wait for status changed
		time.Sleep(15 * time.Second)
	}

	if _, err := appClient.CancelApp(tokenContext,
		&appmgr.AppID{AppId: testing_app_id}); err != nil {
		t.Error(err)
	} else {
		t.Logf("cancel app successfully : appid %s \n" + testing_app_id)
		time.Sleep(2 * time.Second)
	}

	// delete the namespace created
	time.Sleep(10 * time.Second)
	appClient.DeleteNamespace(tokenContext,&appmgr.DeleteNamespaceRequest{NsId: testing_ns_id})
}

func TestUpdateApp(t *testing.T) {

	conn, err := grpc.Dial(app_test_addr, grpc.WithInsecure())
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
		"authorization": token,
	})
	ctx := metadata.NewOutgoingContext(context.Background(), md)
	tokenContext, cancel := context.WithTimeout(ctx, 80*time.Second)
	defer cancel()

	// create app for app update test
	rsp, err := appClient.CreateNamespace(tokenContext,
		&appmgr.CreateNamespaceRequest{Namespace: &common_proto.Namespace{
			NsName:         "app_update_test",
			NsCpuLimit:     1000,
			NsMemLimit:     2000,
			NsStorageLimit: 50000,
		}})

	t.Log("create ns successfully for app_update \n" )
	testing_ns_id = rsp.NsId
	// wait for status changed
	time.Sleep(15 * time.Second)

	app := common_proto.App{}
	app.AppName = "app_update_test"
	app.ChartDetail = &common_proto.ChartDetail{
		ChartRepo: "stable",
		ChartName: "wordpress",
		ChartVer:  "5.6.2",
	}
	app.NamespaceData = &common_proto.App_NsId{NsId: testing_ns_id}
	if rsp, err := appClient.CreateApp(tokenContext, &appmgr.CreateAppRequest{App: &app}); err != nil {
		t.Error(err)
	} else {
		t.Logf("create app for app_update_test successfully : appid %s \n ", rsp.AppId)
		testing_app_id = rsp.AppId
		// wait for status changed
		time.Sleep(15 * time.Second)
	}

	// case 1: correct inputs
	appDeployment := common_proto.AppDeployment{}
	appDeployment.AppId = testing_app_id
	t.Logf("APP ID: %s", appDeployment.AppId)
	appDeployment.AppName = "app_update_test"
	appDeployment.ChartDetail = &common_proto.ChartDetail{
		ChartRepo: "stable",
		ChartName: "wordpress",
		ChartVer:  "5.6.2",
	}

	if _, err := appClient.UpdateApp(tokenContext, &appmgr.UpdateAppRequest{AppDeployment: &appDeployment}); err != nil {
		t.Error(err)
	} else {
		time.Sleep(10 * time.Second)
		updateApp, _ := appClient.AppDetail(tokenContext, &appmgr.AppID{AppId: testing_app_id})
		if updateApp.AppReport.AppDeployment.AppName == "app_update_test" {
			t.Logf("update app successfully : appid %s \n ", appDeployment.AppId)
		}else{
			t.Error(err)
		}
	}

	// delete the app created
	appClient.CancelApp(tokenContext, &appmgr.AppID{AppId: testing_app_id})
	time.Sleep(10 * time.Second)

	// delete the namespace created
	appClient.DeleteNamespace(tokenContext,&appmgr.DeleteNamespaceRequest{NsId: testing_ns_id})
}







