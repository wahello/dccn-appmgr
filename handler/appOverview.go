package handler

import (
	appmgr "github.com/Ankr-network/dccn-common/protos/appmgr/v1/grpc"
	common_proto "github.com/Ankr-network/dccn-common/protos/common"
	common_util "github.com/Ankr-network/dccn-common/util"
	"context"
	"log"
)

func (p *AppMgrHandler) AppOverview(ctx context.Context, req *common_proto.Empty) (*appmgr.AppOverviewResponse, error) {
	log.Printf(">>>>>>>>>Debug into AppOverview, ctx: %+v\n", ctx)
	rsp := &appmgr.AppOverviewResponse{}
	userId := common_util.GetUserID(ctx)

	apps, err := p.db.GetAllApp(userId)
	if err != nil {
		log.Println(err.Error())
		return rsp, err
	}

	nss, err := p.db.GetAllNamespace(userId)
	if err != nil {
		log.Println(err.Error())
		return rsp, err
	}

	clusters := map[string]bool{}
	rsp.CpuTotal = 0
	rsp.MemTotal = 0
	rsp.StorageTotal = 0

	for i := 0; i < len(nss); i++ {
		clusters[nss[i].ClusterID] = true
		rsp.CpuTotal += float32(nss[i].CpuLimit)
		rsp.MemTotal += float32(nss[i].MemLimit)
		rsp.StorageTotal += float32(nss[i].StorageLimit)
	}

	rsp.CpuUsage = rsp.CpuTotal / 4
	rsp.MemUsage = rsp.MemTotal / 3
	rsp.StorageUsage = rsp.StorageTotal / 2
	rsp.ClusterCount = uint32(len(clusters))
	rsp.NamespaceCount = uint32(len(nss))
	rsp.NetworkCount = uint32(len(clusters))
	rsp.TotalAppCount = uint32(len(apps))

	return rsp, nil
}
