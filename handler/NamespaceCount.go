package handler

import (
	appmgr "github.com/Ankr-network/dccn-common/protos/appmgr/v1/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	//common_proto "github.com/Ankr-network/dccn-common/protos/common"
	"context"
	"log"
)

func (p *AppMgrHandler) NamespaceCount(ctx context.Context,
	req *appmgr.NamespaceCountRequest) (*appmgr.NamespaceCountResponse, error) {
	log.Printf(">>>>>>>>>Debug into NamespaceCount %+v\nctx: %+v \n", req, ctx)
	rsp := &appmgr.NamespaceCountResponse{}

	nss, err := p.db.GetRunningNamespacesByClusterId(req.ClusterId)
	if err != nil {
		return rsp, status.Error(codes.Unknown, err.Error())
	}

	rsp.Namespace = uint64(len(nss))

	apps, err := p.db.GetRunningAppsByClusterID(req.ClusterId)

	rsp.App = uint64(len(apps))

	return rsp, nil
}
