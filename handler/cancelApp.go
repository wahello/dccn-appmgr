package handler

import (
	"context"
	ankr_default "github.com/Ankr-network/dccn-common/protos"
	appmgr "github.com/Ankr-network/dccn-common/protos/appmgr/v1/grpc"
	common_proto "github.com/Ankr-network/dccn-common/protos/common"
	common_util "github.com/Ankr-network/dccn-common/util"
	"github.com/golang/protobuf/ptypes/timestamp"
	"gopkg.in/mgo.v2/bson"
	"log"
	"time"
)

func (p *AppMgrHandler) CancelApp(ctx context.Context, req *appmgr.AppID) (*common_proto.Empty, error) {
	_, teamId := common_util.GetUserIDAndTeamID(ctx)
	log.Printf(">>>>>>>>>Debug into CancelApp: %+v\nctx: %+v \n", req, ctx)

	if err := checkId(teamId, req.AppId); err != nil {
		log.Println(err.Error())
		return &common_proto.Empty{}, err
	}

	app, err := p.checkOwner(teamId, req.AppId)
	if err != nil {
		log.Println(err.Error())
		return &common_proto.Empty{}, err
	}

	if app.AppStatus == common_proto.AppStatus_APP_UNAVAILABLE || app.AppStatus == common_proto.AppStatus_APP_FAILED {
		log.Printf("app %s is unavailable or failed, cacel directly", req.AppId)
		if err := p.db.Update("app", req.AppId, bson.M{"$set": bson.M{
			"status":           common_proto.AppStatus_APP_CANCELED,
			"hidden":           true,
			"lastmodifieddate": &timestamp.Timestamp{Seconds: time.Now().Unix()},
		}}); err != nil {
			log.Printf("Update app %s to canceled status error: %v", req.AppId, err)
			return &common_proto.Empty{}, err
		}
		return &common_proto.Empty{}, nil
	}

	if app.AppStatus == common_proto.AppStatus_APP_CANCELED {
		return &common_proto.Empty{}, ankr_default.ErrCanceledTwice
	}

	/*
		clusterConnection, err := p.db.GetClusterConnection(app.AppDeployment.Namespace.ClusterId)
		if err != nil || clusterConnection.Status != common_proto.DCStatus_AVAILABLE {
			log.Println("cluster connection not available, app can not be canceled")
			return nil, errors.New("cluster connection not available, app can not be canceled")
		}
	*/

	event := common_proto.DCStream{
		OpType:    common_proto.DCOperation_APP_CANCEL,
		OpPayload: &common_proto.DCStream_AppDeployment{AppDeployment: app.AppDeployment},
	}

	if err := p.deployApp.Publish(&event); err != nil {
		log.Println(err.Error())
		return &common_proto.Empty{}, ankr_default.ErrPublish
	}

	if err := p.db.Update("app", req.AppId, bson.M{"$set": bson.M{
		"status":           common_proto.AppStatus_APP_CANCELING,
		"lastmodifieddate": &timestamp.Timestamp{Seconds: time.Now().Unix()},
	}}); err != nil {
		log.Println(err.Error())
		return &common_proto.Empty{}, err
	}

	return &common_proto.Empty{}, nil
}
