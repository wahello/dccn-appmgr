package handler

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/Ankr-network/dccn-common/protos"
	"github.com/Ankr-network/dccn-common/protos/common"
	"github.com/Ankr-network/dccn-common/protos/appmgr/v1/micro"
	db "github.com/Ankr-network/dccn-appmgr/db_service"
	"github.com/google/uuid"
	"github.com/gorhill/cronexpr"
	micro "github.com/micro/go-micro"
	"github.com/micro/go-micro/metadata"
	"gopkg.in/mgo.v2/bson"
	"k8s.io/helm/pkg/chartutil"
)

type AppMgrHandler struct {
	db         db.DBService
	deployApp micro.Publisher
}

func New(db db.DBService, deployApp micro.Publisher) *AppMgrHandler {

	return &AppMgrHandler{
		db:         db,
		deployApp: deployApp,
	}
}

type Token struct {
	Exp int64
	Jti string
	Iss string
}

func getUserID(ctx context.Context) string {
	meta, ok := metadata.FromContext(ctx)
	// Note this is now uppercase (not entirely sure why this is...)
	var token string
	if ok {
		token = meta["token"]
	}

	parts := strings.Split(token, ".")

	decoded, err := base64.StdEncoding.DecodeString(parts[1])
	if err != nil {
		fmt.Println("decode error:", err)

	}
	fmt.Println(string(decoded))

	var dat Token

	if err := json.Unmarshal(decoded, &dat); err != nil {
		panic(err)
	}

	return string(dat.Jti)
}

func (p *AppMgrHandler) CreateChart(ctx context.Context, req *appmgr.CreateChartRequest, rsp *appmgr.CreateChartResponse) error {
	uid := getUserID(ctx)

	path, err := filepath.Abs("chart_repo/" + req.App.Type.String())
	if err != nil {
		log.Printf("cannot get chart folder path...\n %s \n", err.Error())
		return errors.New("internal error: cannot get chart folder path")
	}

	customChart, err := chartutil.LoadDir(path)
	if err != nil {
		log.Printf("cannot load chart from folder %s ...\n %s \n", path, err.Error())
		return errors.New("internal error: cannot load chart from folder")
	}

	//customChart.Values.Values["replicaCount"] = &chart.Value{Value: fmt.Sprint(req.App.Attributes.Replica)}
	customChart.Metadata.Version = req.App.ChartVer
	customChart.Metadata.Name = req.App.ChartName

	dest, err := os.Getwd()
	if err != nil {
		log.Printf("cannot get chart outdir")
		return errors.New("internal error: cannot get chart outdir")
	}
	log.Printf("save to outdir: %s\n", dest)
	name, err := chartutil.Save(customChart, dest)
	if err == nil {
		log.Printf("Successfully packaged chart and saved it to: %s\n", name)
	} else {
		log.Printf("Failed to save: %s", err)
		return errors.New("internal error: Failed to save chart to outdir")
	}

	file, err := os.Open(name)
	if err != nil {
		log.Printf("cannot open chart zip file")
		return errors.New("internal error: cannot open chart zip file")
	}

	chartReq, err := http.NewRequest("POST", "http://chart-dev.dccn.ankr.network:8080/api/user/"+uid+"/charts", file)
	if err != nil {
		log.Printf("cannot open chart zip file")
		return errors.New("internal error: cannot open chart zip file")
	}

	chartRes, err := http.DefaultClient.Do(chartReq)

	message, _ := ioutil.ReadAll(chartRes.Body)
	defer chartRes.Body.Close()

	log.Printf(string(message))

}

func (p *AppMgrHandler) CreateApp(ctx context.Context, req *appmgr.CreateAppRequest, rsp *appmgr.CreateAppResponse) error {
	uid := getUserID(ctx)
	log.Println("app manager service CreateApp")

	if req.App.Attributes.Replica < 0 || req.App.Attributes.Replica >= 100 {
		log.Println(ankr_default.ErrReplicaTooMany)
		return ankr_default.ErrReplicaTooMany
	}

	log.Printf("CreateApp app %+v", req)

	if req.App.Attributes.Replica == 0 {
		req.App.Attributes.Replica = 1
	}

	if req.App.Type == common_proto.AppType_CRONJOB { // check schudule filed
		_, err := cronexpr.Parse(req.App.GetTypeCronJob().Schedule)
		if err != nil {
			log.Printf("check crobjob scheducle fomat error %s \n", err.Error())
			return ankr_default.ErrCronJobScheduleFormat
		}

	}

	req.App.Status = common_proto.AppStatus_STARTING
	req.App.Id = uuid.New().String()
	req.App.Uid = uid
	rsp.AppId = req.App.Id

	event := common_proto.DCStream{
		OpType:    common_proto.DCOperation_TASK_CREATE,
		OpPayload: &common_proto.DCStream_App{App: req.App},
	}

	if err := p.deployApp.Publish(context.Background(), &event); err != nil {
		log.Println(ankr_default.ErrPublish)
		return ankr_default.ErrPublish
	} else {
		log.Println("app manager service send CreateApp MQ message to dc manager service (api)")
	}

	if err := p.db.Create(req.App, uid); err != nil {
		log.Println(err.Error())
		return err
	}

	return nil
}

// Must return nil for gRPC handler
func (p *AppMgrHandler) CancelApp(ctx context.Context, req *appmgr.AppID, rsp *common_proto.Empty) error {
	userId := getUserID(ctx)
	log.Println("Debug into CancelApp")
	if err := checkId(userId, req.AppId); err != nil {
		log.Println(err.Error())
		return err
	}
	app, err := p.checkOwner(userId, req.AppId)
	if err != nil {
		log.Println(err.Error())
		return err
	}

	if app.Status == common_proto.AppStatus_CANCELLED {
		return ankr_default.ErrCanceledTwice
	}

	event := common_proto.DCStream{
		OpType:    common_proto.DCOperation_TASK_CANCEL,
		OpPayload: &common_proto.DCStream_App{App: app},
	}

	if err := p.deployApp.Publish(context.Background(), &event); err != nil {
		log.Println(err.Error())
		return err
	}

	if err := p.db.Update(app.Id, bson.M{"$set": bson.M{"status": common_proto.AppStatus_CANCELLED}}); err != nil {
		log.Println(err.Error())
		return err
	}

	return nil
}

func convertToAppMessage(app db.AppRecord) common_proto.App {
	message := common_proto.App{}
	message.Id = app.ID
	message.Name = app.Name
	message.Type = app.Type
	message.Status = app.Status
	message.DataCenterName = app.Datacenter
	message.Attributes = &common_proto.AppAttributes{}
	message.Attributes.Replica = app.Replica
	message.Attributes.LastModifiedDate = app.Last_modified_date
	message.Attributes.CreationDate = app.Creation_date

	//deployMessage := common_proto.AppTypeDeployment{Image : app.Image}
	if app.Type == common_proto.AppType_DEPLOYMENT {
		t := common_proto.App_TypeDeployment{TypeDeployment: &common_proto.AppTypeDeployment{Image: app.Image}}
		message.TypeData = &t
	}

	if app.Type == common_proto.AppType_JOB {
		t := common_proto.App_TypeJob{TypeJob: &common_proto.AppTypeJob{Image: app.Image}}
		message.TypeData = &t
	}

	if app.Type == common_proto.AppType_CRONJOB {
		t := common_proto.App_TypeCronJob{TypeCronJob: &common_proto.AppTypeCronJob{Image: app.Image, Schedule: app.Schedule}}
		message.TypeData = &t
	}

	return message

}

func (p *AppMgrHandler) AppList(ctx context.Context, req *appmgr.AppListRequest, rsp *appmgr.AppListResponse) error {
	userId := getUserID(ctx)
	log.Println("app service into AppList")

	apps, err := p.db.GetAll(userId)
	log.Printf(">>>>>>appMessage  %+v \n", apps)
	if err != nil {
		log.Println(err.Error())
		return err
	}

	appsWithoutHidden := make([]*common_proto.App, 0)

	for i := 0; i < len(apps); i++ {
		if apps[i].Hidden != true {
			appMessage := convertToAppMessage(apps[i])
			log.Printf("appMessage  %+v \n", appMessage)
			appsWithoutHidden = append(appsWithoutHidden, &appMessage)
		}
	}

	rsp.Apps = appsWithoutHidden

	return nil
}

func (p *AppMgrHandler) UpdateApp(ctx context.Context, req *appmgr.UpdateAppRequest, rsp *common_proto.Empty) error {
	userId := getUserID(ctx)

	if err := checkId(userId, req.App.Id); err != nil {
		log.Println(err.Error())
		return err
	}

	app, err := p.checkOwner(userId, req.App.Id)
	if err != nil {
		log.Println(err.Error())
		return err
	}

	req.App.Name = strings.ToLower(req.App.Name)

	if req.App.Attributes.Replica == 0 {
		req.App.Attributes.Replica = app.Attributes.Replica
	}

	if req.App.Attributes.Replica < 0 || req.App.Attributes.Replica >= 100 {
		log.Println(ankr_default.ErrReplicaTooMany.Error())
		return ankr_default.ErrReplicaTooMany
	}

	if app.Status == common_proto.AppStatus_CANCELLED ||
		app.Status == common_proto.AppStatus_DONE {
		log.Println(ankr_default.ErrAppStatusCanNotUpdate.Error())
		return ankr_default.ErrAppStatusCanNotUpdate
	}

	app.ChartRepo = req.App.ChartRepo
	app.ChartName = req.App.ChartName
	app.ChartVer = req.App.ChartVer
	app.Uid = userId

	event := common_proto.DCStream{
		OpType:    common_proto.DCOperation_TASK_UPDATE,
		OpPayload: &common_proto.DCStream_App{App: app},
	}

	if err := p.deployApp.Publish(context.Background(), &event); err != nil {
		log.Println(err.Error())
		return err
	}
	// TODO: wait deamon notify
	req.App.Status = common_proto.AppStatus_UPDATING
	if err := p.db.UpdateApp(req.App.Id, req.App); err != nil {
		log.Println(err.Error())
		return err
	}
	return nil
}

func (p *AppMgrHandler) AppOverview(ctx context.Context, req *common_proto.Empty, rsp *appmgr.AppOverviewResponse) error {
	log.Printf("AppOverview in app manager service\n")
	//rsp = &appmgr.AppOverviewResponse{}
	userId := getUserID(ctx)
	apps, err := p.db.GetAll(userId)
	failed := 0

	for i := 0; i < len(apps); i++ {
		t := apps[i]
		if t.Status == common_proto.AppStatus_START_FAILED || t.Status == common_proto.AppStatus_CANCEL_FAILED || t.Status == common_proto.AppStatus_UPDATE_FAILED {
			failed++
		}
	}

	rsp.ClusterCount = 0
	rsp.EnvironmentCount = 0
	rsp.RegionCount = 0
	rsp.TotalAppCount = 1
	rsp.HealthAppCount = 1

	if err == nil && len(apps) > 0 {
		rsp.ClusterCount = int32(len(apps))
		rsp.EnvironmentCount = int32(len(apps))
		rsp.RegionCount = 3
		rsp.TotalAppCount = int32(len(apps))
		rsp.HealthAppCount = int32(len(apps) - failed)

	}

	return nil
}

func (p *AppMgrHandler) AppLeaderBoard(ctx context.Context, req *common_proto.Empty, rsp *appmgr.AppLeaderBoardResponse) error {
	log.Printf("AppleaderBoard in app manager service\n")
	//rsp = &appmgr.AppLeaderBoardResponse{}
	list := make([]*appmgr.AppLeaderBoardDetail, 0)
	{
		detail := appmgr.AppLeaderBoardDetail{}
		detail.Name = "app_1"
		detail.Number = 99.34
		list = append(list, &detail)
	}

	{
		detail := appmgr.AppLeaderBoardDetail{}
		detail.Name = "app_2"
		detail.Number = 98.53
		list = append(list, &detail)
	}

	{
		detail := appmgr.AppLeaderBoardDetail{}
		detail.Name = "app_3"
		detail.Number = 97.98
		list = append(list, &detail)
	}

	userId := getUserID(ctx)
	apps, err := p.db.GetAll(userId)
	if err == nil && len(apps) > 0 {
		offset := len(apps)

		for i := 0; i < len(list); i++ {
			if offset == 0 {
				break
			}
			list[i].Name = apps[offset-1].Name
			offset--
		}
	}

	rsp.List = list

	log.Printf("AppleaderBoard  <<<<<>>>>>> %+v", rsp.List)

	return nil

}

func (p *AppMgrHandler) PurgeApp(ctx context.Context, req *appmgr.AppID, rsp *common_proto.Empty) error {
	error := p.CancelApp(ctx, req, rsp)

	if error == ankr_default.ErrCanceledTwice {
		return ankr_default.ErrPurgedTwice
	}

	if error == nil {
		log.Printf(" PurgeApp  %s \n", req.AppId)
		p.db.Update(req.AppId, bson.M{"$set": bson.M{"hidden": true}})
	}
	return error
}

func (p *AppMgrHandler) checkOwner(userId, appId string) (*common_proto.App, error) {
	app, err := p.db.Get(appId)

	if err != nil {
		return nil, err
	}

	log.Printf("appid : %s user id -%s-   user_token_id -%s-  ", appId, app.Userid, userId)

	if app.Userid != userId {
		return nil, ankr_default.ErrUserNotOwn
	}

	appMessage := convertToAppMessage(app)

	return &appMessage, nil
}

func checkId(userId, appId string) error {
	if userId == "" {
		return ankr_default.ErrUserNotExist
	}

	if appId == "" {
		return ankr_default.ErrUserNotOwn
	}

	return nil
}
