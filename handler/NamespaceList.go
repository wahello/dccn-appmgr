package handler

import (
	appmgr "github.com/Ankr-network/dccn-common/protos/appmgr/v1/grpc"
	common_proto "github.com/Ankr-network/dccn-common/protos/common"
	common_util "github.com/Ankr-network/dccn-common/util"
	"github.com/golang/protobuf/ptypes/timestamp"
	"gopkg.in/mgo.v2/bson"
	"log"
	"context"
	"time"
	"errors"
)

// NamespaceList will return a namespace list for certain user
func (p *AppMgrHandler) NamespaceList(ctx context.Context,
	req *common_proto.Empty) (*appmgr.NamespaceListResponse, error) {

	rsp := &appmgr.NamespaceListResponse{}
	_, teamId := common_util.GetUserIDAndTeamID(ctx)
	if len(teamId) == 0 {
		return rsp, errors.New("user id not found in context")
	}
	log.Printf(">>>>>>>>>Debug into NamespaceList, ctx: %+v\n", ctx)

	namespaceRecords, err := p.db.GetAllNamespace(teamId)
	log.Printf("NamespaceMessage  %+v \n", namespaceRecords)
	if err != nil {
		log.Println(err.Error())
		return rsp, err
	}

	namespacesWithoutCancel := make([]*common_proto.NamespaceReport, 0)

	for i := 0; i < len(namespaceRecords); i++ {

		if !namespaceRecords[i].Hidden && (namespaceRecords[i].Status == common_proto.NamespaceStatus_NS_CANCELED ||
			namespaceRecords[i].Status == common_proto.NamespaceStatus_NS_CANCELING) &&
			(namespaceRecords[i].LastModifiedDate == nil || namespaceRecords[i].LastModifiedDate.Seconds < (time.Now().Unix()-7200)) {
			namespaceRecords[i].Hidden = true
			p.db.Update("namespace", namespaceRecords[i].ID, bson.M{"$set": bson.M{"hidden": true,
				"lastmodifieddate": &timestamp.Timestamp{Seconds: time.Now().Unix()}}})
		}

		if !namespaceRecords[i].Hidden {
			namespaceMessage := convertFromNamespaceRecord(namespaceRecords[i])
			log.Printf("NamespaceMessage  %+v \n", namespaceMessage)
			clusterConnection, err := p.db.GetClusterConnection(namespaceRecords[i].ClusterID)
			if err != nil || clusterConnection.Status == common_proto.DCStatus_UNAVAILABLE {
				namespaceMessage.NsStatus = common_proto.NamespaceStatus_NS_UNAVAILABLE
				namespaceMessage.NsEvent = common_proto.NamespaceEvent_NS_HEARBEAT_FAILED
			}
			namespacesWithoutCancel = append(namespacesWithoutCancel, &namespaceMessage)
		}
	}

	rsp.NamespaceReports = namespacesWithoutCancel

	return rsp, nil
}
