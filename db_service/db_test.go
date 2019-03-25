package dbservice

import (
	"reflect"
	"testing"

	dbcommon "github.com/Ankr-network/dccn-common/db"
	common_proto "github.com/Ankr-network/dccn-common/protos/common"
	"gopkg.in/mgo.v2/bson"
)

func mockDB() (DBService, error) {
	conf := dbcommon.Config{
		DB:         "test",
		Collection: "task",
		Host:       "127.0.0.1:27017",
		Timeout:    15,
		PoolLimit:  15,
	}

	return New(conf)
}

func TestDB_New(t *testing.T) {
	db, err := mockDB()
	if err != nil {
		t.Fatal(err.Error())
	}
	db.Close()
}

func mockTasks() []*common_proto.Task {
	return []*common_proto.Task{
		&common_proto.Task{
			Id:           "001",
			UserId:       1,
			Name:         "task01",
			Type:         "web",
			Image:        "nginx",
			Replica:      2,
			DataCenter:   "dc01",
			DataCenterId: 1,
		},
		&common_proto.Task{
			Id:           "002",
			UserId:       1,
			Name:         "task02",
			Type:         "web",
			Image:        "nginx",
			Replica:      2,
			DataCenter:   "dc02",
			DataCenterId: 1,
		},
		&common_proto.Task{
			Id:           "003",
			UserId:       2,
			Name:         "task01",
			Type:         "web",
			Image:        "nginx",
			Replica:      2,
			DataCenter:   "dc01",
			DataCenterId: 1,
		},
	}
}

func TestDB_Create(t *testing.T) {
	db, err := mockDB()
	if err != nil {
		t.Fatal(err.Error())
	}
	defer db.Close()
	defer db.dropCollection()

	tasks := mockTasks()
	for i, _ := range tasks {
		if err = db.Create(tasks[i]); err != nil {
			t.Fatal(err.Error())
		}
	}
}

func isEqual(origin, dst *common_proto.Task) bool {
	ok := origin.Id == dst.Id &&
		origin.UserId == dst.UserId &&
		origin.Type == dst.Type &&
		origin.Name == dst.Name &&
		origin.Image == dst.Image &&
		origin.Replica == dst.Replica &&
		origin.DataCenter == dst.DataCenter &&
		origin.DataCenterId == dst.DataCenterId &&
		origin.Status == dst.Status &&
		origin.UniqueName == dst.UniqueName &&
		origin.Url == dst.Url &&
		origin.Hidden == dst.Hidden &&
		origin.Uptime == dst.Uptime &&
		origin.CreationDate == dst.CreationDate
	if origin.Extra != nil && dst.Extra != nil {
		ok = ok && reflect.DeepEqual(origin.Extra, dst.Extra)
	}
	return ok
}

func TestDB_Get(t *testing.T) {
	db, err := mockDB()
	if err != nil {
		t.Fatal(err.Error())
	}
	defer db.Close()
	defer db.dropCollection()

	tasks := mockTasks()
	for i := range tasks {
		if err = db.Create(tasks[i]); err != nil {
			t.Fatal(err.Error())
		}
	}

	for _, task := range tasks {
		dbTask, err := db.Get(task.Id)
		if err != nil {
			t.Fatal(err.Error())
		}
		if !isEqual(dbTask, task) {
			t.Fatalf("want %#v\nbut %#v\n", task, dbTask)
		}
	}

	t.Logf("Get Ok\n")
}

func TestDB_GetAll(t *testing.T) {
	db, err := mockDB()
	if err != nil {
		t.Fatal(err.Error())
	}
	defer db.Close()
	defer db.dropCollection()

	tasks := mockTasks()
	for i := range tasks {
		if err = db.Create(tasks[i]); err != nil {
			t.Fatal(err.Error())
		}
	}

	dbTasks, err := db.GetAll(tasks[0].UserId)
	if err != nil {
		t.Fatal(err.Error())
	}
	if len(*dbTasks) != 2 {
		t.Fatal("Get All Error")
	}

	t.Logf("GetAll Ok\n")
}

func TestDB_Update(t *testing.T) {
	db, err := mockDB()
	if err != nil {
		t.Fatal(err.Error())
	}
	defer db.Close()
	defer db.dropCollection()

	tasks := mockTasks()
	for i := range tasks {
		if err = db.Create(tasks[i]); err != nil {
			t.Fatal(err.Error())
		}
	}

	newTask := *tasks[0]
	newTask.Name = "task003"
	if err = db.Update(newTask.Id, bson.M{"$set": bson.M{"name": newTask.Name}}); err != nil {
		t.Fatal(err.Error())
	}

	dbTask, err := db.Get(newTask.Id)
	if err != nil {
		t.Fatal(err.Error())
	}

	if dbTask.Name != newTask.Name {
		t.Fatal("Update not change")
	}

	t.Log("Update OK")
}
