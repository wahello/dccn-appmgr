package subscriber

import (
	dbservice "github.com/Ankr-network/dccn-appmgr/db_service"
	common_proto "github.com/Ankr-network/dccn-common/protos/common"
	"github.com/golang/protobuf/ptypes/timestamp"
	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
	"log"
	"time"
)

type MetricsSubscriber struct {
	DB dbservice.DBService
}

func (p *MetricsSubscriber) Handle(dc *common_proto.DataCenterStatus) error {
	log.Printf("MetricsSubscriber handle metrics message: %+v", dc)
	if dc.DcHeartbeatReport != nil && dc.DcHeartbeatReport.MetricsRaw != nil {
		if _, err := p.DB.GetClusterConnection(dc.DcId); err == mgo.ErrNotFound {
			if err = p.DB.CreateClusterConnection(dc.DcId, dc.DcStatus, dc.DcHeartbeatReport.MetricsRaw); err != nil {
				log.Printf("Create cluster connection failed %v", err)
				return err
			}
		} else {
			// clusterconnection found
			update := bson.M{}
			update["metrics"] = dc.DcHeartbeatReport.MetricsRaw
			update["status"] = dc.DcStatus
			update["lastmodifieddate"] = &timestamp.Timestamp{Seconds: time.Now().Unix()}
			if err := p.DB.Update("clusterconnection", dc.DcId, bson.M{"$set": update}); err != nil {
				return err
			}
		}
		log.Printf("update namespace & app by heartbeat metrics")
		p.DB.UpdateByHeartbeatMetrics(dc.DcId, dc.DcHeartbeatReport.MetricsRaw)
	} else {
		log.Printf("metrics is miss, skip")
	}
	return nil
}
