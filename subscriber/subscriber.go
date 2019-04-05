package subscriber

import (
	"context"
	"log"

	common_proto "github.com/Ankr-network/dccn-common/protos/common"
	db "github.com/Ankr-network/dccn-appmgr/db_service"
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

	app := stream.GetAppReport().App
	log.Printf(">>>>>>>>HandlerFeedbackEventFromDataCenter: Receive New Event: %+v from dc : %s ", app, app.DataCenterName)
	var update bson.M
	switch stream.OpType {
	case common_proto.DCOperation_TASK_CREATE: // feedback  AppStatus_START_FAILED  AppStatus_START_SUCCESS => AppStatus_RUNNING
		status := common_proto.AppStatus_RUNNING
		if app.Status == common_proto.AppStatus_START_FAILED {
			status = common_proto.AppStatus_START_FAILED
		}
		update = bson.M{"$set": bson.M{"status": status, "datacenter": app.DataCenterName}}

	case common_proto.DCOperation_TASK_UPDATE:
		status := common_proto.AppStatus_RUNNING
		update = bson.M{"$set": bson.M{"status": status}}
	case common_proto.DCOperation_TASK_CANCEL:
		status := common_proto.AppStatus_CANCELLED
		if app.Status == common_proto.AppStatus_CANCEL_FAILED {
			status = common_proto.AppStatus_CANCEL_FAILED
		}
		update = bson.M{"$set": bson.M{"status": status}}
	}

	return p.db.Update(app.Id, update)
}
