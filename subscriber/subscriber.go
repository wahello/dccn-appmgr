package subscriber

import (
	"context"
	"log"

	db "github.com/Ankr-network/dccn-appmgr/db_service"
	common_proto "github.com/Ankr-network/dccn-common/protos/common"
	"gopkg.in/mgo.v2/bson"
)

type AppStatusFeedback struct {
	db db.DBService
}

func New(db db.DBService) *AppStatusFeedback {
	return &AppStatusFeedback{db}
}

// UHandlerFeedbackEventFromDataCenter receives app report result from data center and update record
func (p *AppStatusFeedback) HandlerFeedbackEventFromDataCenter(ctx context.Context, stream *common_proto.DCStream) error {

	log.Printf(">>>>>>>>HandlerFeedbackEventFromDataCenter: Receive New Event: %+v with payload: %+v ", stream.GetOpType(), stream.GetOpPayload())
	var update bson.M
	var collection string
	var id string
	switch x := stream.OpPayload.(type) {

	case *common_proto.DCStream_AppReport:

		appReport := stream.GetAppReport()

		appRecord, err := p.db.GetApp(appReport.AppDeployment.Id)
		if err != nil {
			log.Println(err.Error())
			return err
		}

		update = bson.M{
			"report": appReport.Report,
			"event":  appReport.AppEvent,
		}

		opType := stream.GetOpType()
		switch opType {
		case common_proto.DCOperation_APP_CREATE:
			if appRecord.Status == common_proto.AppStatus_APP_DISPATCHING ||
				appRecord.Status == common_proto.AppStatus_APP_LAUNCHING {
				switch appReport.AppEvent {
				case common_proto.AppEvent_LAUNCH_APP_SUCCEED:
					update["status"] = common_proto.AppStatus_APP_RUNNING
				case common_proto.AppEvent_LAUNCH_APP_FAILED:
					update["status"] = common_proto.AppStatus_APP_FAILED
				case common_proto.AppEvent_DISPATCH_APP:
					update["status"] = common_proto.AppStatus_APP_LAUNCHING
				}
			}
		case common_proto.DCOperation_APP_UPDATE:
			if appRecord.Status == common_proto.AppStatus_APP_UPDATING {
				switch appReport.AppEvent {
				case common_proto.AppEvent_UPDATE_APP_SUCCEED:
					update["status"] = common_proto.AppStatus_APP_RUNNING
				case common_proto.AppEvent_UPDATE_APP_FAILED:
					update["status"] = common_proto.AppStatus_APP_UPDATE_FAILED
				}
			}
		case common_proto.DCOperation_APP_CANCEL:
			update["status"] = common_proto.AppStatus_APP_CANCELED
		case common_proto.DCOperation_APP_DETAIL:
			update["detail"] = appReport.Detail
		default:
			log.Printf("OpType has unexpected type %v", opType)
		}

		collection = "app"
		id = appReport.AppDeployment.Id

	case *common_proto.DCStream_NsReport:

		nsReport := stream.GetNsReport()

		nsRecord, err := p.db.GetNamespace(nsReport.Namespace.Id)
		if err != nil {
			log.Println(err.Error())
			return err
		}

		update = bson.M{
			"event": nsReport.NsEvent,
		}

		opType := stream.GetOpType()
		switch opType {
		case common_proto.DCOperation_NS_CREATE:
			if nsRecord.Status == common_proto.NamespaceStatus_NS_DISPATCHING ||
				nsRecord.Status == common_proto.NamespaceStatus_NS_LAUNCHING {
				update["clusterid"] = nsReport.Namespace.ClusterId
				update["clustername"] = nsReport.Namespace.ClusterName
				switch nsReport.NsEvent {
				case common_proto.NamespaceEvent_LAUNCH_NS_SUCCEED:
					update["status"] = common_proto.NamespaceStatus_NS_RUNNING
				case common_proto.NamespaceEvent_LAUNCH_NS_FAILED:
					update["status"] = common_proto.NamespaceStatus_NS_FAILED
				case common_proto.NamespaceEvent_DISPATCH_NS:
					update["status"] = common_proto.NamespaceStatus_NS_LAUNCHING
				}
			}
		case common_proto.DCOperation_NS_UPDATE:
			if nsRecord.Status == common_proto.NamespaceStatus_NS_UPDATING {
				switch nsReport.NsEvent {
				case common_proto.NamespaceEvent_UPDATE_NS_SUCCEED:
					update["status"] = common_proto.NamespaceStatus_NS_RUNNING
				case common_proto.NamespaceEvent_UPDATE_NS_FAILED:
					update["status"] = common_proto.NamespaceStatus_NS_UPDATE_FAILED
				}
			}
		case common_proto.DCOperation_NS_CANCEL:
			update["status"] = common_proto.NamespaceStatus_NS_CANCELED
		default:
			log.Printf("OpType has unexpected type %v", opType)
		}

		collection = "namespace"
		id = nsReport.Namespace.Id

	default:
		log.Printf("OpPayload has unexpected type %T", x)
	}
	log.Printf(">>>>>>>>HandlerFeedbackEventFromDataCenter: Update Collection %s on ID %s Update: %s", collection, id, update)
	return p.db.Update(collection, id, bson.M{"$set": update})
}
