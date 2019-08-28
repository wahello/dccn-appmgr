package handler

import (
	"context"
	appmgr "github.com/Ankr-network/dccn-common/protos/appmgr/v1/grpc"
	common_proto "github.com/Ankr-network/dccn-common/protos/common"
	common_util "github.com/Ankr-network/dccn-common/util"
	"log"
)

func (p *AppMgrHandler) AppOverview(ctx context.Context, req *common_proto.Empty) (*appmgr.AppOverviewResponse, error) {
	log.Printf(">>>>>>>>>Debug into AppOverview, ctx: %+v\n", ctx)
	rsp := &appmgr.AppOverviewResponse{}
	_, teamId := common_util.GetUserIDAndTeamID(ctx)

	nss, err := p.db.GetRunningNamespaces(teamId)
	if err != nil {
		log.Println(err.Error())
		return rsp, err
	}

	rsp.NamespaceCount = uint32(len(nss))

	for _, ns := range nss {
		log.Printf("caculating ns %+v", ns)
		rsp.CpuTotal += float32(ns.CpuLimit)
		rsp.CpuUsage += float32(ns.CpuUsage)
		rsp.MemTotal += float32(ns.MemLimit)
		rsp.MemUsage += float32(ns.MemUsage)
		rsp.StorageTotal += float32(ns.StorageLimit)
		rsp.StorageUsage += float32(ns.StorageUsage)
	}

	if clusterConnections, err := p.db.GetAvailableClusterConnections(); err != nil {
		log.Printf("GetAvailableClusterConnections error: %v", err)
	} else {
		rsp.ClusterCount = uint32(len(clusterConnections))
		for _, connection := range clusterConnections {
			metrics := connection.Metrics
			if metrics != nil {
				rsp.NetworkCount += uint32(metrics.EndPointCount)
			}
		}
	}

	charts, err := getCharts("team", "stable")
	if err != nil {
		return rsp, err
	}

	rsp.TotalAppCount = uint32(len(charts))

	return rsp, nil
}
