package main

import (
	"log"

	grpc "github.com/micro/go-grpc"
)

var token = "token"

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	log.Println("client service start...")
	srv := grpc.NewService()

	srv.Init()
	/*
		cl := appmgr.NewAppMgrService(ankr_default.AppMgrRegistryServerName, srv.Client())

		tokenContext := metadata.NewContext(context.Background(), map[string]string{
			"Token": token,
		})

		apps := testCommon.MockApps()
		for i := range apps {
			if _, err := cl.CreateApp(tokenContext, &appmgr.CreateAppRequest{UserId: apps[i].UserId, App: &apps[i]}); err != nil {
				log.Fatal(err.Error())
			} else {
				log.Println("CreateApp Ok")
			}
		}

		userApps := []*common_proto.App{}
		if rsp, err := cl.AppList(tokenContext, &appmgr.ID{UserId: "1"}); err != nil {
			log.Fatal(err.Error())
		} else {
			userApps = append(userApps, rsp.Apps...)
			log.Println("AppList Ok")
		}

		if len(userApps) == 0 {
			log.Fatalf("no apps belongs to %d\n", 1)
		}

		// CancelApp
		cancelApp := userApps[0]
		if _, err := cl.CancelApp(tokenContext, &appmgr.Request{UserId: cancelApp.UserId, AppId: cancelApp.Id}); err != nil {
			log.Fatal(err.Error())
		} else {
			log.Println("CancelApp Ok")
		}

		// Verify Canceled app
		if _, err := cl.AppDetail(tokenContext, &appmgr.Request{UserId: cancelApp.UserId, AppId: cancelApp.Id}); err != nil {
			log.Fatal(err.Error())
		} else {
			log.Println("AppDetail Ok")
		}

		// UpdateApp
		cancelApp.Name = "updateApp"
		if _, err := cl.UpdateApp(tokenContext, &appmgr.UpdateAppRequest{UserId: cancelApp.UserId, App: cancelApp}); err != nil {
			log.Fatal(err.Error())
		} else {
			log.Println("AppDetail Ok")
		}

		/* Verify updated app
		if rsp, err := cl.AppDetail(tokenContext, &appmgr.Request{UserId: cancelApp.UserId, AppId: cancelApp.Id}); err != nil {
			log.Fatal(err.Error())
		} else {
			if !testCommon.IsEqual(rsp.App, cancelApp) || rsp.App.Status != common_proto.AppStatus_UPDATING {
				log.Fatal("UpdateApp operation does not take effect")
			}
			log.Println("UpdateApp takes effect")
		}*/
}
