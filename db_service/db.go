package dbservice

import (
	"log"
	"time"

	dbcommon "github.com/Ankr-network/dccn-common/db"
	common_proto "github.com/Ankr-network/dccn-common/protos/common"

	mgo "gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
)

type DBService interface {
	// GetApp gets a app item by appmgr's id.
	GetApp(id string) (AppRecord, error)
	// GetAll gets all app related to user id.
	GetAllApp(userId string) ([]AppRecord, error)
	GetAppCount(userId string, clusterId string) ([]AppRecord, error)
	GetNamespace(namespaceId string) (NamespaceRecord, error)
	GetAllNamespace(userId string) ([]NamespaceRecord, error)
	// CancelApp sets app status CANCEL
	CancelApp(appId string) error
	CancelNamespace(namespaceId string) error
	CreateNamespace(namespace *common_proto.Namespace, uid string) error
	// Create Creates a new dc item if not exits.
	CreateApp(appDeployment *common_proto.AppDeployment, uid string) error
	// Update updates collection item
	Update(collectId string, Id string, update bson.M) error
	// UpdateApp updates app item
	UpdateApp(app *common_proto.AppDeployment) error
	// Close closes db connection
	UpdateNamespace(namespace *common_proto.Namespace) error

	Close()
	// for test usage
	dropCollection()
}

// UserDB implements DBService
type DB struct {
	dbName              string
	collectionName      string
	eventCollectionName string
	session             *mgo.Session
}

type AppRecord struct {
	ID            string
	UID           string
	Name          string
	NamespaceID   string                 // mongodb name is low field
	Status        common_proto.AppStatus // 1 new 2 running 3. done 4 cancelling 5.canceled 6. updating 7. updateFailed
	Event         common_proto.AppEvent
	Detail        string
	Report        string
	Hidden        bool
	Attributes    common_proto.AppAttributes
	ChartDetail   common_proto.ChartDetail
	ChartUpdating common_proto.ChartDetail
}

// New returns DBService.
func New(conf dbcommon.Config) (*DB, error) {
	session, err := dbcommon.CreateDBConnection(conf)
	if err != nil {
		return nil, err
	}

	return &DB{
		dbName:         conf.DB,
		collectionName: conf.Collection,
		session:        session,
	}, nil
}

func (p *DB) collection(session *mgo.Session, collection string) *mgo.Collection {
	return session.DB(p.dbName).C(collection)
}

// Get gets app item by id.
func (p *DB) GetApp(appId string) (AppRecord, error) {
	session := p.session.Clone()
	defer session.Close()

	var app AppRecord
	err := p.collection(session, "app").Find(bson.M{"id": appId}).One(&app)
	return app, err
}

// Get gets app count by userid and clusterid.
func (p *DB) GetAppCount(userId string, clusterId string) ([]AppRecord, error) {
	session := p.session.Clone()
	defer session.Close()

	var apps []AppRecord

	log.Printf("find app count with uid %s clusterid %s", userId, clusterId)

	if err := p.collection(session, "app").Find(bson.M{"uid": userId}).All(&apps); err != nil {
		return nil, err
	}

	var res []AppRecord

	if len(clusterId) == 0 {
		for _, v := range apps {
			var namespace NamespaceRecord
			if err := p.collection(session, "namespace").Find(bson.M{"id": v.NamespaceID}).One(&namespace); err != nil {
				return nil, err
			}
			if namespace.ClusterID == clusterId {
				res = append(res, v)
			}
		}
	} else {
		res = apps
	}

	return res, nil
}

func (p *DB) GetAllApp(userId string) ([]AppRecord, error) {
	session := p.session.Clone()
	defer session.Close()

	var apps []AppRecord

	log.Printf("find apps with uid %s", userId)

	if err := p.collection(session, "app").Find(bson.M{"uid": userId}).All(&apps); err != nil {
		return nil, err
	}
	return apps, nil
}

// GetByEvent gets app by event id.
func (p *DB) GetByEvent(event string) (*[]*common_proto.App, error) {
	session := p.session.Copy()
	defer session.Close()

	var apps []*common_proto.App
	if err := p.collection(session, "app").Find(bson.M{"event": event}).One(&apps); err != nil {
		return nil, err
	}
	return &apps, nil
}

// CreateApp creates a new app deployment item if it not exists
func (p *DB) CreateApp(appDeployment *common_proto.AppDeployment, uid string) error {
	session := p.session.Copy()
	defer session.Close()

	appRecord := AppRecord{}
	appRecord.ID = appDeployment.Id
	appRecord.UID = uid
	appRecord.Name = appDeployment.Name
	appRecord.Status = common_proto.AppStatus_APP_DISPATCHING
	appRecord.Event = common_proto.AppEvent_DISPATCH_APP
	appRecord.NamespaceID = appDeployment.Namespace.Id
	appRecord.ChartDetail = *appDeployment.ChartDetail
	now := time.Now().Unix()
	appRecord.Attributes.LastModifiedDate = uint64(now)
	appRecord.Attributes.CreationDate = uint64(now)
	return p.collection(session, "app").Insert(appRecord)
}

// Update updates item.
func (p *DB) Update(collection string, id string, update bson.M) error {
	session := p.session.Copy()
	defer session.Close()

	return p.collection(session, collection).Update(bson.M{"id": id}, update)
}

func (p *DB) UpdateApp(appDeployment *common_proto.AppDeployment) error {
	session := p.session.Copy()
	defer session.Close()

	fields := bson.M{}
	if len(appDeployment.Name) > 0 {
		fields["name"] = appDeployment.Name
	}

	fields["event"] = common_proto.AppEvent_UPDATE_APP
	fields["status"] = common_proto.AppStatus_APP_UPDATING

	now := time.Now().Unix()
	fields["lastmodifieddate"] = now
	fields["chartupdating"] = appDeployment.ChartDetail

	return p.collection(session, "app").Update(
		bson.M{"id": appDeployment.Id}, bson.M{"$set": fields})

}

// Cancel cancel app, sets app status CANCEL
func (p *DB) CancelApp(appId string) error {
	session := p.session.Copy()
	defer session.Close()

	now := time.Now().Unix()
	return p.collection(session, "app").Update(bson.M{"id": appId},
		bson.M{"$set": bson.M{"status": common_proto.AppStatus_APP_CANCELED,
			"attributes": bson.M{"lastmodifieddate": now}}})
}

// Close closes the db connection.
func (p *DB) Close() {
	p.session.Close()
}

func (p *DB) dropCollection() {
	err := p.session.DB(p.dbName).C(p.collectionName).DropCollection()
	if err != nil {
		log.Println(err.Error())
	}
}

type NamespaceRecord struct {
	ID                   string // short hash of uid+name+cluster_id
	Name                 string
	NameUpdating         string
	UID                  string
	ClusterID            string //id of cluster
	ClusterName          string //name of cluster
	LastModifiedDate     uint64
	CreationDate         uint64
	CpuLimit             uint64
	CpuLimitUpdating     uint64
	MemLimit             uint64
	MemLimitUpdating     uint64
	StorageLimit         uint64
	StorageLimitUpdating uint64
	Status               common_proto.NamespaceStatus
	Event                common_proto.NamespaceEvent
}

func (p *DB) GetNamespace(namespaceId string) (NamespaceRecord, error) {
	session := p.session.Clone()
	defer session.Close()

	var namespace NamespaceRecord
	err := p.collection(session, "namespace").Find(bson.M{"id": namespaceId}).One(&namespace)

	return namespace, err
}

func (p *DB) GetAllNamespace(userId string) ([]NamespaceRecord, error) {
	session := p.session.Clone()
	defer session.Close()

	var namespaces []NamespaceRecord

	log.Printf("find apps with uid %s", userId)

	if err := p.collection(session, "namespace").Find(bson.M{"uid": userId}).All(&namespaces); err != nil {
		return nil, err
	}
	return namespaces, nil
}

func (p *DB) CreateNamespace(namespace *common_proto.Namespace, uid string) error {
	session := p.session.Copy()
	defer session.Close()

	namespacerecord := NamespaceRecord{}
	namespacerecord.ID = namespace.Id
	namespacerecord.Name = namespace.Name
	namespacerecord.UID = uid
	namespacerecord.Status = common_proto.NamespaceStatus_NS_DISPATCHING
	namespacerecord.Event = common_proto.NamespaceEvent_DISPATCH_NS
	now := time.Now().Unix()
	namespacerecord.LastModifiedDate = uint64(now)
	namespacerecord.CreationDate = uint64(now)
	namespacerecord.CpuLimit = namespace.CpuLimit
	namespacerecord.MemLimit = namespace.MemLimit
	namespacerecord.StorageLimit = namespace.StorageLimit
	return p.collection(session, "namespace").Insert(namespacerecord)
}

func (p *DB) UpdateNamespace(namespace *common_proto.Namespace) error {
	session := p.session.Copy()
	defer session.Close()

	fields := bson.M{}

	if len(namespace.Name) > 0 {
		fields["name"] = namespace.Name
	}

	if namespace.CpuLimit > 0 && namespace.MemLimit > 0 && namespace.StorageLimit > 0 {
		fields["cpulimitupdating"] = namespace.CpuLimit
		fields["memlimitupdating"] = namespace.MemLimit
		fields["storagelimitupdating"] = namespace.StorageLimit
	}

	return p.collection(session, "namespace").Update(bson.M{"id": namespace.Id},
		bson.M{"$set": fields})

}

func (p *DB) CancelNamespace(NamespaceId string) error {
	session := p.session.Copy()
	defer session.Close()

	now := time.Now().Unix()
	return p.collection(session, "namespace").Update(
		bson.M{"id": NamespaceId},
		bson.M{"$set": bson.M{"status": common_proto.NamespaceStatus_NS_CANCELED,
			"Last_modified_date": now}})
}
