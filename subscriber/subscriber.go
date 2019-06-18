package subscriber

import (
	"log"
	"time"

	"gopkg.in/mgo.v2"

	db "github.com/Ankr-network/dccn-appmgr/db_service"
	common_proto "github.com/Ankr-network/dccn-common/protos/common"
	"github.com/golang/protobuf/ptypes/timestamp"
	"gopkg.in/mgo.v2/bson"
)

type AppStatusFeedback struct {
	db db.DBService
}

func New(db db.DBService) *AppStatusFeedback {
	return &AppStatusFeedback{db}
}

// UHandlerFeedbackEventFromDataCenter receives app report result from data center and update record
func (p *AppStatusFeedback) HandlerFeedbackEventFromDataCenter(stream *common_proto.DCStream) error {

	log.Printf(">>>>>>>>HandlerFeedbackEventFromDataCenter: Receive New Event: %+v with payload: %+v ", stream.GetOpType(), stream.GetOpPayload())
	update := bson.M{}
	var collection string
	var id string
	switch x := stream.OpPayload.(type) {

	case *common_proto.DCStream_AppReport:

		appReport := stream.GetAppReport()

		appRecord, err := p.db.GetApp(appReport.AppDeployment.AppId)
		if err != nil {
			log.Println(err.Error())
			return err
		}

		update["report"] = appReport.Report

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
				update["event"] = appReport.AppEvent
			}
		case common_proto.DCOperation_APP_UPDATE:
			if appRecord.Status == common_proto.AppStatus_APP_UPDATING {
				switch appReport.AppEvent {
				case common_proto.AppEvent_UPDATE_APP_SUCCEED:
					update["status"] = common_proto.AppStatus_APP_RUNNING
					update["chartdetail"] = appRecord.ChartUpdating
				case common_proto.AppEvent_UPDATE_APP_FAILED:
					update["status"] = common_proto.AppStatus_APP_UPDATE_FAILED
				}
				update["event"] = appReport.AppEvent
			}
		case common_proto.DCOperation_APP_CANCEL:
			update["status"] = common_proto.AppStatus_APP_CANCELED
			update["event"] = appReport.AppEvent
		case common_proto.DCOperation_APP_DETAIL:
			update["detail"] = appReport.Detail
		default:
			log.Printf("OpType has unexpected type %v", opType)
		}

		collection = "app"
		id = appReport.AppDeployment.AppId

	case *common_proto.DCStream_NsReport:

		nsReport := stream.GetNsReport()

		nsRecord, err := p.db.GetNamespace(nsReport.Namespace.NsId)
		if err != nil {
			log.Println(err.Error())
			return err
		}

		update["event"] = nsReport.NsEvent

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
					update["cpulimit"] = nsRecord.CpuLimitUpdating
					update["memlimit"] = nsRecord.MemLimitUpdating
					update["storagelimit"] = nsRecord.StorageLimitUpdating
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
		id = nsReport.Namespace.NsId
		update["lastmodifieddate"] = &timestamp.Timestamp{Seconds: time.Now().Unix()}

	case *common_proto.DCStream_DataCenter:
		if _, err := p.db.GetClusterConnection(stream.GetDataCenter().DcId); err == mgo.ErrNotFound {
			if err = p.db.CreateClusterConnection(stream.GetDataCenter().DcId, stream.GetDataCenter().DcStatus); err != nil {
				log.Printf("Create cluster connection failed %v", err)
				return err
			}
		}
		collection = "clusterconnection"
		id = stream.GetDataCenter().DcId
		update["status"] = stream.GetDataCenter().DcStatus

	default:
		log.Printf("OpPayload has unexpected type %T", x)
	}
	log.Printf(">>>>>>>>HandlerFeedbackEventFromDataCenter: Update Collection %s on ID %s Update: %s", collection, id, update)

	update["lastmodifieddate"] = &timestamp.Timestamp{Seconds: time.Now().Unix()}
	return p.db.Update(collection, id, bson.M{"$set": update})
}
