package handler

import (
	common_proto "github.com/Ankr-network/dccn-common/protos/common"
	db "github.com/Ankr-network/dccn-appmgr/db_service"
	)

func convertFromNamespaceRecord(namespace db.NamespaceRecord) common_proto.NamespaceReport {
	message := common_proto.Namespace{}
	message.NsId = namespace.ID
	message.NsName = namespace.Name
	message.ClusterId = namespace.ClusterID
	message.ClusterName = namespace.ClusterName
	message.CreationDate = namespace.CreationDate
	message.LastModifiedDate = namespace.LastModifiedDate
	message.NsCpuLimit = namespace.CpuLimit
	message.NsMemLimit = namespace.MemLimit
	message.NsStorageLimit = namespace.StorageLimit
	namespaceReport := common_proto.NamespaceReport{
		Namespace: &message,
		NsEvent:   namespace.Event,
		NsStatus:  namespace.Status,
	}
	return namespaceReport
}