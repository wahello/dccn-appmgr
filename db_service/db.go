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
	// UpdateApp updates dc item
	UpdateApp(appId string, app *common_proto.AppDeployment) error
	// Close closes db connection
	UpdateNamespace(namespaceId string, namespace *common_proto.Namespace) error

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
	ID          string
	Userid      string
	Name        string
	Namespace   common_proto.Namespace // mongodb name is low field
	Status      common_proto.AppStatus // 1 new 2 running 3. done 4 cancelling 5.canceled 6. updating 7. updateFailed
	Hidden      bool
	Attributes  common_proto.AppAttributes
	ChartDetail common_proto.ChartDetail
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

func (p *DB) GetAllApp(userId string) ([]AppRecord, error) {
	session := p.session.Clone()
	defer session.Close()

	var apps []AppRecord

	log.Printf("find apps with uid %s", userId)

	if err := p.collection(session, "app").Find(bson.M{"userid": userId}).All(&apps); err != nil {
		return nil, err
	}
	return apps, nil
}

// GetByEventId gets app by event id.
func (p *DB) GetByEventId(eventId string) (*[]*common_proto.App, error) {
	session := p.session.Copy()
	defer session.Close()

	var apps []*common_proto.App
	if err := p.collection(session, "app").Find(bson.M{"eventid": eventId}).One(&apps); err != nil {
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
	appRecord.Userid = uid
	appRecord.Name = appDeployment.Name
	appRecord.Status = appDeployment.Status
	appRecord.Namespace = *appDeployment.Namespace
	appRecord.ChartDetail = *appDeployment.ChartDetail
	now := time.Now().Unix()
	appRecord.Attributes.LastModifiedDate = uint64(now)
	appRecord.Attributes.CreationDate = uint64(now)
	return p.collection(session, "app").Insert(appRecord)
}

// Update updates app item.
func (p *DB) Update(collection string, id string, update bson.M) error {
	session := p.session.Copy()
	defer session.Close()

	return p.collection(session, collection).Update(bson.M{"id": id}, update)
}

func (p *DB) UpdateApp(appId string, app *common_proto.AppDeployment) error {
	session := p.session.Copy()
	defer session.Close()

	fields := bson.M{}

	if len(app.Name) > 0 {
		fields["name"] = app.Name
	}

	if app.Status > 0 {
		fields["status"] = app.Status
	}

	now := time.Now().Unix()
	fields["Last_modified_date"] = now

	return p.collection(session, "app").Update(bson.M{"id": appId}, bson.M{"$set": fields})

	//return p.collection(session).Update(bson.M{"id": appId}, app)
}

// Cancel cancel app, sets app status CANCEL
func (p *DB) CancelApp(appId string) error {
	session := p.session.Copy()
	defer session.Close()

	now := time.Now().Unix()
	return p.collection(session, "app").Update(bson.M{"id": appId}, bson.M{"$set": bson.M{"status": common_proto.AppStatus_APP_CANCELLED, "Last_modified_date": now}})
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
