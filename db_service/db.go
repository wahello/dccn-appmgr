package dbservice

import (
	"errors"
	"log"
	"time"

	dbcommon "github.com/Ankr-network/dccn-common/db"
	ankr_default "github.com/Ankr-network/dccn-common/protos"
	common_proto "github.com/Ankr-network/dccn-common/protos/common"
	"github.com/golang/protobuf/ptypes/timestamp"
	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
)

type DBService interface {
	// GetApp gets a app item by app's id.
	GetApp(id string) (AppRecord, error)
	// GetRunningApps gets running app related to team id.
	GetRunningApps(teamId string) ([]AppRecord, error)
	// GetAllApps get all app app related to team id
	GetAllApps(teamID string) ([]AppRecord, error)
	// GetAllAppsByNamespaceId gets all app related to namespace id.
	GetAllAppsByNamespaceId(namespaceId string) ([]AppRecord, error)
	// GetRunningAppsByNamespaceId gets running app related to namespace id.
	GetRunningAppsByNamespaceId(namespaceId string) ([]AppRecord, error)
	// GetRunningAppsByClusterID get running app related to cluster id
	GetRunningAppsByClusterID(clusterID string) ([]AppRecord, error)
	// CountRunningAppsByClusterID count running app related to cluster id
	CountRunningAppsByClusterID(clusterID string) (int, error)
	// CountRunningApps count all running apps
	CountRunningApps() (int, error)
	// GetRunningAppsByTeamIDAndClusterID gets all app related to user id in specific cluster.
	GetRunningAppsByTeamIDAndClusterID(teamId string, clusterId string) ([]AppRecord, error)
	// GetNamespace gets a namespace item by namespace's id.
	GetNamespace(namespaceId string) (NamespaceRecord, error)
	// GetRunningNamespacesByClusterId gets a namespace item by cluster's id.
	GetRunningNamespacesByClusterId(clusterId string) ([]NamespaceRecord, error)
	// CountRunningNamespaces count running namespace related to cluster id
	CountRunningNamespacesByClusterID(clusterID string) (int, error)
	// GetRunningNamespaces gets running namespace items by teamID
	GetRunningNamespaces(teamId string) ([]NamespaceRecord, error)
	// CountRunningNamespaces count all running namespace
	CountRunningNamespaces() (int, error)
	// GetAllNamespaces get all namespace items by teamID
	GetAllNamespaces(teamID string) ([]NamespaceRecord, error)
	// GetRunningNamespacesByTeamIDAndClusterID get running namespace related to teamId & clusterId
	GetRunningNamespacesByTeamIDAndClusterID(teamID string, clusterId string) ([]NamespaceRecord, error)
	// CancelApp sets app status CANCEL
	CancelApp(appId string) error
	// CancelNamespace sets namespace status CANCEL
	CancelNamespace(namespaceId string) error
	// CreateNamespace create new namespace
	CreateNamespace(namespace *common_proto.Namespace, teamId string, creator string) error
	// Create Creates a new app item if not exits.
	CreateApp(appDeployment *common_proto.AppDeployment, teamId string, creator string) error
	// Update updates collection item
	Update(collectId string, id string, update bson.M) error
	// UpdateMany update collection all item
	UpdateMany(collection string, filter, update bson.M) (*mgo.ChangeInfo, error)
	// UpdateApp updates app item
	UpdateApp(app *common_proto.AppDeployment) error
	// UpdateNamespace update namespace item
	UpdateNamespace(namespace *common_proto.Namespace) error
	// UpdateByHeartbeatMetrics update app & namespace by dc heartbeat metrics
	UpdateByHeartbeatMetrics(clusterID string, metrics *common_proto.DCHeartbeatReport_Metrics)
	// Create a new cluster connection
	CreateClusterConnection(clusterID string, clusterStatus common_proto.DCStatus, metrics *common_proto.DCHeartbeatReport_Metrics) error
	// GetClusterConnection gets a cluster connection item by cluster's id.
	GetClusterConnection(clusterID string) (ClusterConnectionRecord, error)
	// GetAvailableClusterConnections count available cluster
	GetAvailableClusterConnections() ([]ClusterConnectionRecord, error)
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

func (p *DB) CountRunningAppsByClusterID(clusterID string) (int, error) {
	nss, err := p.GetRunningNamespacesByClusterId(clusterID)
	if err != nil {
		return 0, err
	}

	nsids := make([]string, len(nss))
	for i := range nss {
		nsids[i] = nss[i].ID
	}

	session := p.session.Clone()
	defer session.Close()

	count, err := p.collection(session, "app").Find(bson.M{"namespaceid": bson.M{"$in": nsids}, "status": common_proto.AppStatus_APP_RUNNING}).Count()
	if err != nil {
		return 0, errors.New(ankr_default.DbError + err.Error())
	}

	return count, nil
}

func (p *DB) CountRunningApps() (int, error) {
	session := p.session.Clone()
	defer session.Close()

	count, err := p.collection(session, "app").Find(bson.M{"status": common_proto.AppStatus_APP_RUNNING}).Count()
	if err != nil {
		return 0, errors.New(ankr_default.DbError + err.Error())
	}

	return count, nil
}

func (p *DB) CountRunningNamespacesByClusterID(clusterID string) (int, error) {
	session := p.session.Clone()
	defer session.Close()

	count, err := p.collection(session, "namespace").Find(bson.M{"clusterid": clusterID, "status": common_proto.NamespaceStatus_NS_RUNNING}).Count()
	if err != nil {
		return 0, errors.New(ankr_default.DbError + err.Error())
	}

	return count, nil
}

func (p *DB) CountRunningNamespaces() (int, error) {
	session := p.session.Clone()
	defer session.Close()

	count, err := p.collection(session, "namespace").Find(bson.M{"clusterid": bson.M{"$ne": ""}, "status": common_proto.NamespaceStatus_NS_RUNNING}).Count()
	if err != nil {
		return 0, errors.New(ankr_default.DbError + err.Error())
	}

	return count, nil
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

func (p *DB) GetRunningAppsByTeamIDAndClusterID(teamId string, clusterId string) ([]AppRecord, error) {
	session := p.session.Clone()
	defer session.Close()

	var apps []AppRecord

	log.Printf("GetRunningAppsByTeamIDAndClusterID with teamID %s clusterid %s", teamId, clusterId)
	if err := p.collection(session, "app").Find(bson.M{"teamid": teamId, "status": common_proto.AppStatus_APP_RUNNING}).All(&apps); err != nil {
		return nil, errors.New(ankr_default.DbError + err.Error())
	}

	if len(clusterId) == 0 {
		return apps, nil
	}

	nss, err := p.GetRunningNamespacesByClusterId(clusterId)
	if err != nil {
		return nil, errors.New(ankr_default.DbError + err.Error())
	}

	nsMap := map[string]struct{}{}
	for _, ns := range nss {
		nsMap[ns.ID] = struct{}{}
	}

	var res []AppRecord

	for _, app := range apps {
		if _, ok := nsMap[app.NamespaceID]; ok {
			res = append(res, app)
		}
	}

	return res, nil
}

func (p *DB) GetRunningApps(teamId string) ([]AppRecord, error) {
	session := p.session.Clone()
	defer session.Close()

	var apps []AppRecord

	if err := p.collection(session, "app").Find(bson.M{"teamid": teamId, "status": common_proto.AppStatus_APP_RUNNING}).All(&apps); err != nil {
		return nil, errors.New(ankr_default.DbError + err.Error())
	}
	return apps, nil
}

func (p *DB) GetAllApps(teamId string) ([]AppRecord, error) {
	session := p.session.Clone()
	defer session.Close()

	var apps []AppRecord

	if err := p.collection(session, "app").Find(bson.M{"teamid": teamId}).All(&apps); err != nil {
		return nil, errors.New(ankr_default.DbError + err.Error())
	}
	return apps, nil
}

func (p *DB) GetAllAppsByNamespaceId(namespaceId string) ([]AppRecord, error) {
	session := p.session.Clone()
	defer session.Close()

	var apps []AppRecord

	log.Printf("find apps with namespace id %s", namespaceId)

	if err := p.collection(session, "app").Find(bson.M{"namespaceid": namespaceId}).All(&apps); err != nil {
		return nil, errors.New(ankr_default.DbError + err.Error())
	}
	return apps, nil
}

func (p *DB) GetRunningAppsByNamespaceId(namespaceId string) ([]AppRecord, error) {
	session := p.session.Clone()
	defer session.Close()

	var apps []AppRecord

	if err := p.collection(session, "app").Find(bson.M{"namespaceid": namespaceId, "status": common_proto.AppStatus_APP_RUNNING}).All(&apps); err != nil {
		return nil, err
	}
	return apps, nil
}

// getAppsByEvent gets app by event id.
func (p *DB) getAppsByEvent(event string) (*[]*common_proto.App, error) {
	session := p.session.Copy()
	defer session.Close()

	var apps []*common_proto.App
	if err := p.collection(session, "app").Find(bson.M{"event": event}).One(&apps); err != nil {
		return nil, errors.New(ankr_default.DbError + err.Error())
	}
	return &apps, nil
}

// CreateApp creates a new app deployment item if it not exists
func (p *DB) CreateApp(appDeployment *common_proto.AppDeployment, teamId string, creator string) error {
	session := p.session.Copy()
	defer session.Close()

	appRecord := AppRecord{}
	appRecord.ID = appDeployment.AppId
	appRecord.TeamID = teamId
	appRecord.Creator = creator
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

func (p *DB) UpdateMany(collection string, filter, update bson.M) (*mgo.ChangeInfo, error) {
	session := p.session.Copy()
	defer session.Close()

	changeInfo, err := p.collection(session, collection).UpdateAll(filter, update)
	if err != nil {
		return nil, errors.New(ankr_default.DbError + err.Error())
	}

	return changeInfo, nil
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

func (p *DB) GetRunningNamespaces(teamId string) ([]NamespaceRecord, error) {
	session := p.session.Clone()
	defer session.Close()

	var namespaces []NamespaceRecord

	log.Printf("find apps with teamId %s", teamId)

	if err := p.collection(session, "namespace").Find(bson.M{"teamid": teamId, "status": common_proto.NamespaceStatus_NS_RUNNING}).All(&namespaces); err != nil {
		return nil, errors.New(ankr_default.DbError + err.Error())
	}
	return namespaces, nil
}

func (p *DB) GetAllNamespaces(teamId string) ([]NamespaceRecord, error) {
	session := p.session.Clone()
	defer session.Close()

	var namespaces []NamespaceRecord

	log.Printf("find apps with teamId %s", teamId)

	if err := p.collection(session, "namespace").Find(bson.M{"teamid": teamId}).All(&namespaces); err != nil {
		return nil, errors.New(ankr_default.DbError + err.Error())
	}
	return namespaces, nil
}

func (p *DB) CreateNamespace(namespace *common_proto.Namespace, teamId string, creator string) error {
	session := p.session.Copy()
	defer session.Close()

	namespacerecord := NamespaceRecord{}
	namespacerecord.ID = namespace.NsId
	namespacerecord.Name = namespace.NsName
	namespacerecord.TeamID = teamId
	namespacerecord.Creator = creator
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

func (p *DB) UpdateByHeartbeatMetrics(clusterID string, metrics *common_proto.DCHeartbeatReport_Metrics) {
	if metrics == nil {
		log.Printf("metrics is nil, skip")
		return
	}

	session := p.session.Copy()
	defer session.Close()

	log.Printf("UpdateByHeartbeatMetrics %+v", metrics)

	for nsID, r := range metrics.NsUsed {
		log.Printf("mark ns %s running and update usage %+v", nsID, r)
		if err := p.collection(session, "namespace").Update(bson.M{
			"id": nsID,
			"status": bson.M{
				"$in": []common_proto.NamespaceStatus{common_proto.NamespaceStatus_NS_FAILED, common_proto.NamespaceStatus_NS_UNAVAILABLE, common_proto.NamespaceStatus_NS_RUNNING},
			},
		}, bson.M{
			"$set": bson.M{
				"cpuusage":     r.CPU,
				"memusage":     r.Memory,
				"storageusage": r.Storage,
				"status":       common_proto.NamespaceStatus_NS_RUNNING,
			},
		}); err != nil {
			log.Printf("update ns %s by metrics %+v error: %+v", nsID, r, err)
		} else {
			changeInfo, err := p.collection(session, "app").UpdateAll(bson.M{
				"namespaceid": nsID,
				"status": bson.M{
					"$in": []common_proto.AppStatus{
						common_proto.AppStatus_APP_UNAVAILABLE,
						common_proto.AppStatus_APP_FAILED,
					},
				},
			}, bson.M{
				"$set": bson.M{
					"status": common_proto.AppStatus_APP_RUNNING,
				},
			})
			if err != nil {
				log.Printf("UpdateAll app change unavailable or failed app to available error: %v", err)
			}
			log.Printf("mark unavailable or failed apps available of namespace %s, changeInfo %+v", nsID, changeInfo)
		}
	}

	nss, err := p.GetRunningNamespacesByClusterId(clusterID)
	if err != nil {
		log.Printf("get cluster %s namespaces error: %v", clusterID, err)
	}

	dbNsIDMap := make(map[string]struct{})
	for _, ns := range nss {
		dbNsIDMap[ns.ID] = struct{}{}
	}

	for nsID := range metrics.NsUsed {
		if _, ok := dbNsIDMap[nsID]; !ok {
			log.Printf("ns %s not in db", nsID)
		}
	}

	for _, ns := range nss {
		if _, ok := metrics.NsUsed[ns.ID]; !ok {
			p.markNamespaceUnavailable(session, ns.ID)
			apps, err := p.GetRunningAppsByNamespaceId(ns.ID)
			if err != nil {
				log.Printf("get namespace %s all app error: %+v", ns.ID, err)
			}
			for _, app := range apps {
				p.markAppUnavailable(session, app.ID)
			}
		}
	}
}

func (p *DB) markNamespaceUnavailable(session *mgo.Session, nsID string) {
	log.Printf("mark namespace %s unavailable", nsID)
	markThreshold := time.Now().Unix() - 60
	if err := p.collection(session, "namespace").Update(bson.M{
		"id":     nsID,
		"status": common_proto.NamespaceStatus_NS_RUNNING,
		"lastmodifieddate.seconds": bson.M{
			"$lt": markThreshold,
		},
	}, bson.M{"$set": bson.M{
		"status": common_proto.NamespaceStatus_NS_UNAVAILABLE,
	}}); err != nil {
		log.Printf("mark namespace %s unavailabe error: %+v", nsID, err)
	}
}

func (p *DB) markAppUnavailable(session *mgo.Session, appID string) {
	log.Printf("mark app %s unavailable", appID)
	markThreshold := time.Now().Unix() - 60
	if err := p.collection(session, "app").Update(bson.M{
		"id":     appID,
		"status": common_proto.AppStatus_APP_RUNNING,
		"lastmodifieddate.seconds": bson.M{
			"$lt": markThreshold,
		},
	}, bson.M{"$set": bson.M{
		"status": common_proto.AppStatus_APP_UNAVAILABLE,
	}}); err != nil {
		log.Printf("mark app %s unavailabe error: %+v", appID, err)
	}
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

func (p *DB) GetClusterConnection(clusterID string) (ClusterConnectionRecord, error) {
	session := p.session.Clone()
	defer session.Close()

	var clusterConnection ClusterConnectionRecord
	err := p.collection(session, "clusterconnection").Find(bson.M{"id": clusterID}).One(&clusterConnection)

	return clusterConnection, err
}

func (p *DB) GetAvailableClusterConnections() ([]ClusterConnectionRecord, error) {
	session := p.session.Clone()
	defer session.Close()

	var connections []ClusterConnectionRecord
	if err := p.collection(session, "clusterconnection").Find(bson.M{"status": common_proto.DCStatus_AVAILABLE}).All(&connections); err != nil {
		return nil, errors.New(ankr_default.DbError + err.Error())
	}

	return connections, nil
}

func (p *DB) CreateClusterConnection(clusterID string, clusterStatus common_proto.DCStatus, metrics *common_proto.DCHeartbeatReport_Metrics) error {
	session := p.session.Copy()
	defer session.Close()

	now := time.Now().Unix()
	clusterConnection := &ClusterConnectionRecord{
		ID:               clusterID,
		Status:           clusterStatus,
		Metrics:          metrics,
		LastModifiedDate: &timestamp.Timestamp{Seconds: now},
		CreationDate:     &timestamp.Timestamp{Seconds: now},
	}

	return p.collection(session, "clusterconnection").Insert(clusterConnection)
}

func (p *DB) GetRunningAppsByClusterID(clusterID string) ([]AppRecord, error) {
	rsp := make([]AppRecord, 0)
	nss, err := p.GetRunningNamespacesByClusterId(clusterID)
	if err != nil {
		return rsp, err
	}

	nsids := make([]string, len(nss))
	for i := range nss {
		nsids[i] = nss[i].ID
	}

	session := p.session.Clone()
	defer session.Close()

	var apps []AppRecord
	if err := p.collection(session, "app").Find(bson.M{"namespaceid": bson.M{"$in": nsids}, "status": common_proto.AppStatus_APP_RUNNING}).All(&apps); err != nil {
		return rsp, errors.New(ankr_default.DbError + err.Error())
	}

	return apps, nil
}

func (p *DB) GetRunningNamespacesByClusterId(clusterId string) ([]NamespaceRecord, error) {
	session := p.session.Clone()
	defer session.Close()

	var nss []NamespaceRecord
	if err := p.collection(session, "namespace").Find(bson.M{"clusterid": clusterId, "status": common_proto.NamespaceStatus_NS_RUNNING}).All(&nss); err != nil {
		log.Printf("get cluster %s runing namespace error: %v", clusterId, err)
		return nss, errors.New(ankr_default.DbError + err.Error())
	}

	return nss, nil
}

func (p *DB) GetRunningNamespacesByTeamIDAndClusterID(teamID string, clusterId string) ([]NamespaceRecord, error) {
	session := p.session.Clone()
	defer session.Close()

	filter := bson.M{
		"teamid": teamID,
		"status": common_proto.NamespaceStatus_NS_RUNNING,
	}

	if len(clusterId) > 0 {
		filter["clusterid"] = clusterId
	}

	var nss []NamespaceRecord
	if err := p.collection(session, "namespace").Find(filter).All(&nss); err != nil {
		log.Printf("get cluster %s runing namespace error: %v", clusterId, err)
		return nss, errors.New(ankr_default.DbError + err.Error())
	}

	return nss, nil
}
