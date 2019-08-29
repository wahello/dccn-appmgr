package dbservice

import (
	"github.com/Ankr-network/dccn-common/protos/common"
	"github.com/golang/protobuf/ptypes/timestamp"
)

type AppRecord struct {
	ID                   string
	TeamID               string
	Name                 string
	NamespaceID          string                 // mongodb name is low field
	Status               common_proto.AppStatus // 1 new 2 running 3. done 4 cancelling 5.canceled 6. updating 7. updateFailed
	Event                common_proto.AppEvent
	Detail               string
	Report               string
	Hidden               bool
	CreationDate         *timestamp.Timestamp
	LastModifiedDate     *timestamp.Timestamp
	ChartDetail          common_proto.ChartDetail
	ChartUpdating        common_proto.ChartDetail
	CustomValues         []*common_proto.CustomValue
	CustomValuesUpdating []*common_proto.CustomValue
	Creator              string
}

type NamespaceRecord struct {
	ID                   string // short hash of uid+name+cluster_id
	Name                 string
	NameUpdating         string
	TeamID               string
	ClusterID            string //id of cluster
	ClusterName          string //name of cluster
	LastModifiedDate     *timestamp.Timestamp
	CreationDate         *timestamp.Timestamp
	CpuLimit             uint32
	CpuLimitUpdating     uint32
	MemLimit             uint32
	MemLimitUpdating     uint32
	StorageLimit         uint32
	StorageLimitUpdating uint32
	CpuUsage             uint32
	MemUsage             uint32
	StorageUsage         uint32
	Status               common_proto.NamespaceStatus
	Event                common_proto.NamespaceEvent
	Hidden               bool
	Creator              string
}

type ClusterConnectionRecord struct {
	ID               string
	Status           common_proto.DCStatus
	Metrics          *common_proto.DCHeartbeatReport_Metrics
	LastModifiedDate *timestamp.Timestamp
	CreationDate     *timestamp.Timestamp
}
