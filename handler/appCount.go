package handler

import (
	"context"
	appmgr "github.com/Ankr-network/dccn-common/protos/appmgr/v1/grpc"
	"log"
)

func (p *AppMgrHandler) AppCount(ctx context.Context,
	req *appmgr.AppCountRequest) (*appmgr.AppCountResponse, error) {
	log.Printf(">>>>>>>>>Debug into AppCount %+v\nctx: %+v \n", req, ctx)
	rsp := &appmgr.AppCountResponse{}
	if len(req.TeamId) == 0 || len(req.ClusterId) == 0 {
		rsp.AppCount = 0
	}

	apps, err := p.db.GetRunningAppsByTeamIDAndClusterID(req.TeamId, req.ClusterId)
	if err != nil {
		log.Println(err.Error())
		return rsp, err
	}

	rsp.AppCount = uint32(len(apps))

	return rsp, nil
}
