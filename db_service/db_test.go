//+build integration

package dbservice

import (
	"github.com/Ankr-network/dccn-appmgr/config"
	"github.com/Ankr-network/dccn-common/protos/common"
	"gopkg.in/mgo.v2/bson"
	"log"
	"testing"
)

var (
	testDB *DB
)

func init() {
	localDB, err := New(config.Default.DB)
	if err != nil {
		log.Fatal(err)
	}
	testDB = localDB
}

func TestDB_UpdateByHeartbeatMetrics(t *testing.T) {
	s := testDB.session.Copy()
	defer s.Close()

	var c ClusterConnectionRecord
	s.DB("dccn").C("clusterconnection").Find(bson.M{"status": common_proto.DCStatus_AVAILABLE}).One(&c)

	t.Log(len(c.Metrics.NsUsed))
	testDB.UpdateByHeartbeatMetrics(c.ID, c.Metrics)
}
