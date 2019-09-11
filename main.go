package main

import (
	"log"

	micro2 "github.com/Ankr-network/dccn-common/ankr-micro"

	ankr_default "github.com/Ankr-network/dccn-common/protos"
	appmgr "github.com/Ankr-network/dccn-common/protos/appmgr/v1/grpc"

	"github.com/Ankr-network/dccn-appmgr/config"
	dbservice "github.com/Ankr-network/dccn-appmgr/db_service"
	"github.com/Ankr-network/dccn-appmgr/handler"
	"github.com/Ankr-network/dccn-appmgr/subscriber"

	"github.com/Ankr-network/dccn-common/broker/rabbitmq"
)

var (
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
	srv := micro2.NewService()

	// Register Function as AppStatusFeedback to update app by data center manager's feedback.
	if err := micro2.RegisterSubscriber(ankr_default.MQFeedbackApp, subscriber.New(db)); err != nil {
		log.Fatal(err)
	}

	broker := rabbitmq.NewBroker(conf.RabbitMQUrl)
	metricsSubscriber := subscriber.MetricsSubscriber{DB: db}
	if err := broker.Subscribe("appmgr.metrics", "ankr.topic.metrics", metricsSubscriber.Handle); err != nil {
		log.Fatal(err)
	}

	// New Publisher to deploy new app action.
	deployAppPublisher := micro2.NewPublisher(ankr_default.MQDeployApp)
	// Register Handler
	deployAppHandler := handler.New(db, deployAppPublisher)
	appmgr.RegisterAppMgrServer(srv.GetServer(), deployAppHandler)

	// Run srv
	srv.Start()
}
