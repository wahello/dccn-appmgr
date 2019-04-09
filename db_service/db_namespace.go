package dbservice

import (
	"log"
	"time"

//	dbcommon "github.com/Ankr-network/dccn-common/db"
	common_proto "github.com/Ankr-network/dccn-common/protos/common"

//	mgo "gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
)

type NamespaceRecord struct {
	NamespaceID string// short hash of uid+name+cluster_id
	Name string 
	NamespaceUserID string 
	Status common_proto.NamespaceStatus
	Cluster_ID string //id of cluster
	Cluster_Name string //name of cluster
	Creation_date uint64
	Cpu_limit float64
	Mem_limit uint64
	Storage_limit uint64
	}


func (p *DB) GetNamespace(NamespaceId string) (NamespaceRecord, error) {
	session := p.session.Clone()
	defer session.Close()	

	var namespace NamespaceRecord
	err := p.collection(session).Find(bson.M{"namespaceid": NamespaceId}).One(&namespace)
	return namespace, err
}


func (p *DB) GetAllNamespace(userId string) ([]NamespaceRecord, error) {
	session := p.session.Clone()
	defer session.Close()

	var namespaces []NamespaceRecord

	log.Printf("find tasks with uid %s", userId)

	if err := p.collection(session).Find(bson.M{"namespaceuserid": userId}).All(&namespaces); err != nil {
		return nil, err
	}
	return namespaces, nil
}

func (p *DB) CreateNamespace(namespace *common_proto.Namespace, uid string) error {
	session := p.session.Copy()
	defer session.Close()

	namespacerecord := NamespaceRecord{}
	namespacerecord.NamespaceID = namespace.Id
	namespacerecord.Name = namespace.Name
	namespacerecord.NamespaceUserID = uid
	namespacerecord.Cluster_ID = namespace.Cluster_ID
	namespacerecord.Status = namespace.Status
	namespacerecord.Cluster_Name = namespace.Cluster_Name
	namespacerecord.Creation_date = namespace.Creation_date
	namespacerecord.Cpu_limit = namespace.Cpu_limit
	namespacerecord.Mem_limit = namespace.Mem_limit
	namespacerecord.Storage_limit = namespace.Storage_limit
	return p.collection(session).Insert(namespacerecord)
}


func (p *DB) UpdateNamespace(NamespaceId string, Namespace *common_proto.Namespace) error {
	session := p.session.Copy()
	defer session.Close()

	fields := bson.M{}

	namespacerecord.Cpu_limit = namespace.Cpu_limit
	namespacerecord.Mem_limit = namespace.Mem_limit
	namespacerecord.Storage_limit = namespace.Storage_limit
	namespacerecord.Status = namespace.Status
	if len(Namespace.Name) > 0 {
		fields["name"] = namespace.Name
	}

	if Namespace.Status > 0 {
		fields["status"] = Namespace.Status
	}
	if Namespace.Cpu_limit > 0 {
		fields["Cpu_limit"] = Namespace.Cpu_limit
	}

	if Namespace.Mem_limit > 0 {
		fields["Mem_limit"] = namespace.Mem_limit
	}
	if Namespace.Storage_limit > 0 {
		fields["Storage_limit"] = namespace.Storage_limit
	}
	return p.collection(session).Update(bson.M{"Namespaceid": NamespaceId}, bson.M{"$set": fields})
	//return p.collection(session).Update(bson.M{"id": taskId}, task)
}

func (p *DB) CancelNamespace(NamespaceId string) error {
	session := p.session.Copy()
	defer session.Close()

	now := time.Now().Unix()
	return p.collection(session).Update(bson.M{"id": NamespaceId}, bson.M{"$set": bson.M{"status": common_proto.NamespaceStatus_CANCELLED, "Last_modified_date" : now}})
}
