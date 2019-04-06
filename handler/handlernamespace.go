package handler

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/Ankr-network/dccn-common/protos"
	"github.com/Ankr-network/dccn-common/protos/common"
	"github.com/Ankr-network/dccn-common/protos/appmgr/v1/micro"
	db "github.com/Ankr-network/dccn-appmgr/db_service"
	"github.com/google/uuid"
	"github.com/gorhill/cronexpr"
	micro "github.com/micro/go-micro"
	"github.com/micro/go-micro/metadata"
	"gopkg.in/mgo.v2/bson"
	"k8s.io/helm/pkg/chartutil"
)

type AppMgrHandler struct {
	db         db.DBService
	deployApp micro.Publisher
}

func New(db db.DBService, deployApp micro.Publisher) *AppMgrHandler {

	return &AppMgrHandler{
		db:         db,
		deployApp: deployApp,
	}
}

type Token struct {
	Exp int64
	Jti string
	Iss string
}

func getUserID(ctx context.Context) string {
	meta, ok := metadata.FromContext(ctx)
	// Note this is now uppercase (not entirely sure why this is...)
	var token string
	if ok {
		token = meta["token"]
	}

	parts := strings.Split(token, ".")

	decoded, err := base64.StdEncoding.DecodeString(parts[1])
	if err != nil {
		fmt.Println("decode error:", err)

	}
	fmt.Println(string(decoded))

	var dat Token

	if err := json.Unmarshal(decoded, &dat); err != nil {
		panic(err)
	}

	return string(dat.Jti)
}

func (p *AppMgrHandler) CreateNamespace(ctx context.Context, req *appmgr.CreateNamespaceRequest, rsp *common_proto.Empty) error {
	uid := getUserID(ctx)
	log.Println("app manager service CreateApp")

	log.Printf("CreateNamespace Namespace %+v", req)

	req.Namespace.Id = uuid.New().String()
	req.Namespace.NamespaceUserID = uid

	event := common_proto.DCStream{
		OpType:    common_proto.DCOperation_TASK_CREATE,
		OpPayload: &common_proto.DCStream_Namespace{Namespace: req.namespace},
	}

	if err := p.deployApp.Publish(context.Background(), &event); err != nil {
		log.Println(ankr_default.ErrPublish)
		return ankr_default.ErrPublish
	} else {
		log.Println("app manager service send CreateApp MQ message to dc manager service (api)")
	}

	if err := p.db.CreateNamespace(req.namespace, uid); err != nil {
		log.Println(err.Error())
		return err
	}

	return nil
}
type NamespaceRecord struct {
	NamespaceID string// short hash of uid+name+cluster_id
	Name string
	NamespaceUserID string 
	Cluster_ID string //id of cluster
	Cluster_Name string //name of cluster
	Creation_date uint64
	Cpu_limit float64
	Mem_limit uint64
	Storage_limit uint64
	}
func convertFromNamespaceRecord(namespace db.NamespaceRecord) common_proto.Namespace {
	message := common_proto.Namespace{}
	message.Id = Namespace.NamespaceID
	message.Cluster_ID = Namespace.Cluster_ID
	message.Cluster_Name = Namespace.Cluster_Name
	message.Status = Namespace.Status
	message.Creation_date = Namespace.Creation_date
	message.Cpu_limit = Namespace.Cpu_limit
	message.Mem_limit = Namespace.Mem_limit
	message.Storage_limit = Namespace.Storage_limit
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
		if namespaces[i].Status != common_proto.NamespaceStatus_CANCELLED {
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
	namespace, err := p.checkOwner(userId, req.Namespace.Id)
	if err != nil {
		log.Println(err.Error())
		return err
	}

	req.Namespace.Name = strings.ToLower(req.Namespace.Name)

	if namespace.Status == common_proto.NamespaceStatus_CANCELLED ||
	namespace.Status == common_proto.NamespaceStatus_DONE {
		log.Println(ankr_default.ErrAppStatusCanNotUpdate.Error())
		return ankr_default.ErrAppStatusCanNotUpdate
	}

	namespace.Status = req.Namespace.Status
	namespace.Cpu_limit = req.Namespace.Cpu_limit
	namespace.Mem_limit = req.Namespace.Mem_limit
	namespace.Storage_limit = req.Namespace.Storage_limit

	event := common_proto.DCStream{
		OpType:    common_proto.DCOperation_TASK_UPDATE,
		OpPayload: &common_proto.DCStream_Namespace{Namespace: namespace},
	}

	if err := p.deployApp.Publish(context.Background(), &event); err != nil {
		log.Println(err.Error())
		return err
	}
	// TODO: wait deamon notify
	req.Namespace.Status = common_proto.NamespaceStatus_UPDATING
	if err := p.db.UpdateApp(req.Namespace.Id, req.Namespace); err != nil {
		log.Println(err.Error())
		return err
	}
	return nil
}



func (p *AppMgrHandler) DeleteNamespace(ctx context.Context, req *appmgr.NamespaceID, rsp *common_proto.Empty) error {
	userId := getUserID(ctx)
	log.Println("Debug into DeleteNamespace")
	if err := checkId(userId, req.NamespaceUserID); err != nil {
		log.Println(err.Error())
		return err
	}
	namespace, err := p.checkOwner(userId, req.NamespaceUserID)
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

func (p *AppMgrHandler) checkOwner(userId, NamespaceId string) (*db.Namespace, error) {
	namespace, err := p.db.GetNamespace(NamespaceId)

	if err != nil {
		return nil, err
	}

	log.Printf("NamespaceId : %s user id -%s-   user_token_id -%s-  ", NamespaceId, namespace.NamespaceUserid, userId)

	if namespace.NamespaceUserid != userId {
		return nil, ankr_default.ErrUserNotOwn
	}
	namespacenessage := convertFromNamespaceRecord(namespace)
	return &namespacemessage, nil
}

func checkId(userId, appId string) error {
	if userId == "" {
		return ankr_default.ErrUserNotExist
	}

	if appId == "" {
		return ankr_default.ErrUserNotOwn
	}

	return nil
}
