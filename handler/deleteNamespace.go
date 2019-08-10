package handler

import (
	ankr_default "github.com/Ankr-network/dccn-common/protos"
	appmgr "github.com/Ankr-network/dccn-common/protos/appmgr/v1/grpc"
	common_proto "github.com/Ankr-network/dccn-common/protos/common"
	common_util "github.com/Ankr-network/dccn-common/util"
	"gopkg.in/mgo.v2/bson"
	"log"
	"errors"
	"context"
)

// DeleteNamespace will delete a namespace with no resource owned
func (p *AppMgrHandler) DeleteNamespace(ctx context.Context,
	req *appmgr.DeleteNamespaceRequest) (*common_proto.Empty, error) {

	userId := common_util.GetUserID(ctx)
	log.Printf(">>>>>>>>>Debug into DeleteNamespace %+v\nctx: %+v\n", req, ctx)

	namespaceRecord, err := p.db.GetNamespace(req.NsId)
	if err != nil {
		log.Println(err.Error())
		return &common_proto.Empty{}, err
	}

	if err := checkNsId(userId, namespaceRecord.UID); err != nil {
		log.Println(err.Error())
		return &common_proto.Empty{}, err
	}

	if namespaceRecord.Status == common_proto.NamespaceStatus_NS_CANCELED {
		return &common_proto.Empty{}, ankr_default.ErrCanceledTwice
	}

	clusterConnection, err := p.db.GetClusterConnection(namespaceRecord.ClusterID)
	if err != nil || clusterConnection.Status != common_proto.DCStatus_AVAILABLE {
		log.Println("cluster connection not available, namespace can not be deleted")
		return nil, errors.New("cluster connection not available, namespace can not be deleted")
	}

	apps, err := p.db.GetAllAppByNamespaceId(req.NsId)
	if err != nil {
		log.Println(err.Error())
		return &common_proto.Empty{}, err
	}
	for _, app := range apps {
		if app.Status != common_proto.AppStatus_APP_CANCELED &&
			app.Status != common_proto.AppStatus_APP_CANCELING &&
			app.Status != common_proto.AppStatus_APP_FAILED {
			return &common_proto.Empty{}, errors.New("namespace still got running app, can not delete.")
		}
	}

	namespaceReport := convertFromNamespaceRecord(namespaceRecord)

	event := common_proto.DCStream{
		OpType:    common_proto.DCOperation_NS_CANCEL,
		OpPayload: &common_proto.DCStream_Namespace{Namespace: namespaceReport.Namespace},
	}

	if err := p.deployApp.Publish(&event); err != nil {
		log.Println(err.Error())
		return &common_proto.Empty{}, errors.New(ankr_default.PublishError + err.Error())
	}

	if err := p.db.Update("namespace", req.NsId, bson.M{
		"$set": bson.M{"status": common_proto.NamespaceStatus_NS_CANCELING}}); err != nil {
		log.Println(err.Error())
		return &common_proto.Empty{}, err
	}

	return &common_proto.Empty{}, nil
}

