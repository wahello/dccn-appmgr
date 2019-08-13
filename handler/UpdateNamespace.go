package handler

import (
	ankr_default "github.com/Ankr-network/dccn-common/protos"
	appmgr "github.com/Ankr-network/dccn-common/protos/appmgr/v1/grpc"
	common_proto "github.com/Ankr-network/dccn-common/protos/common"
	common_util "github.com/Ankr-network/dccn-common/util"
	"google.golang.org/grpc/status"
	"context"
	"log"
	"errors"
)

// UpdateNamespace will update a namespace with cpu/mem/storage limit
func (p *AppMgrHandler) UpdateNamespace(ctx context.Context,
	req *appmgr.UpdateNamespaceRequest) (*common_proto.Empty, error) {

	log.Printf(">>>>>>>>>Debug into UpdateNamespace: %+v\nctx: %+v\n", req, ctx)
	_, teamId := common_util.GetUserIDAndTeamID(ctx)

	if req.Namespace == nil || (req.Namespace.NsCpuLimit == 0 ||
		req.Namespace.NsMemLimit == 0 || req.Namespace.NsStorageLimit == 0) {
		log.Printf("invalid input: empty namespace properties not accepted \n")
		return &common_proto.Empty{}, ankr_default.ErrNsEmpty
	}

	namespaceRecord, err := p.db.GetNamespace(req.Namespace.NsId)
	if err != nil {
		log.Println(err.Error())
		return &common_proto.Empty{}, err
	}
	if err := checkNsId(teamId, namespaceRecord.TeamID); err != nil {
		log.Println(err.Error())
		return &common_proto.Empty{}, err
	}

	if namespaceRecord.Status != common_proto.NamespaceStatus_NS_RUNNING &&
		namespaceRecord.Status != common_proto.NamespaceStatus_NS_UPDATE_FAILED {
		log.Println("namespace status is not running, cannot update")
		return &common_proto.Empty{}, ankr_default.ErrNSStatusCanNotUpdate
	}

	clusterConnection, err := p.db.GetClusterConnection(namespaceRecord.ClusterID)
	if err != nil || clusterConnection.Status != common_proto.DCStatus_AVAILABLE {
		log.Println("cluster connection not available, namespace can not be updated")
		return &common_proto.Empty{}, errors.New("cluster connection not available, namespace can not be updated")
	}

	namespaceReport := convertFromNamespaceRecord(namespaceRecord)

	if req.Namespace.NsCpuLimit > 0 && req.Namespace.NsMemLimit > 0 && req.Namespace.NsStorageLimit > 0 {
		namespaceReport.Namespace.NsCpuLimit = req.Namespace.NsCpuLimit
		namespaceReport.Namespace.NsMemLimit = req.Namespace.NsMemLimit
		namespaceReport.Namespace.NsStorageLimit = req.Namespace.NsStorageLimit
	}

	event := common_proto.DCStream{
		OpType:    common_proto.DCOperation_NS_UPDATE,
		OpPayload: &common_proto.DCStream_Namespace{Namespace: namespaceReport.Namespace},
	}

	if err := p.deployApp.Publish(&event); err != nil {
		log.Println(err.Error())
		return &common_proto.Empty{}, errors.New(ankr_default.PublishError + err.Error())
	}
	// TODO: wait deamon notify
	if err := p.db.UpdateNamespace(req.Namespace); err != nil {
		log.Printf("Err Status: %s, Err Message: %s", status.Code(err), err.Error())
		return &common_proto.Empty{}, err
	}
	return &common_proto.Empty{}, nil
}

