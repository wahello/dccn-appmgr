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
		Collection: "app",
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

func mockApps() []*common_proto.App {
	return []*common_proto.App{
		&common_proto.App{
			Id:           "001",
			UserId:       1,
			Name:         "app01",
			Type:         "web",
			Image:        "nginx",
			Replica:      2,
			DataCenter:   "dc01",
			DataCenterId: 1,
		},
		&common_proto.App{
			Id:           "002",
			UserId:       1,
			Name:         "app02",
			Type:         "web",
			Image:        "nginx",
			Replica:      2,
			DataCenter:   "dc02",
			DataCenterId: 1,
		},
		&common_proto.App{
			Id:           "003",
			UserId:       2,
			Name:         "app01",
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

	apps := mockApps()
	for i, _ := range apps {
		if err = db.Create(apps[i]); err != nil {
			t.Fatal(err.Error())
		}
	}
}

func isEqual(origin, dst *common_proto.App) bool {
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

	apps := mockApps()
	for i := range apps {
		if err = db.Create(apps[i]); err != nil {
			t.Fatal(err.Error())
		}
	}

	for _, app := range apps {
		dbApp, err := db.Get(app.Id)
		if err != nil {
			t.Fatal(err.Error())
		}
		if !isEqual(dbApp, app) {
			t.Fatalf("want %#v\nbut %#v\n", app, dbApp)
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

	apps := mockApps()
	for i := range apps {
		if err = db.Create(apps[i]); err != nil {
			t.Fatal(err.Error())
		}
	}

	dbApps, err := db.GetAll(apps[0].UserId)
	if err != nil {
		t.Fatal(err.Error())
	}
	if len(*dbApps) != 2 {
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

	apps := mockApps()
	for i := range apps {
		if err = db.Create(apps[i]); err != nil {
			t.Fatal(err.Error())
		}
	}

	newApp := *apps[0]
	newApp.Name = "app003"
	if err = db.Update(newApp.Id, bson.M{"$set": bson.M{"name": newApp.Name}}); err != nil {
		t.Fatal(err.Error())
	}

	dbApp, err := db.Get(newApp.Id)
	if err != nil {
		t.Fatal(err.Error())
	}

	if dbApp.Name != newApp.Name {
		t.Fatal("Update not change")
	}

	t.Log("Update OK")
}
