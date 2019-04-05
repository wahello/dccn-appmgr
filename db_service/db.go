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
	// Get gets a app item by appmgr's id.
	Get(id string) (AppRecord, error)
	// GetAll gets all app related to user id.
	GetAll(userId string) ([]AppRecord, error)

	GetAllNamespace(userId string) ([]NamespaceRecord, error)
	// Cancel sets app status CANCEL
	Cancel(appId string) error

	CreateNamespace(namespace *common_proto.NameSpace, uid string) error
	// Create Creates a new dc item if not exits.
	Create(app *common_proto.App, uid string) error
	// Update updates dc item
	Update(appId string, update bson.M) error
	// UpdateApp updates dc item
	UpdateApp(appId string, app *common_proto.App) error
	// Close closes db connection
	UpdateNamespace(taskid string,namespace *common_proto.NameSpace) error

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
	ID           string
	Userid       string
	Name         string
	Image        string
	Datacenter   string
	Type         common_proto.AppType
	Replica      int32
	Datacenterid string  // mongodb name is low field
	Status       common_proto.AppStatus // 1 new 2 running 3. done 4 cancelling 5.canceled 6. updating 7. updateFailed
	Hidden       bool
	Schedule     string
	Last_modified_date uint64
	Creation_date uint64

}

type NamespaceRecord struct {
	NamespaceID string// short hash of uid+name+cluster_id
	Name string
	NamespaceUserID string 
	Cluster_ID string //id of cluster
	Cluster_Name string //name of cluster
	Creation_date uint64
	Cpu_limit float64
	Mem_limit uint64
	Storage_limit uint64
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

func (p *DB) collection(session *mgo.Session) *mgo.Collection {
	return session.DB(p.dbName).C(p.collectionName)
}

// Get gets app item by id.
func (p *DB) Get(appId string) (AppRecord, error) {
	session := p.session.Clone()
	defer session.Close()

	var app AppRecord
	err := p.collection(session).Find(bson.M{"id": appId}).One(&app)
	return app, err
}

func (p *DB) GetNamespace(NamespaceId string) (NamespaceRecord, error) {
	session := p.session.Clone()
	defer session.Close()	

	var namespace NamespaceRecord
	err := p.collection(session).Find(bson.M{"namespaceid": NamespaceId}).One(&namespace)
	return namespace, err
}

func (p *DB) GetAll(userId string) ([]AppRecord, error) {
	session := p.session.Clone()
	defer session.Close()

	var apps []AppRecord

	log.Printf("find apps with uid %s", userId)

	if err := p.collection(session).Find(bson.M{"userid": userId}).All(&apps); err != nil {
		return nil, err
	}
	return apps, nil
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

// GetByEventId gets app by event id.
func (p *DB) GetByEventId(eventId string) (*[]*common_proto.App, error) {
	session := p.session.Copy()
	defer session.Close()

	var apps []*common_proto.App
	if err := p.collection(session).Find(bson.M{"eventid": eventId}).One(&apps); err != nil {
		return nil, err
	}
	return &apps, nil
}

// Create creates a new app item if it not exists
func (p *DB) Create(app *common_proto.App, uid string) error {
	session := p.session.Copy()
	defer session.Close()

	appRecord := AppRecord{}
	appRecord.ID = app.Id
	appRecord.Userid = uid
	appRecord.Name = app.Name
	appRecord.Image = getAppImage(app)
	appRecord.Type = app.Type
	appRecord.Status = app.Status
	appRecord.Replica = app.Attributes.Replica
	if app.Type == common_proto.AppType_CRONJOB {
		appRecord.Schedule = app.GetTypeCronJob().Schedule
	}

	now := time.Now().Unix()
	appRecord.Last_modified_date = uint64(now)
	appRecord.Creation_date = uint64(now)
	return p.collection(session).Insert(appRecord)
}

func (p *DB) CreateNamespace(namespace *common_proto.NameSpace, uid string) error {
	session := p.session.Copy()
	defer session.Close()

	namespacerecord := NamespaceRecord{}
	namespacerecord.NamespaceID = namespace.Id
	namespacerecord.Name = namespace.Name
	namespacerecord.NamespaceUserID = uid
	namespacerecord.Cluster_ID = namespace.Cluster_ID
	namespacerecord.Cluster_Name = namespace.Cluster_Name
	namespacerecord.Creation_date = namespace.Creation_date
	namespacerecord.Cpu_limit = namespace.Cpu_limit
	namespacerecord.Mem_limit = namespace.Mem_limit
	namespacerecord.Storage_limit = namespace.Storage_limit
	return p.collection(session).Insert(namespacerecord)
}

func getAppImage(app *common_proto.App) string{
	if app.Type == common_proto.AppType_DEPLOYMENT {
		return app.GetTypeDeployment().Image
	}

	if app.Type == common_proto.AppType_JOB {
		return app.GetTypeJob().Image
	}

	if app.Type == common_proto.AppType_CRONJOB {
		return app.GetTypeCronJob().Image
	}

	return ""
}


// Update updates app item.
func (p *DB) Update(appId string, update bson.M) error {
	session := p.session.Copy()
	defer session.Close()

	return p.collection(session).Update(bson.M{"id": appId}, update)
}

func (p *DB) UpdateApp(appId string, app *common_proto.App) error {
	session := p.session.Copy()
	defer session.Close()

	fields := bson.M{}

	if len(app.Name) > 0 {
		fields["name"] = app.Name
	}

	if app.Attributes.Replica > 0 {
		fields["replica"] = app.Attributes.Replica
	}

	if app.Status > 0 {
		fields["status"] = app.Status
	}

	if app.Attributes.Hidden {
		fields["hidden"] = app.Attributes.Hidden
	}

	if app.Type == common_proto.AppType_CRONJOB {
		fields["schedule"] = app.GetTypeCronJob().Schedule
	}

	image := getAppImage(app)

	if len(image) > 0 {
		fields["image"] = image
	}

	if len(app.DataCenterName) > 0 {
		fields["datacenter"] = app.DataCenterName
	}

	now := time.Now().Unix()
	fields["Last_modified_date"] = now


	return p.collection(session).Update(bson.M{"id": appId}, bson.M{"$set": fields})


	//return p.collection(session).Update(bson.M{"id": appId}, app)
}

// Cancel cancel app, sets app status CANCEL
func (p *DB) Cancel(appId string) error {
	session := p.session.Copy()
	defer session.Close()

	now := time.Now().Unix()
	return p.collection(session).Update(bson.M{"id": appId}, bson.M{"$set": bson.M{"status": common_proto.AppStatus_CANCELLED, "Last_modified_date" : now}})
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
