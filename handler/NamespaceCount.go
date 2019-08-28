package handler

import (
	appmgr "github.com/Ankr-network/dccn-common/protos/appmgr/v1/grpc"
	//common_proto "github.com/Ankr-network/dccn-common/protos/common"
	"context"
	"log"
)

func (p *AppMgrHandler) NamespaceCount(ctx context.Context,
	req *appmgr.NamespaceCountRequest) (*appmgr.NamespaceCountResponse, error) {
	log.Printf(">>>>>>>>>Debug into NamespaceCount %+v\nctx: %+v \n", req, ctx)
	rsp := &appmgr.NamespaceCountResponse{}

	if len(req.ClusterId) > 0 {
		nsCount, err := p.db.CountRunningNamespacesByClusterID(req.ClusterId)
		if err != nil {
			log.Printf("CountRunningNamespacesByClusterID error %v", err)
			return rsp, err
		}

		rsp.Namespace = uint64(nsCount)

		appCount, err := p.db.CountRunningAppsByClusterID(req.ClusterId)
		if err != nil {
			log.Printf("CountRunningAppsByClusterID error %v", err)
			return rsp, err
		}

		rsp.App = uint64(appCount)
	} else {
		nsCount, err := p.db.CountRunningNamespaces()
		if err != nil {
			log.Printf("CountRunningNamespaces error %v", err)
			return rsp, err
		}

		rsp.Namespace = uint64(nsCount)

		appCount, err := p.db.CountRunningApps()
		if err != nil {
			log.Printf("CountRunningApps error %v", err)
			return rsp, err
		}

		rsp.App = uint64(appCount)
	}

	return rsp, nil
}
