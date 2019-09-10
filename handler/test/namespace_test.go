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

var ns_test_addr = "appmgr:50051"
var ns_test_token string

// login to get token before other tests
func TestNsUserLogin(t *testing.T) {
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
		ns_test_token = rsp.AuthenticationResult.AccessToken
	}
}

func TestCreateNamespace(t *testing.T) {

	conn, err := grpc.Dial(ns_test_addr, grpc.WithInsecure())
	if err != nil {
		t.Error(err)
	}
	defer func(conn *grpc.ClientConn) {
		if err := conn.Close(); err != nil {
			t.Error(err)
		}
	}(conn)
	appClient := appmgr.NewAppMgrClient(conn)
	var testing_ns_id string

	// valid access token
	md := metadata.New(map[string]string{
		"authorization": ns_test_token,
	})
	ctx := metadata.NewOutgoingContext(context.Background(), md)
	tokenContext, cancel := context.WithTimeout(ctx, 120 * time.Second)

	defer cancel()

	// case 1: correct inputs
	if rsp, err := appClient.CreateNamespace(tokenContext,
		&appmgr.CreateNamespaceRequest{Namespace: &common_proto.Namespace{
			NsName:         "ns_create_1",
			NsCpuLimit:     1000,
			NsMemLimit:     2000,
			NsStorageLimit: 50000,
		}}); err != nil || len(rsp.NsId) <= 0 {
		t.Error(err)
	} else {
		t.Log("case 1: correct inputs: create ns successfully : nsid   \n  " + rsp.NsId)
		testing_ns_id = rsp.NsId
		time.Sleep(15 * time.Second)
	}

	// case 2: empty inputs
	if _, err := appClient.CreateNamespace(tokenContext,
		&appmgr.CreateNamespaceRequest{Namespace: &common_proto.Namespace{
			NsName:         "ns_create_2",
			NsCpuLimit:     0,
			NsMemLimit:     0,
			NsStorageLimit: 0,
		}}); err == nil {
		t.Error(err)
	} else {
		t.Log("case 2: empty inputs: cannot create ns for empty properties inputs \n  ")
	}

	// delete namespace created
	appClient.DeleteNamespace(tokenContext, &appmgr.DeleteNamespaceRequest{NsId: testing_ns_id})
}

func TestNamespaceList(t *testing.T) {

	conn, err := grpc.Dial(ns_test_addr, grpc.WithInsecure())
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
		"authorization": ns_test_token,
	})
	ctx := metadata.NewOutgoingContext(context.Background(), md)
	tokenContext, cancel := context.WithTimeout(ctx, 120 * time.Second)

	defer cancel()

	// case 1: correct inputs
	if rsp, err := appClient.NamespaceList(tokenContext, &common_proto.Empty{}); err != nil || len(rsp.NamespaceReports) < 0 {
		t.Error(err)
	} else {
		t.Logf("case 1: correct inputs: namespace list successfully: \n %+v  \n ", rsp.NamespaceReports)
		time.Sleep(2 * time.Second)
	}
}

func TestUpdateNamespace(t *testing.T) {
	conn, err := grpc.Dial(ns_test_addr, grpc.WithInsecure())
	if err != nil {
		t.Error(err)
	}
	defer func(conn *grpc.ClientConn) {
		if err := conn.Close(); err != nil {
			t.Error(err)
		}
	}(conn)
	appClient := appmgr.NewAppMgrClient(conn)
	var testing_ns_id string

	// valid access token
	md := metadata.New(map[string]string{
		"authorization": ns_test_token,
	})
	ctx := metadata.NewOutgoingContext(context.Background(), md)
	tokenContext, cancel := context.WithTimeout(ctx, 120 * time.Second)

	defer cancel()

	// create ns for update
	if rsp, err := appClient.CreateNamespace(tokenContext,
		&appmgr.CreateNamespaceRequest{Namespace: &common_proto.Namespace{
			NsName:         "ns_update_1",
			NsCpuLimit:     1000,
			NsMemLimit:     2000,
			NsStorageLimit: 50000,
		}}); err != nil && len(rsp.NsId) > 0 {
		t.Error(err)
	} else {
		t.Log("create ns for ns_update successfully : nsid   \n  " + rsp.NsId)
		testing_ns_id = rsp.NsId
	}

	// wait for namespace status changed
	time.Sleep(15 * time.Second)
	// nsList1, _ := appClient.NamespaceList(tokenContext, &common_proto.Empty{})
	// t.Log(nsList1)

	// case 1: correct inputs
	if _, err := appClient.UpdateNamespace(tokenContext,
		&appmgr.UpdateNamespaceRequest{Namespace: &common_proto.Namespace{
			NsId:           testing_ns_id,
			NsCpuLimit:     2000,
			NsMemLimit:     4000,
			NsStorageLimit: 100000,
		}}); err != nil {
		t.Error(err)
	} else {
		// wait for ns_status changed
		time.Sleep(15 * time.Second)
		// check update ns results
		nsList, _ := appClient.NamespaceList(tokenContext, &common_proto.Empty{})
		// t.Log(nsList)
		for i := 0; i < len(nsList.NamespaceReports); i++ {
			if nsList.NamespaceReports[i].Namespace.NsId == testing_ns_id {
				if nsList.NamespaceReports[i].Namespace.NsCpuLimit != 2000 || nsList.NamespaceReports[i].Namespace.NsMemLimit != 4000 || nsList.NamespaceReports[i].Namespace.NsStorageLimit != 100000 {
					t.Error(err)
				}
				break
			}
		}
		t.Log("case 1: correct inputs: update namespace successfully \n ")
	}

	// delete namespace created
	appClient.DeleteNamespace(tokenContext, &appmgr.DeleteNamespaceRequest{NsId: testing_ns_id})
}

func TestCancelNamespace(t *testing.T) {
	conn, err := grpc.Dial(ns_test_addr, grpc.WithInsecure())
	if err != nil {
		t.Error(err)
	}
	defer func(conn *grpc.ClientConn) {
		if err := conn.Close(); err != nil {
			t.Error(err)
		}
	}(conn)
	appClient := appmgr.NewAppMgrClient(conn)
	var testing_ns_id string
	// valid token
	md := metadata.New(map[string]string{
		"authorization": ns_test_token,
	})
	ctx := metadata.NewOutgoingContext(context.Background(), md)
	tokenContext, cancel := context.WithTimeout(ctx, 120 * time.Second)

	defer cancel()

	// create namespace for ns_delete
	if rsp, err := appClient.CreateNamespace(tokenContext,
		&appmgr.CreateNamespaceRequest{Namespace: &common_proto.Namespace{
			NsName:         "ns_cancel_1",
			NsCpuLimit:     1000,
			NsMemLimit:     2000,
			NsStorageLimit: 50000,
		}}); err != nil && len(rsp.NsId) > 0 {
		t.Error(err)
	} else {
		t.Log("create ns for ns_delete successfully : nsid   \n  " + rsp.NsId)
		testing_ns_id = rsp.NsId
	}

	// wait for namespace status changed
	time.Sleep(15 * time.Second)

	// case 1: correct inputs
	if _, err := appClient.DeleteNamespace(tokenContext,
		&appmgr.DeleteNamespaceRequest{NsId: testing_ns_id}); err != nil {
		t.Error(err)
	} else {
		t.Log("case 1: correct inputs: delete ns successfully\n")
		time.Sleep(2 * time.Second)
	}
}
