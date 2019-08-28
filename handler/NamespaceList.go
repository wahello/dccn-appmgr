package handler

import (
	"context"
	"errors"
	appmgr "github.com/Ankr-network/dccn-common/protos/appmgr/v1/grpc"
	common_proto "github.com/Ankr-network/dccn-common/protos/common"
	common_util "github.com/Ankr-network/dccn-common/util"
	"github.com/golang/protobuf/ptypes/timestamp"
	"gopkg.in/mgo.v2/bson"
	"log"
	"time"
)

// NamespaceList will return a namespace list for certain user
func (p *AppMgrHandler) NamespaceList(ctx context.Context, req *common_proto.Empty) (*appmgr.NamespaceListResponse, error) {
	rsp := &appmgr.NamespaceListResponse{}

	_, teamId := common_util.GetUserIDAndTeamID(ctx)
	if len(teamId) == 0 {
		return rsp, errors.New("timeId not found in context")
	}

	namespaceRecords, err := p.db.GetAllNamespaces(teamId)
	if err != nil {
		log.Printf("GetAllNamespaces error: %v", err)
		return rsp, err
	}

	namespacesWithoutCancel := make([]*common_proto.NamespaceReport, 0)

	for _, ns := range namespaceRecords {
		if !ns.Hidden && (ns.Status == common_proto.NamespaceStatus_NS_CANCELED || ns.Status == common_proto.NamespaceStatus_NS_CANCELING) &&
			(ns.LastModifiedDate == nil || ns.LastModifiedDate.Seconds < (time.Now().Unix()-7200)) {
			ns.Hidden = true
			p.db.Update("namespace", ns.ID, bson.M{"$set": bson.M{"hidden": true,
				"lastmodifieddate": &timestamp.Timestamp{Seconds: time.Now().Unix()}}})
		}

		if !ns.Hidden {
			namespaceMessage := convertFromNamespaceRecord(ns)
			if len(ns.ClusterID) == 0 {
				log.Printf("clusterID for ns %+v is empty, skip check connection", ns)
			} else {
				clusterConnection, err := p.db.GetClusterConnection(ns.ClusterID)
				if err != nil || clusterConnection.Status == common_proto.DCStatus_UNAVAILABLE {
					namespaceMessage.NsStatus = common_proto.NamespaceStatus_NS_UNAVAILABLE
					namespaceMessage.NsEvent = common_proto.NamespaceEvent_NS_HEARBEAT_FAILED
				}
			}
			namespacesWithoutCancel = append(namespacesWithoutCancel, &namespaceMessage)
		}
	}

	rsp.NamespaceReports = namespacesWithoutCancel

	return rsp, nil
}
