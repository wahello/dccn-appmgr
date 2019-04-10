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

	appDeployment := stream.GetAppReport().AppDeployment
	log.Printf(">>>>>>>>HandlerFeedbackEventFromDataCenter: Receive New Event: %+v from dc : %s ", appDeployment, appDeployment.Namespace.ClusterName)
	var update bson.M
	switch stream.OpType {
	case common_proto.DCOperation_APP_CREATE: // feedback  AppStatus_START_FAILED  AppStatus_START_SUCCESS => AppStatus_RUNNING
		status := common_proto.AppStatus_APP_RUNNING
		if appDeployment.Status == common_proto.AppStatus_APP_START_FAILED {
			status = common_proto.AppStatus_APP_START_FAILED
		}
		update = bson.M{"$set": bson.M{"status": status, "namespace": appDeployment.Namespace}}

	case common_proto.DCOperation_APP_UPDATE:
		status := common_proto.AppStatus_APP_RUNNING
		update = bson.M{"$set": bson.M{"status": status}}
	case common_proto.DCOperation_APP_CANCEL:
		status := common_proto.AppStatus_APP_CANCELLED
		if appDeployment.Status == common_proto.AppStatus_APP_CANCEL_FAILED {
			status = common_proto.AppStatus_APP_CANCEL_FAILED
		}
		update = bson.M{"$set": bson.M{"status": status}}
	}

	return p.db.Update("app", appDeployment.Id, update)
}
