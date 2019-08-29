package handler

import (
	"context"
	"errors"
	"github.com/Ankr-network/dccn-common/protos"
	"github.com/Ankr-network/dccn-common/protos/appmgr/v1/grpc"
	"github.com/Ankr-network/dccn-common/protos/common"
	commonutil "github.com/Ankr-network/dccn-common/util"
	"github.com/golang/protobuf/ptypes/timestamp"
	"gopkg.in/mgo.v2/bson"
	"log"
	"time"
)

// DeleteNamespace will delete a namespace with no resource owned
func (p *AppMgrHandler) DeleteNamespace(ctx context.Context,
	req *appmgr.DeleteNamespaceRequest) (*common_proto.Empty, error) {

	_, teamId := commonutil.GetUserIDAndTeamID(ctx)
	log.Printf(">>>>>>>>>Debug into DeleteNamespace %+v", req)

	namespaceRecord, err := p.db.GetNamespace(req.NsId)
	if err != nil {
		log.Printf("GetNamespace error %v", err)
		return &common_proto.Empty{}, err
	}

	if err := checkNsId(teamId, namespaceRecord.TeamID); err != nil {
		log.Printf("checkNsId error %v", err)
		return &common_proto.Empty{}, err
	}

	if namespaceRecord.Status == common_proto.NamespaceStatus_NS_CANCELED {
		log.Printf("ns %s already canceled", namespaceRecord.ID)
		return &common_proto.Empty{}, ankr_default.ErrCanceledTwice
	}

	if namespaceRecord.Status == common_proto.NamespaceStatus_NS_FAILED || namespaceRecord.Status == common_proto.NamespaceStatus_NS_UNAVAILABLE {
		if err := p.db.Update("namespace", req.NsId, bson.M{"$set": bson.M{
			"status":           common_proto.NamespaceStatus_NS_CANCELED,
			"hidden":           true,
			"lastmodifieddate": &timestamp.Timestamp{Seconds: time.Now().Unix()},
		}}); err != nil {
			log.Printf("mark ns %s to canceled error: %v", req.NsId, err)
			return &common_proto.Empty{}, err
		} else {
			if changeInfo, err := p.db.UpdateMany("app", bson.M{
				"namespaceid": req.NsId,
				"hidden":      bson.M{"$ne": true},
			}, bson.M{
				"$set": bson.M{"hidden": true},
			}); err != nil {

				return &common_proto.Empty{}, err
			} else {
				log.Printf("hide app of namespace %s, changeInfo:%+v", req.NsId, changeInfo)
				return &common_proto.Empty{}, nil
			}
		}
	}

	apps, err := p.db.GetAllAppsByNamespaceId(req.NsId)
	if err != nil {
		log.Printf("GetAllAppsByNamespaceId error: %v", err)
		return &common_proto.Empty{}, err
	}
	for _, app := range apps {
		if app.Status == common_proto.AppStatus_APP_UNAVAILABLE {
			log.Printf("cancel unavailable app %s", app.ID)
			if err := p.db.Update("app", app.ID, bson.M{
				"status":           common_proto.AppStatus_APP_CANCELED,
				"lastmodifieddate": &timestamp.Timestamp{Seconds: time.Now().Unix()},
			}); err != nil {
				log.Printf("Update app %s status to canceled error: %v", app.ID, err)
				return &common_proto.Empty{}, err
			}
		} else if app.Status != common_proto.AppStatus_APP_CANCELED &&
			app.Status != common_proto.AppStatus_APP_CANCELING &&
			app.Status != common_proto.AppStatus_APP_FAILED {
			log.Printf("app %s is running, cannot delete namespace", app.ID)
			return &common_proto.Empty{}, errors.New(ankr_default.LogicError + "namespace still got running app, can not delete")
		}
	}

	namespaceReport := convertFromNamespaceRecord(namespaceRecord)

	event := common_proto.DCStream{
		OpType:    common_proto.DCOperation_NS_CANCEL,
		OpPayload: &common_proto.DCStream_Namespace{Namespace: namespaceReport.Namespace},
	}

	if err := p.deployApp.Publish(&event); err != nil {
		log.Printf("publish namespace cancel message error: %v", err)
		return &common_proto.Empty{}, errors.New(ankr_default.PublishError + err.Error())
	}

	if err := p.db.Update("namespace", req.NsId, bson.M{"$set": bson.M{
		"status":           common_proto.NamespaceStatus_NS_CANCELING,
		"lastmodifieddate": &timestamp.Timestamp{Seconds: time.Now().Unix()},
	}}); err != nil {
		log.Printf("Update namespace status to canceling error: %v", err)
		return &common_proto.Empty{}, err
	}

	return &common_proto.Empty{}, nil
}
