package handler

import (
	ankr_default "github.com/Ankr-network/dccn-common/protos"
	appmgr "github.com/Ankr-network/dccn-common/protos/appmgr/v1/grpc"
	common_proto "github.com/Ankr-network/dccn-common/protos/common"
	"gopkg.in/mgo.v2/bson"
	"log"
	"context"
)

func (p *AppMgrHandler) PurgeApp(ctx context.Context, req *appmgr.AppID) (*common_proto.Empty, error) {
	log.Printf(">>>>>>>>>Debug into PurgeApp, ctx: %+v\n", ctx)
	appRecord, err := p.db.GetApp(req.AppId)

	if err != nil {
		log.Printf(" PurgeApp for app id %s not found \n", req.AppId)
		return &common_proto.Empty{}, ankr_default.ErrAppNotExist
	}

	if appRecord.Hidden {
		log.Printf(" app id %s already purged \n", req.AppId)
		return &common_proto.Empty{}, ankr_default.ErrAlreadyPurged
	}

	if appRecord.Status != common_proto.AppStatus_APP_CANCELED {

		if _, err := p.CancelApp(ctx, req); err != nil {
			log.Printf(err.Error())
			return &common_proto.Empty{}, err
		}
	}

	log.Printf(" PurgeApp  %s \n", req.AppId)
	if err := p.db.Update("app", req.AppId, bson.M{"$set": bson.M{"hidden": true}}); err != nil {
		log.Printf(err.Error())
		return &common_proto.Empty{}, err
	}

	return &common_proto.Empty{}, nil
}
