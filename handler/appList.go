package handler

import (
	appmgr "github.com/Ankr-network/dccn-common/protos/appmgr/v1/grpc"
	common_proto "github.com/Ankr-network/dccn-common/protos/common"
	common_util "github.com/Ankr-network/dccn-common/util"
	"github.com/golang/protobuf/ptypes/timestamp"
	"gopkg.in/mgo.v2/bson"
	"log"
	"time"
	"context"
)

func (p *AppMgrHandler) AppList(ctx context.Context, req *common_proto.Empty) (*appmgr.AppListResponse, error) {
	userId := common_util.GetUserID(ctx)
	log.Printf(">>>>>>>>>Debug into AppList, ctx: %+v \n", ctx)

	apps, err := p.db.GetAllApp(userId)
	rsp := &appmgr.AppListResponse{}
	log.Printf("appMessage  %+v \n", apps)
	if err != nil {
		log.Println(err.Error())
		return rsp, err
	}

	appsWithoutHidden := make([]*common_proto.AppReport, 0)

	for i := 0; i < len(apps); i++ {
		if !apps[i].Hidden && (apps[i].Status == common_proto.AppStatus_APP_CANCELED ||
			apps[i].Status == common_proto.AppStatus_APP_CANCELING) && (apps[i].LastModifiedDate == nil ||
			apps[i].LastModifiedDate.Seconds < (time.Now().Unix()-7200)) {
			log.Printf("Canceled App time exceeded 7200 seconds, Now: %+v \n", time.Now().Unix())
			apps[i].Hidden = true
			p.db.Update("app", apps[i].ID, bson.M{"$set": bson.M{"hidden": true,
				"lastmodifieddate": &timestamp.Timestamp{Seconds: time.Now().Unix()}}})
		}
		if !apps[i].Hidden {
			appMessage := convertToAppMessage(apps[i], p.db)
			log.Printf("appMessage  %+v \n", appMessage)
			if len(appMessage.AppDeployment.Namespace.ClusterId) > 0 {
				clusterConnection, err := p.db.GetClusterConnection(appMessage.AppDeployment.Namespace.ClusterId)
				if err != nil || clusterConnection.Status == common_proto.DCStatus_UNAVAILABLE {
					appMessage.AppStatus = common_proto.AppStatus_APP_UNAVAILABLE
					appMessage.AppEvent = common_proto.AppEvent_APP_HEARTBEAT_FAILED
				}
			}
			appsWithoutHidden = append(appsWithoutHidden, &appMessage)
			if appMessage.AppStatus == common_proto.AppStatus_APP_RUNNING {
				event := common_proto.DCStream{
					OpType:    common_proto.DCOperation_APP_DETAIL,
					OpPayload: &common_proto.DCStream_AppDeployment{AppDeployment: appMessage.AppDeployment},
				}
				if err := p.deployApp.Publish(&event); err != nil {
					log.Println(err.Error())
				}
			}
		}
	}

	rsp.AppReports = appsWithoutHidden

	return rsp, nil
}
