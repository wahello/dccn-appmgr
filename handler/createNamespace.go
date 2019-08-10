package handler

import (
	ankr_default "github.com/Ankr-network/dccn-common/protos"
	appmgr "github.com/Ankr-network/dccn-common/protos/appmgr/v1/grpc"
	common_proto "github.com/Ankr-network/dccn-common/protos/common"
	common_util "github.com/Ankr-network/dccn-common/util"
	"github.com/google/uuid"
	"context"
	"log"
	"errors"
)

// CreateNamespace will create a namespace on cluster which is desinated by dcmgr
func (p *AppMgrHandler) CreateNamespace(ctx context.Context,
	req *appmgr.CreateNamespaceRequest) (*appmgr.CreateNamespaceResponse, error) {

	rsp := &appmgr.CreateNamespaceResponse{}
	uid := common_util.GetUserID(ctx)
	if len(uid) == 0 {
		return rsp, errors.New("user id not found in context")
	}
	log.Printf(">>>>>>>>>Debug into CreateNamespace: %+v\nctx: %+v\n", req, ctx)

	if req.Namespace == nil || req.Namespace.NsCpuLimit == 0 ||
		req.Namespace.NsMemLimit == 0 || req.Namespace.NsStorageLimit == 0 {
		log.Printf("invalid input: empty namespace properties not accepted \n")
		return rsp, ankr_default.ErrNsEmpty
	}

	if len(req.Namespace.ClusterId) > 0 {
		clusterConnection, err := p.db.GetClusterConnection(req.Namespace.ClusterId)
		if err != nil || clusterConnection.Status != common_proto.DCStatus_AVAILABLE {
			log.Println("cluster connection not available, namespace can not be created")
			return rsp, errors.New("cluster connection not available, namespace can not be created")
		}
	}

	req.Namespace.NsId = "ns-" + uuid.New().String()

	event := common_proto.DCStream{
		OpType:    common_proto.DCOperation_NS_CREATE,
		OpPayload: &common_proto.DCStream_Namespace{Namespace: req.Namespace},
	}

	if err := p.deployApp.Publish(&event); err != nil {
		log.Println(ankr_default.ErrPublish)
		return rsp, ankr_default.ErrPublish
	} else {
		log.Println("app manager service send CreateNamespace MQ message to dc manager service (api)")
	}

	if err := p.db.CreateNamespace(req.Namespace, uid); err != nil {
		log.Println(err.Error())
		return rsp, err
	}

	rsp.NsId = req.Namespace.NsId

	return rsp, nil
}