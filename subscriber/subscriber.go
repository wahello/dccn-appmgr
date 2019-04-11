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

// UpdateAppByFeedback receives app result from data center, returns to v1
// UpdateAppStatusByFeedback updates database status by performing feedback from the data center of the app.
// sets executor's id, updates app status.
func (p *AppStatusFeedback) HandlerFeedbackEventFromDataCenter(ctx context.Context, stream *common_proto.DCStream) error {

	log.Printf(">>>>>>>>HandlerFeedbackEventFromDataCenter: Receive New Event: %+v with payload: %+v ", stream.GetOpType(), stream.GetOpPayload())
	var update bson.M
	var collection string
	var id string
	switch stream.OpPayload.(type) {
	case *common_proto.DCStream_AppDeployment:
		appDeployment := stream.GetAppReport().AppDeployment
		update = bson.M{"$set": bson.M{"status": appDeployment.Status, "namespace": appDeployment.Namespace}}
		collection = "app"
		id = appDeployment.Id
		if stream.OpType == common_proto.DCOperation_APP_CREATE {
			p.db.Update("namespace", appDeployment.Namespace.Id, bson.M{"$set": bson.M{"status": appDeployment.Namespace.Status, "clusterid": appDeployment.Namespace.ClusterId, "clustername": appDeployment.Namespace.ClusterName}})
		} // feedback  AppStatus_START_FAILED  AppStatus_START_SUCCESS => AppStatus_RUNNING

	case *common_proto.DCStream_Namespace:
		namespace := stream.GetNamespace()
		update = bson.M{"$set": bson.M{"status": namespace.Status}}
		if stream.OpType == common_proto.DCOperation_NS_CREATE {
			update = bson.M{"$set": bson.M{"status": namespace.Status, "clusterid": namespace.ClusterId, "clustername": namespace.ClusterName}}
		}
		collection = "namespace"
		id = namespace.Id
	}

	return p.db.Update(collection, id, update)
}
