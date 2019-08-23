package handler

import (
	appmgr "github.com/Ankr-network/dccn-common/protos/appmgr/v1/grpc"
	//common_proto "github.com/Ankr-network/dccn-common/protos/common"
	"log"
	"context"
)

func (p *AppMgrHandler) NamespaceCount(ctx context.Context,
	req *appmgr.NamespaceCountRequest) (*appmgr.NamespaceCountResponse, error) {
	log.Printf(">>>>>>>>>Debug into NamespaceCount %+v\nctx: %+v \n", req, ctx)
	rsp := &appmgr.NamespaceCountResponse{}
	appRecord, err := p.db.GetAllNamespaceByClusterId(req.ClusterId)
	if err != nil {
		log.Println(err.Error())
		return rsp, err
	}

	rsp.Namespace = uint64(len(appRecord))
	var temp uint64
	temp = 0
	for _, v := range(appRecord) {
		Record, err := p.db.GetAllAppByNamespaceId(v.ID)
		if err != nil {
			log.Println(err.Error())
			return rsp, err
		}
		temp = temp + uint64(len(Record))
	}
	rsp.App = temp

	return rsp, nil
}