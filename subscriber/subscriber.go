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
	switch stream.OpPayload.(type) {

	case *common_proto.DCStream_AppReport:
		appReport := stream.GetAppReport()
		update = bson.M{"$set": bson.M{
			"status": appReport.AppStatus,
			"event":  appReport.AppEvent,
			"detail": appReport.Detail,
			"report": appReport.Report}}
		if stream.OpType == common_proto.DCOperation_APP_CREATE {
			update["namespace"] = bson.M{"cluserid": appReport.AppDeployment.Namespace.ClusterId,
				"clustername": appReport.AppDeployment.Namespace.ClusterName}
		}
		collection = "app"
		id = appReport.AppDeployment.Id

	case *common_proto.DCStream_NsReport:
		nsReport := stream.GetNsReport()
		update = bson.M{"$set": bson.M{
			"status": nsReport.NsStatus,
			"event":  nsReport.NsEvent}}
		if stream.OpType == common_proto.DCOperation_NS_CREATE {
			update["clusterid"] = nsReport.Namespace.ClusterId
			update["clustername"] = nsReport.Namespace.ClusterName
		}
		collection = "namespace"
		id = nsReport.Namespace.Id
	}

	return p.db.Update(collection, id, update)
}
