package handler

import (
	ankr_default "github.com/Ankr-network/dccn-common/protos"
	appmgr "github.com/Ankr-network/dccn-common/protos/appmgr/v1/grpc"
	common_proto "github.com/Ankr-network/dccn-common/protos/common"
	"log"
	"context"
)

func (p *AppMgrHandler) AppDetail(ctx context.Context, req *appmgr.AppID) (*appmgr.AppDetailResponse, error) {
	log.Printf(">>>>>>>>>Debug into AppDetail: %+v\nctx: %+v \n", req, ctx)

	rsp := &appmgr.AppDetailResponse{}
	appRecord, err := p.db.GetApp(req.AppId)
	if err != nil {
		log.Println(err.Error())
		return rsp, err
	}

	log.Printf("appMessage  %+v \n", appRecord)
	if err != nil {
		log.Println(err.Error())
		return rsp, err
	}

	if appRecord.Hidden == true {
		log.Printf("app id %s already purged \n", req.AppId)
		return rsp, ankr_default.ErrAlreadyPurged
	}

	appMessage := convertToAppMessage(appRecord, p.db)
	log.Printf("appMessage %+v \n", appMessage)
	rsp.AppReport = &appMessage

	event := common_proto.DCStream{
		OpType:    common_proto.DCOperation_APP_DETAIL,
		OpPayload: &common_proto.DCStream_AppDeployment{AppDeployment: appMessage.AppDeployment},
	}

	if err := p.deployApp.Publish(&event); err != nil {
		log.Println(err.Error())
		return rsp, ankr_default.ErrPublish
	}

	return rsp, nil
}
