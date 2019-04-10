package handler

import (
	"context"
	"log"
	"strings"

	db "github.com/Ankr-network/dccn-appmgr/db_service"
	"github.com/Ankr-network/dccn-common/protos"
	"github.com/Ankr-network/dccn-common/protos/appmgr/v1/micro"
	"github.com/Ankr-network/dccn-common/protos/common"
	"github.com/google/uuid"
	"gopkg.in/mgo.v2/bson"
)

func (p *AppMgrHandler) CreateNamespace(ctx context.Context, req *appmgr.CreateNamespaceRequest, rsp *common_proto.Empty) error {
	uid := getUserID(ctx)
	log.Println("app manager service CreateApp")

	log.Printf("CreateNamespace Namespace %+v", req)

	req.Namespace.Id = uuid.New().String()

	event := common_proto.DCStream{
		OpType:    common_proto.DCOperation_NS_CREATE,
		OpPayload: &common_proto.DCStream_Namespace{Namespace: req.Namespace},
	}

	if err := p.deployApp.Publish(context.Background(), &event); err != nil {
		log.Println(ankr_default.ErrPublish)
		return ankr_default.ErrPublish
	} else {
		log.Println("app manager service send CreateApp MQ message to dc manager service (api)")
	}

	if err := p.db.CreateNamespace(req.Namespace, uid); err != nil {
		log.Println(err.Error())
		return err
	}

	return nil
}

type NamespaceRecord struct {
	NamespaceID     string // short hash of uid+name+cluster_id
	Name            string
	NamespaceUserID string
	Cluster_ID      string //id of cluster
	Cluster_Name    string //name of cluster
	Creation_date   uint64
	Cpu_limit       float64
	Mem_limit       uint64
	Storage_limit   uint64
}

func convertFromNamespaceRecord(namespace db.NamespaceRecord) common_proto.Namespace {
	message := common_proto.Namespace{}
	message.Id = namespace.NamespaceID
	message.Name = namespace.Name
	message.ClusterId = namespace.Cluster_ID
	message.ClusterName = namespace.Cluster_Name
	message.NamespaceStatus = namespace.Status
	message.CreationDate = namespace.Creation_date
	message.CpuLimit = namespace.Cpu_limit
	message.MemLimit = namespace.Mem_limit
	message.StorageLimit = namespace.Storage_limit
	return message

}

func (p *AppMgrHandler) NamespaceList(ctx context.Context, req *appmgr.NamespaceListRequest, rsp *appmgr.NamespaceListResponse) error {
	userId := getUserID(ctx)
	log.Println("app service into NamespaceList")

	namespaces, err := p.db.GetAllNamespace(userId)
	log.Printf(">>>>>>NamespaceMessage  %+v \n", namespaces)
	if err != nil {
		log.Println(err.Error())
		return err
	}

	namespacesWithoutCancel := make([]*common_proto.Namespace, 0)

	for i := 0; i < len(namespaces); i++ {
		if namespaces[i].Status != common_proto.NamespaceStatus_NS_CANCELLED {
			NamespaceMessage := convertFromNamespaceRecord(namespaces[i])
			log.Printf("NamespaceMessage  %+v \n", NamespaceMessage)
			namespacesWithoutCancel = append(namespacesWithoutCancel, &NamespaceMessage)
		}
	}

	rsp.Namespaces = namespacesWithoutCancel

	return nil
}

func (p *AppMgrHandler) UpdateNamespace(ctx context.Context, req *appmgr.UpdateNamespaceRequest, rsp *common_proto.Empty) error {
	userId := getUserID(ctx)
	appDeployment, err := p.checkOwner(userId, req.Namespace.Id)
	if err != nil {
		log.Println(err.Error())
		return err
	}
	namespace := appDeployment.Namespace
	req.Namespace.Name = strings.ToLower(req.Namespace.Name)

	if namespace.NamespaceStatus == common_proto.NamespaceStatus_NS_CANCELLED ||
		namespace.NamespaceStatus == common_proto.NamespaceStatus_NS_DONE {
		log.Println(ankr_default.ErrAppStatusCanNotUpdate.Error())
		return ankr_default.ErrAppStatusCanNotUpdate
	}

	namespace.NamespaceStatus = req.Namespace.NamespaceStatus
	namespace.CpuLimit = req.Namespace.CpuLimit
	namespace.MemLimit = req.Namespace.MemLimit
	namespace.StorageLimit = req.Namespace.StorageLimit

	event := common_proto.DCStream{
		OpType:    common_proto.DCOperation_NS_UPDATE,
		OpPayload: &common_proto.DCStream_Namespace{Namespace: namespace},
	}

	if err := p.deployApp.Publish(context.Background(), &event); err != nil {
		log.Println(err.Error())
		return err
	}
	// TODO: wait deamon notify
	req.Namespace.NamespaceStatus = common_proto.NamespaceStatus_NS_UPDATING
	if err := p.db.UpdateNamespace(req.Namespace.Id, req.Namespace); err != nil {
		log.Println(err.Error())
		return err
	}
	return nil
}

func (p *AppMgrHandler) DeleteNamespace(ctx context.Context, req *appmgr.DeleteNamespaceRequest, rsp *common_proto.Empty) error {
	userId := getUserID(ctx)
	log.Println("Debug into DeleteNamespace")

	namespaceRecord, err := p.db.GetNamespace(req.Id)
	if err != nil {
		log.Println(err.Error())
		return err
	}

	if err := checkId(userId, namespaceRecord.NamespaceUserID); err != nil {
		log.Println(err.Error())
		return err
	}
	namespace, err := p.checkOwner(userId, namespaceRecord.NamespaceUserID)
	if err != nil {
		log.Println(err.Error())
		return err
	}

	if namespace.Status == common_proto.NamespaceStatus_CANCELLED {
		return ankr_default.ErrCanceledTwice
	}

	event := common_proto.DCStream{
		OpType:    common_proto.DCOperation_TASK_CANCEL,
		OpPayload: &common_proto.DCStream_Namespace{Namespace: namespace},
	}

	if err := p.deployApp.Publish(context.Background(), &event); err != nil {
		log.Println(err.Error())
		return err
	}

	if err := p.db.Update(app.Id, bson.M{"$set": bson.M{"status": common_proto.NamespaceStatus_CANCELLED}}); err != nil {
		log.Println(err.Error())
		return err
	}

	return nil
}
