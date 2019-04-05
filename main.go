package main

import (
	"log"

	grpc "github.com/micro/go-grpc"
	micro "github.com/micro/go-micro"

	ankr_default "github.com/Ankr-network/dccn-common/protos"
	pb "github.com/Ankr-network/dccn-common/protos/appmgr/v1/micro"

	"github.com/Ankr-network/dccn-appmgr/config"
	dbservice "github.com/Ankr-network/dccn-appmgr/db_service"
	"github.com/Ankr-network/dccn-appmgr/handler"
	"github.com/Ankr-network/dccn-appmgr/subscriber"

	_ "github.com/micro/go-plugins/broker/rabbitmq"
)

var (
	srv  micro.Service
	conf config.Config
	db   dbservice.DBService
	err  error
)

func main() {
	Init()

	if db, err = dbservice.New(conf.DB); err != nil {
		log.Fatal(err.Error())
	}
	defer db.Close()

	startHandler(db)
}

// Init starts handler to listen.
func Init() {
	log.SetFlags(log.Lshortfile | log.LstdFlags)

	if conf, err = config.Load(); err != nil {
		log.Fatal(err.Error())
	}
	log.Printf("load config: %#v\n", conf)
}

// StartHandler starts handler to listen.
func startHandler(db dbservice.DBService) {
	// var srv micro.Service
	// New Service
	srv = grpc.NewService(
		micro.Name(ankr_default.AppMgrRegistryServerName),
	)

	// Initialise srv
	srv.Init()

	// New Publisher to deploy new app action.
	deployApp := micro.NewPublisher(ankr_default.MQDeployApp, srv.Client())

	// Register Function as AppStatusFeedback to update app by data center manager's feedback.
	opt := srv.Server().Options()
	opt.Broker.Connect()
	if err := micro.RegisterSubscriber(ankr_default.MQFeedbackApp, srv.Server(), subscriber.New(db)); err != nil {
		log.Fatal(err.Error())
	}

	// Register Handler
	if err := pb.RegisterAppMgrHandler(srv.Server(), handler.New(db, deployApp)); err != nil {
		log.Fatal(err.Error())
	}

	// Run srv
	if err := srv.Run(); err != nil {
		log.Println(err.Error())
	}
}
