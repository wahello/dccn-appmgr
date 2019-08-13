package dbservice

import (
	"errors"
	"log"
	"time"

	dbcommon "github.com/Ankr-network/dccn-common/db"
	ankr_default "github.com/Ankr-network/dccn-common/protos"
	common_proto "github.com/Ankr-network/dccn-common/protos/common"
	"github.com/golang/protobuf/ptypes/timestamp"
	mgo "gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
)

type DBService interface {
	// GetApp gets a app item by app's id.
	GetApp(id string) (AppRecord, error)
	// GetAllApp gets all app related to user id.
	GetAllApp(userId string) ([]AppRecord, error)
	// GetAllAppByNamespaceId gets all app related to namespace id.
	GetAllAppByNamespaceId(namespaceId string) ([]AppRecord, error)
	// GetAppCount gets all app related to user id in specific cluster.
	GetAppCount(userId string, clusterId string) ([]AppRecord, error)
	// GetNamespace gets a namespace item by namespace's id.
	GetNamespace(namespaceId string) (NamespaceRecord, error)
	// GetAllNamespaceByClusterId gets a namespace item by cluster's id.
	GetAllNamespaceByClusterId(clusterId string) ([]NamespaceRecord, error)
	// GetAllNamespace gets a namespace item by user's id.
	GetAllNamespace(userId string) ([]NamespaceRecord, error)
	// CancelApp sets app status CANCEL
	CancelApp(appId string) error
	// CancelNamespace sets namespace status CANCEL
	CancelNamespace(namespaceId string) error
	// CreateNamespace create new namespace
	CreateNamespace(namespace *common_proto.Namespace, uid string) error
	// Create Creates a new app item if not exits.
	CreateApp(appDeployment *common_proto.AppDeployment, uid string) error
	// Update updates collection item
	Update(collectId string, Id string, update bson.M) error
	// UpdateApp updates app item
	UpdateApp(app *common_proto.AppDeployment) error
	// Close closes db connection
	UpdateNamespace(namespace *common_proto.Namespace) error
	// Create a new cluster connection
	CreateClusterConnection(clusterID string, clusterStatus common_proto.DCStatus) error
	// GetClusterConnection gets a cluster connection item by cluster's id.
	GetClusterConnection(clusterID string) (ClusterConnectionRecord, error)

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
	ID                   string
	UID                  string
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
	if err != nil {
		return app, errors.New(ankr_default.DbError + err.Error())
	}

	return app, err
}

// Get gets app count by userid and clusterid.
func (p *DB) GetAppCount(userId string, clusterId string) ([]AppRecord, error) {
	session := p.session.Clone()
	defer session.Close()

	var apps []AppRecord

	log.Printf("find app count with uid %s clusterid %s", userId, clusterId)
	if err := p.collection(session, "app").Find(bson.M{"uid": userId}).All(&apps); err != nil {
		return nil, errors.New(ankr_default.DbError + err.Error())
	}

	var res []AppRecord

	if len(clusterId) == 0 {
		for _, v := range apps {
			var namespace NamespaceRecord
			if err := p.collection(session, "namespace").Find(bson.M{"id": v.NamespaceID}).One(&namespace); err != nil {
				return nil, errors.New(ankr_default.DbError + err.Error())
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
		return nil, errors.New(ankr_default.DbError + err.Error())
	}
	return apps, nil
}

func (p *DB) GetAllAppByNamespaceId(namespaceId string) ([]AppRecord, error) {
	session := p.session.Clone()
	defer session.Close()

	var apps []AppRecord

	log.Printf("find apps with namespace id %s", namespaceId)

	if err := p.collection(session, "app").Find(bson.M{"namespaceid": namespaceId}).All(&apps); err != nil {
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
		return nil, errors.New(ankr_default.DbError + err.Error())
	}
	return &apps, nil
}

// CreateApp creates a new app deployment item if it not exists
func (p *DB) CreateApp(appDeployment *common_proto.AppDeployment, uid string) error {
	session := p.session.Copy()
	defer session.Close()

	appRecord := AppRecord{}
	appRecord.ID = appDeployment.AppId
	appRecord.UID = uid
	appRecord.Name = appDeployment.AppName
	appRecord.Status = common_proto.AppStatus_APP_DISPATCHING
	appRecord.Event = common_proto.AppEvent_DISPATCH_APP
	appRecord.NamespaceID = appDeployment.Namespace.NsId
	appRecord.ChartDetail = *appDeployment.ChartDetail
	now := time.Now().Unix()
	appRecord.LastModifiedDate = &timestamp.Timestamp{Seconds: now}
	appRecord.CreationDate = &timestamp.Timestamp{Seconds: now}
	appRecord.CustomValues = appDeployment.CustomValues
	err := p.collection(session, "app").Insert(appRecord)
	if err != nil {
		return errors.New(ankr_default.DbError + err.Error())
	}
	return nil
}

// Update updates item.
func (p *DB) Update(collection string, id string, update bson.M) error {
	session := p.session.Copy()
	defer session.Close()

	err := p.collection(session, collection).Update(bson.M{"id": id}, update)
	if err != nil {
		return errors.New(ankr_default.DbError + err.Error())
	}
	return nil
}

func (p *DB) UpdateApp(appDeployment *common_proto.AppDeployment) error {
	session := p.session.Copy()
	defer session.Close()

	fields := bson.M{}
	if len(appDeployment.AppName) > 0 {
		fields["name"] = appDeployment.AppName
	}

	fields["event"] = common_proto.AppEvent_UPDATE_APP
	fields["status"] = common_proto.AppStatus_APP_UPDATING

	now := time.Now().Unix()
	fields["lastmodifieddate"] = &timestamp.Timestamp{Seconds: now}
	fields["chartupdating"] = appDeployment.ChartDetail
	fields["customvaluesupdating"] = appDeployment.CustomValues

	err := p.collection(session, "app").Update(
		bson.M{"id": appDeployment.AppId}, bson.M{"$set": fields})
	if err != nil {
		return errors.New(ankr_default.DbError + err.Error())
	}
	return nil

}

// Cancel cancel app, sets app status CANCEL
func (p *DB) CancelApp(appId string) error {
	session := p.session.Copy()
	defer session.Close()

	now := time.Now().Unix()
	err := p.collection(session, "app").Update(bson.M{"id": appId},
		bson.M{"$set": bson.M{"status": common_proto.AppStatus_APP_CANCELED,
			"lastmodifieddate": &timestamp.Timestamp{Seconds: now}}})
	if err != nil {
		return errors.New(ankr_default.DbError + err.Error())
	}
	return nil
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
	LastModifiedDate     *timestamp.Timestamp
	CreationDate         *timestamp.Timestamp
	CpuLimit             uint32
	CpuLimitUpdating     uint32
	MemLimit             uint32
	MemLimitUpdating     uint32
	StorageLimit         uint32
	StorageLimitUpdating uint32
	Status               common_proto.NamespaceStatus
	Event                common_proto.NamespaceEvent
	Hidden               bool
}

func (p *DB) GetNamespace(namespaceId string) (NamespaceRecord, error) {
	session := p.session.Clone()
	defer session.Close()

	var namespace NamespaceRecord
	err := p.collection(session, "namespace").Find(bson.M{"id": namespaceId}).One(&namespace)
	if err != nil {
		return namespace, errors.New(ankr_default.DbError + err.Error())
	}
	return namespace, err
}

func (p *DB) GetAllNamespace(userId string) ([]NamespaceRecord, error) {
	session := p.session.Clone()
	defer session.Close()

	var namespaces []NamespaceRecord

	log.Printf("find apps with uid %s", userId)

	if err := p.collection(session, "namespace").Find(bson.M{"uid": userId}).All(&namespaces); err != nil {
		return nil, errors.New(ankr_default.DbError + err.Error())
	}
	return namespaces, nil
}

func (p *DB) GetAllNamespaceByClusterId(clusterId string) ([]NamespaceRecord, error) {
	session := p.session.Clone()
	defer session.Close()

	var namespaces []NamespaceRecord

	log.Printf("find apps with cluster id %s", clusterId)

	if err := p.collection(session, "namespace").Find(bson.M{"clusterid": clusterId}).All(&namespaces); err != nil {
		return nil, err
	}
	return namespaces, nil
}

func (p *DB) CreateNamespace(namespace *common_proto.Namespace, uid string) error {
	session := p.session.Copy()
	defer session.Close()

	namespacerecord := NamespaceRecord{}
	namespacerecord.ID = namespace.NsId
	namespacerecord.Name = namespace.NsName
	namespacerecord.UID = uid
	namespacerecord.Status = common_proto.NamespaceStatus_NS_DISPATCHING
	namespacerecord.Event = common_proto.NamespaceEvent_DISPATCH_NS
	now := time.Now().Unix()
	namespacerecord.LastModifiedDate = &timestamp.Timestamp{Seconds: now}
	namespacerecord.CreationDate = &timestamp.Timestamp{Seconds: now}
	namespacerecord.CpuLimit = namespace.NsCpuLimit
	namespacerecord.MemLimit = namespace.NsMemLimit
	namespacerecord.StorageLimit = namespace.NsStorageLimit
	err := p.collection(session, "namespace").Insert(namespacerecord)
	if err != nil {
		return errors.New(ankr_default.DbError + err.Error())
	}
	return nil
}

func (p *DB) UpdateNamespace(namespace *common_proto.Namespace) error {
	session := p.session.Copy()
	defer session.Close()

	fields := bson.M{}

	if len(namespace.NsName) > 0 {
		fields["name"] = namespace.NsName
	}

	if namespace.NsCpuLimit > 0 && namespace.NsMemLimit > 0 && namespace.NsStorageLimit > 0 {
		fields["cpulimitupdating"] = namespace.NsCpuLimit
		fields["memlimitupdating"] = namespace.NsMemLimit
		fields["storagelimitupdating"] = namespace.NsStorageLimit
		fields["status"] = common_proto.NamespaceStatus_NS_UPDATING
	}

	now := time.Now().Unix()
	fields["lastmodifieddate"] = &timestamp.Timestamp{Seconds: now}

	err := p.collection(session, "namespace").Update(bson.M{"id": namespace.NsId},
		bson.M{"$set": fields})
	if err != nil {
		return errors.New(ankr_default.DbError + err.Error())
	}
	return nil
}

func (p *DB) CancelNamespace(NamespaceId string) error {
	session := p.session.Copy()
	defer session.Close()

	now := time.Now().Unix()
	err := p.collection(session, "namespace").Update(
		bson.M{"id": NamespaceId},
		bson.M{"$set": bson.M{"status": common_proto.NamespaceStatus_NS_CANCELED,
			"lastmodifieddate": &timestamp.Timestamp{Seconds: now}}})
	if err != nil {
		return errors.New(ankr_default.DbError + err.Error())
	}
	return nil
}

type ClusterConnectionRecord struct {
	ID               string
	Status           common_proto.DCStatus
	LastModifiedDate *timestamp.Timestamp
	CreationDate     *timestamp.Timestamp
}

func (p *DB) GetClusterConnection(clusterID string) (ClusterConnectionRecord, error) {
	session := p.session.Clone()
	defer session.Close()

	var clusterConnection ClusterConnectionRecord
	err := p.collection(session, "clusterconnection").Find(bson.M{"id": clusterID}).One(&clusterConnection)

	return clusterConnection, err
}

func (p *DB) CreateClusterConnection(clusterID string, clusterStatus common_proto.DCStatus) error {
	session := p.session.Copy()
	defer session.Close()

	now := time.Now().Unix()
	clusterConnection := &ClusterConnectionRecord{
		ID:               clusterID,
		Status:           clusterStatus,
		LastModifiedDate: &timestamp.Timestamp{Seconds: now},
		CreationDate:     &timestamp.Timestamp{Seconds: now},
	}

	return p.collection(session, "clusterconnection").Insert(clusterConnection)
}
