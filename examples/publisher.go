package main

import (
	_ "github.com/micro/go-plugins/broker/rabbitmq"
)

/*
// send events using the publisher
func sendEv(appId string, p micro.Publisher) {

	// create new event
	ev := common_proto.Event{
		EventType: common_proto.Operation_TASK_CANCEL,
		OpMessage: &common_proto.Event_AppFeedback{AppFeedback: &common_proto.AppFeedback{
			AppId: appId,
			Status: common_proto.AppStatus_CANCEL_FAILED,
		}},
	}

	log.Printf("publishing %+v\n", ev)

	// publish an event
	if err := p.Publish(context.Background(), &ev); err != nil {
		log.Fatalf("error publishing %v\n", err)
	}
}

func main() {

	log.SetFlags(log.LstdFlags | log.Lshortfile)
	// create a service
	service := grpc.NewService()

	// parse command line
	service.Init()

	// create publisher
	pub := micro.NewPublisher(ankr_default.MQFeedbackApp, service.Client())

	cl := appmgr.NewAppMgrService(ankr_default.AppMgrRegistryServerName, service.Client())
	/*app := testCommon.MockApps()[0]
	if _, err := cl.CreateApp(context.TODO(), &appmgr.CreateAppRequest{UserId: app.UserId, App: &app}); err != nil {
		log.Fatal(err.Error())
	} else {
		log.Println("CreateApp Ok")
	}

	var userApps []*common_proto.App
	if rsp, err := cl.AppList(context.TODO(), &appmgr.ID{UserId: "1"}); err != nil {
		log.Fatal(err.Error())
	} else {
		userApps = append(userApps, rsp.Apps...)
		log.Println("AppList Ok")
	}

	if len(userApps) == 0 {
		log.Fatalf("no apps belongs to %d\n", 1)
	}

	pubApp := userApps[0]

	// pub to topic 1
	sendEv(pubApp.Id, pub)

	// waits pub message arrive to mq
	time.Sleep(2 * time.Second)

	// Verify publish event
	if rsp, err := cl.AppDetail(context.TODO(), &appmgr.Request{UserId: pubApp.UserId, AppId: pubApp.Id}); err != nil {
		log.Fatal(err.Error())
	} else {
		if rsp.App.Status != common_proto.AppStatus_CANCEL_FAILED {
			log.Fatal("UpdateAppByFeedback do not app effect")
		} else {
			log.Println("AppDetail Ok")
		}
	}

	log.Println("Pub End")
}
*/
