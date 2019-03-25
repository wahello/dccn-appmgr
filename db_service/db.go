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
	// Get gets a task item by taskmgr's id.
	Get(id string) (TaskRecord, error)
	// GetAll gets all task related to user id.
	GetAll(userId string) ([]TaskRecord, error)
	// Cancel sets task status CANCEL
	Cancel(taskId string) error
	// Create Creates a new dc item if not exits.
	Create(task *common_proto.Task, uid string) error
	// Update updates dc item
	Update(taskId string, update bson.M) error
	// UpdateTask updates dc item
	UpdateTask(taskId string, task *common_proto.Task) error
	// Close closes db connection
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

type TaskRecord struct {
	ID           string
	Userid       string
	Name         string
	Image        string
	Datacenter   string
	Type         common_proto.TaskType
	Replica      int32
	Datacenterid string  // mongodb name is low field
	Status       common_proto.TaskStatus // 1 new 2 running 3. done 4 cancelling 5.canceled 6. updating 7. updateFailed
	Hidden       bool
	Schedule     string
	Last_modified_date uint64
	Creation_date uint64

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

// Get gets task item by id.
func (p *DB) Get(taskId string) (TaskRecord, error) {
	session := p.session.Clone()
	defer session.Close()

	var task TaskRecord
	err := p.collection(session).Find(bson.M{"id": taskId}).One(&task)
	return task, err
}

func (p *DB) GetAll(userId string) ([]TaskRecord, error) {
	session := p.session.Clone()
	defer session.Close()

	var tasks []TaskRecord

	log.Printf("find tasks with uid %s", userId)

	if err := p.collection(session).Find(bson.M{"userid": userId}).All(&tasks); err != nil {
		return nil, err
	}
	return tasks, nil
}

// GetByEventId gets task by event id.
func (p *DB) GetByEventId(eventId string) (*[]*common_proto.Task, error) {
	session := p.session.Copy()
	defer session.Close()

	var tasks []*common_proto.Task
	if err := p.collection(session).Find(bson.M{"eventid": eventId}).One(&tasks); err != nil {
		return nil, err
	}
	return &tasks, nil
}

// Create creates a new task item if it not exists
func (p *DB) Create(task *common_proto.Task, uid string) error {
	session := p.session.Copy()
	defer session.Close()

	taskRecord := TaskRecord{}
	taskRecord.ID = task.Id
	taskRecord.Userid = uid
	taskRecord.Name = task.Name
	taskRecord.Image = getTaskImage(task)
	taskRecord.Type = task.Type
	taskRecord.Status = task.Status
	taskRecord.Replica = task.Attributes.Replica
	if task.Type == common_proto.TaskType_CRONJOB {
		taskRecord.Schedule = task.GetTypeCronJob().Schedule
	}

	now := time.Now().Unix()
	taskRecord.Last_modified_date = uint64(now)
	taskRecord.Creation_date = uint64(now)
	return p.collection(session).Insert(taskRecord)
}

func getTaskImage(task *common_proto.Task) string{
	if task.Type == common_proto.TaskType_DEPLOYMENT {
		return task.GetTypeDeployment().Image
	}

	if task.Type == common_proto.TaskType_JOB {
		return task.GetTypeJob().Image
	}

	if task.Type == common_proto.TaskType_CRONJOB {
		return task.GetTypeCronJob().Image
	}

	return ""
}


// Update updates task item.
func (p *DB) Update(taskId string, update bson.M) error {
	session := p.session.Copy()
	defer session.Close()

	return p.collection(session).Update(bson.M{"id": taskId}, update)
}

func (p *DB) UpdateTask(taskId string, task *common_proto.Task) error {
	session := p.session.Copy()
	defer session.Close()

	fields := bson.M{}

	if len(task.Name) > 0 {
		fields["name"] = task.Name
	}

	if task.Attributes.Replica > 0 {
		fields["replica"] = task.Attributes.Replica
	}

	if task.Status > 0 {
		fields["status"] = task.Status
	}

	if task.Attributes.Hidden {
		fields["hidden"] = task.Attributes.Hidden
	}

	if task.Type == common_proto.TaskType_CRONJOB {
		fields["schedule"] = task.GetTypeCronJob().Schedule
	}

	image := getTaskImage(task)

	if len(image) > 0 {
		fields["image"] = image
	}

	if len(task.DataCenterName) > 0 {
		fields["datacenter"] = task.DataCenterName
	}

	now := time.Now().Unix()
	fields["Last_modified_date"] = now


	return p.collection(session).Update(bson.M{"id": taskId}, bson.M{"$set": fields})


	//return p.collection(session).Update(bson.M{"id": taskId}, task)
}

// Cancel cancel task, sets task status CANCEL
func (p *DB) Cancel(taskId string) error {
	session := p.session.Copy()
	defer session.Close()

	now := time.Now().Unix()
	return p.collection(session).Update(bson.M{"id": taskId}, bson.M{"$set": bson.M{"status": common_proto.TaskStatus_CANCELLED, "Last_modified_date" : now}})
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
