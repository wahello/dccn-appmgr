package handler

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"errors"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"

	db "github.com/Ankr-network/dccn-appmgr/db_service"
	"github.com/Ankr-network/dccn-common/protos"
	"github.com/Ankr-network/dccn-common/protos/appmgr/v1/micro"
	"github.com/Ankr-network/dccn-common/protos/common"
	common_util "github.com/Ankr-network/dccn-common/util"
	"github.com/google/uuid"
	micro "github.com/micro/go-micro"
	"gopkg.in/mgo.v2/bson"
	"k8s.io/helm/pkg/chartutil"
	"k8s.io/helm/pkg/proto/hapi/chart"
)

type AppMgrHandler struct {
	db        db.DBService
	deployApp micro.Publisher
}

func New(db db.DBService, deployApp micro.Publisher) *AppMgrHandler {

	return &AppMgrHandler{
		db:        db,
		deployApp: deployApp,
	}
}

type Token struct {
	Exp int64
	Jti string
	Iss string
}

func (p *AppMgrHandler) CreateApp(ctx context.Context, req *appmgr.CreateAppRequest, rsp *appmgr.CreateAppResponse) error {

	userID := common_util.GetUserID(ctx)
	log.Println("app manager service CreateApp")

	if req.App == nil {
		log.Printf("invalid input: null app provided, %+v \n", req.App)
		return errors.New("invalid input: null app provided")
	}

	appDeployment := &common_proto.AppDeployment{}
	appDeployment.Id = "app-" + uuid.New().String()
	appDeployment.Name = req.App.Name

	if req.App.NamespaceData == nil {
		log.Printf("invalid input: null namespace provided, %+v \n", req.App.NamespaceData)
		return errors.New("invalid input: null namespace provided")
	}

	switch req.App.NamespaceData.(type) {

	case *common_proto.App_NamespaceId:

		namespaceRecord, err := p.db.GetNamespace(req.App.GetNamespaceId())
		if err != nil {
			log.Printf("get namespace failed, %s", err.Error())
			return errors.New("internal error: get namespace failed")
		}
		if namespaceRecord.Status != common_proto.NamespaceStatus_NS_RUNNING {
			log.Printf("namespace status not running")
			return errors.New("internal error: namespace status not running")
		}

		appDeployment.Namespace = &common_proto.Namespace{
			Id:               namespaceRecord.ID,
			Name:             namespaceRecord.Name,
			ClusterId:        namespaceRecord.ClusterID,
			ClusterName:      namespaceRecord.ClusterName,
			CreationDate:     namespaceRecord.CreationDate,
			LastModifiedDate: namespaceRecord.LastModifiedDate,
			CpuLimit:         namespaceRecord.CpuLimit,
			MemLimit:         namespaceRecord.MemLimit,
			StorageLimit:     namespaceRecord.StorageLimit,
		}

	case *common_proto.App_Namespace:
		appDeployment.Namespace = req.App.GetNamespace()
		appDeployment.Namespace.Id = "ns-" + uuid.New().String()
		if err := p.db.CreateNamespace(appDeployment.Namespace, userID); err != nil {
			log.Println(err.Error())
			return err
		}
	}

	appDeployment.ChartDetail = req.App.ChartDetail
	appDeployment.Uid = userID
	rsp.AppId = appDeployment.Id

	event := common_proto.DCStream{
		OpType:    common_proto.DCOperation_APP_CREATE,
		OpPayload: &common_proto.DCStream_AppDeployment{AppDeployment: appDeployment},
	}

	if err := p.deployApp.Publish(context.Background(), &event); err != nil {
		log.Println(ankr_default.ErrPublish)
		return ankr_default.ErrPublish
	} else {
		log.Println("app manager service send CreateApp MQ message to dc manager service (api)")
	}

	if err := p.db.CreateApp(appDeployment, userID); err != nil {
		log.Println(err.Error())
		return err
	}

	return nil
}

// Must return nil for gRPC handler
func (p *AppMgrHandler) CancelApp(ctx context.Context, req *appmgr.AppID, rsp *common_proto.Empty) error {
	userID := common_util.GetUserID(ctx)
	log.Println("Debug into CancelApp")

	if err := checkId(userID, req.AppId); err != nil {
		log.Println(err.Error())
		return err
	}

	app, err := p.checkOwner(userID, req.AppId)
	if err != nil {
		log.Println(err.Error())
		return err
	}

	if app.AppStatus == common_proto.AppStatus_APP_CANCELED {
		return ankr_default.ErrCanceledTwice
	}

	event := common_proto.DCStream{
		OpType:    common_proto.DCOperation_APP_CANCEL,
		OpPayload: &common_proto.DCStream_AppDeployment{AppDeployment: app.AppDeployment},
	}

	if err := p.deployApp.Publish(context.Background(), &event); err != nil {
		log.Println(err.Error())
		return err
	}

	if err := p.db.Update("app", req.AppId, bson.M{"$set": bson.M{"status": common_proto.AppStatus_APP_CANCELING}}); err != nil {
		log.Println(err.Error())
		return err
	}

	return nil
}

func convertToAppMessage(app db.AppRecord, pdb db.DBService) common_proto.AppReport {
	message := common_proto.AppDeployment{}
	message.Id = app.ID
	message.Name = app.Name
	message.Attributes = &app.Attributes
	message.Uid = app.UID
	message.ChartDetail = &app.ChartDetail
	namespaceRecord, err := pdb.GetNamespace(app.NamespaceID)
	if err != nil {
		log.Printf("get namespace record failed, %s", err.Error())
	}
	namespace := convertFromNamespaceRecord(namespaceRecord)
	message.Namespace = namespace.Namespace
	appReport := common_proto.AppReport{
		AppDeployment: &message,
		AppStatus:     app.Status,
		AppEvent:      app.Event,
		Detail:        app.Detail,
		Report:        app.Report,
	}
	return appReport
}

func (p *AppMgrHandler) AppList(ctx context.Context, req *appmgr.AppListRequest, rsp *appmgr.AppListResponse) error {
	userId := common_util.GetUserID(ctx)
	log.Println("debug into AppList")

	apps, err := p.db.GetAllApp(userId)
	log.Printf(">>>>>>appMessage  %+v \n", apps)
	if err != nil {
		log.Println(err.Error())
		return err
	}

	appsWithoutHidden := make([]*common_proto.AppReport, 0)

	for i := 0; i < len(apps); i++ {
		if apps[i].Hidden != true {
			appMessage := convertToAppMessage(apps[i], p.db)
			log.Printf("appMessage  %+v \n", appMessage)
			appsWithoutHidden = append(appsWithoutHidden, &appMessage)
		}
	}

	rsp.AppReports = appsWithoutHidden

	return nil
}

func (p *AppMgrHandler) AppDetail(ctx context.Context, req *appmgr.AppID, rsp *appmgr.AppDetailResponse) error {
	log.Println("debug into AppDetail")

	appRecord, err := p.db.GetApp(req.AppId)

	log.Printf(">>>>>>appMessage  %+v \n", appRecord)
	if err != nil {
		log.Println(err.Error())
		return err
	}

	appMessage := convertToAppMessage(appRecord, p.db)
	log.Printf("appMessage  %+v \n", appMessage)
	rsp.AppReport = &appMessage

	event := common_proto.DCStream{
		OpType:    common_proto.DCOperation_APP_DETAIL,
		OpPayload: &common_proto.DCStream_AppDeployment{AppDeployment: appMessage.AppDeployment},
	}

	if err := p.deployApp.Publish(context.Background(), &event); err != nil {
		log.Println(err.Error())
		return err
	}

	return nil
}

func (p *AppMgrHandler) UpdateApp(ctx context.Context, req *appmgr.UpdateAppRequest, rsp *common_proto.Empty) error {
	userId := common_util.GetUserID(ctx)

	if req.AppDeployment == nil {
		log.Printf("invalid input: null app deployment provided, %+v \n", req.AppDeployment)
		return errors.New("invalid input: null app deployment provided")
	}

	if err := checkId(userId, req.AppDeployment.Id); err != nil {
		log.Println(err.Error())
		return err
	}

	appReport, err := p.checkOwner(userId, req.AppDeployment.Id)
	if err != nil {
		log.Println(err.Error())
		return err
	}

	if appReport.AppStatus != common_proto.AppStatus_APP_RUNNING &&
		appReport.AppStatus != common_proto.AppStatus_APP_UPDATE_FAILED {
		log.Println("app status is not running, cannot update")
		return errors.New("invalid input: app status is not running, cannot update")
	}
	appDeployment := appReport.AppDeployment
	appDeployment.ChartDetail = req.AppDeployment.ChartDetail

	event := common_proto.DCStream{
		OpType:    common_proto.DCOperation_APP_UPDATE,
		OpPayload: &common_proto.DCStream_AppDeployment{AppDeployment: appDeployment},
	}

	if err := p.deployApp.Publish(context.Background(), &event); err != nil {
		log.Println(err.Error())
		return err
	}
	// TODO: wait deamon notify
	if err := p.db.UpdateApp(appDeployment); err != nil {
		log.Println(err.Error())
		return err
	}
	return nil
}

func (p *AppMgrHandler) AppOverview(ctx context.Context, req *common_proto.Empty, rsp *appmgr.AppOverviewResponse) error {
	log.Printf("AppOverview in app manager service\n")
	//rsp = &appmgr.AppOverviewResponse{}
	userId := common_util.GetUserID(ctx)
	apps, err := p.db.GetAllApp(userId)
	failed := 0

	for i := 0; i < len(apps); i++ {
		t := apps[i]
		if t.Status == common_proto.AppStatus_APP_FAILED || t.Status == common_proto.AppStatus_APP_UPDATE_FAILED {
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

	userId := common_util.GetUserID(ctx)
	apps, err := p.db.GetAllApp(userId)
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

	appRecord, err := p.db.GetApp(req.AppId)

	if err != nil {
		log.Printf(" PurgeApp for app id %s not found \n", req.AppId)
		return errors.New("invalid input: app id not found")
	}

	if appRecord.Hidden == true {
		log.Printf(" app id %s already purged \n", req.AppId)
		return errors.New("invalid input: app id already purged")
	}

	if appRecord.Status != common_proto.AppStatus_APP_CANCELED {

		if err := p.CancelApp(ctx, req, rsp); err != nil {
			log.Printf(err.Error())
			return err
		}
	}

	log.Printf(" PurgeApp  %s \n", req.AppId)
	if err := p.db.Update("app", req.AppId, bson.M{"$set": bson.M{"hidden": true}}); err != nil {
		log.Printf(err.Error())
		return err
	}

	return nil
}

func (p *AppMgrHandler) checkOwner(userId, appId string) (*common_proto.AppReport, error) {
	appRecord, err := p.db.GetApp(appId)

	if err != nil {
		return nil, err
	}

	log.Printf("appid : %s user id -%s-   user_token_id -%s-  ", appId, appRecord.UID, userId)

	if appRecord.UID != userId {
		return nil, ankr_default.ErrUserNotOwn
	}

	appMessage := convertToAppMessage(appRecord, p.db)

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

// Chart is a struct representing a chartmuseum chart in the manifest
type Chart struct {
	Name        string       `json:"name"`
	Home        string       `json:"home"`
	Version     string       `json:"version"`
	Description string       `json:"description"`
	Keywords    []string     `json:"keywords"`
	Maintainers []Maintainer `json:"maintainers"`
	Icon        string       `json:"icon"`
	AppVersion  string       `json:"appVersion"`
	URLS        []string     `json:"urls"`
	Created     string       `json:"created"`
	Digest      string       `json:"digest"`
}

// Maintainer is a struct representing a maintainer inside a chart
type Maintainer struct {
	Name  string `json:"name"`
	Email string `json:"email"`
}

var chartmuseumURL string

// CreateChart will upload chart file to the chartmuseum in user catalog under "/user/userID"
func (p *AppMgrHandler) CreateChart(ctx context.Context, req *appmgr.CreateChartRequest, rsp *common_proto.Empty) error {

	log.Println("Uploading charts...")

	uid := common_util.GetUserID(ctx)

	if chartmuseumURL == "" {
		chartmuseumURL = "http://chart-dev.dccn.ankr.com:8080"
	}
	if req.ChartName == "" || req.ChartRepo == "" || req.ChartVer == "" || len(req.ChartFile) == 0 {
		log.Printf("invalid input, create failed.\n")
		return errors.New("invalid input, create failed")
	}
	query, err := http.Get(getChartURL(chartmuseumURL+"/api", uid, req.ChartRepo) + "/" + req.ChartName + "/" + req.ChartVer)
	if query.StatusCode == 200 {
		log.Printf("chart already exist, create failed.\n")
		return errors.New("chart already exist, create failed")
	}

	loadedChart, err := chartutil.LoadArchive(bytes.NewReader(req.ChartFile))
	if err != nil {
		log.Printf("cannot load chart from tar file, %s \n", req.ChartName, err.Error())
		return errors.New("internal error: cannot load chart from tar file")
	}

	loadedChart.Metadata.Version = req.ChartVer
	loadedChart.Metadata.Name = req.ChartName

	dest, err := os.Getwd()
	if err != nil {
		log.Printf("cannot get chart outdir")
		return errors.New("internal error: cannot get chart outdir")
	}
	log.Printf("save to outdir: %s\n", dest)
	tarballName, err := chartutil.Save(loadedChart, dest)
	if err == nil {
		log.Printf("Successfully packaged chart and saved it to: %s\n", tarballName)
	} else {
		log.Printf("Failed to save: %s", err)
		return errors.New("internal error: Failed to save chart to outdir")
	}

	tarball, err := os.Open(tarballName)
	if err != nil {
		log.Printf("cannot open chart tar file")
		return errors.New("internal error: cannot open chart tar file")
	}

	chartReq, err := http.NewRequest("POST", getChartURL(chartmuseumURL+"/api", uid, req.ChartRepo), tarball)
	if err != nil {
		log.Printf("cannot open chart tar file, %s \n", err.Error())
		return errors.New("internal error: cannot open chart tar file")
	}

	chartRes, err := http.DefaultClient.Do(chartReq)
	if err != nil {
		log.Printf("cannot upload chart tar file, %s \n", err.Error())
		return errors.New("internal error: cannot upload chart tar file")
	}

	message, _ := ioutil.ReadAll(chartRes.Body)
	defer chartRes.Body.Close()
	log.Printf(string(message))

	if err = os.Remove(tarballName); err != nil {
		log.Printf("delete temp chart tarball failed, %s \n", err.Error())
	}
	return nil
}

// SaveValuesToChart will save modified values.yaml file to the chart and upload new version to the chartmuseum
func (p *AppMgrHandler) SaveChart(ctx context.Context, req *appmgr.SaveChartRequest, rsp *common_proto.Empty) error {

	log.Println("Saving charts...")

	uid := common_util.GetUserID(ctx)

	if chartmuseumURL == "" {
		chartmuseumURL = "http://chart-dev.dccn.ankr.com:8080"
	}
	if req.ChartName == "" || req.ChartRepo == "" || req.ChartVer == "" || req.SaveName == "" || req.SaveRepo == "" || req.SaveVer == "" {
		log.Printf("invalid input: empty chart properties not accepted \n")
		return errors.New("invalid input: empty chart properties not accepted")
	}

	querySaveChart, err := http.Get(getChartURL(chartmuseumURL+"/api", uid, req.SaveRepo) + "/" + req.SaveName + "/" + req.SaveVer)
	if err != nil {
		log.Printf("cannot get chart %s from chartmuseum\n", req.SaveName, err.Error())
		return errors.New("internal error: cannot get chart from chartmuseum")
	}
	if querySaveChart.StatusCode == 200 {
		log.Printf("invalid input: save chart already exist \n")
		return errors.New("invalid input: save chart already exist")
	}

	queryOriginalChart, err := http.Get(getChartURL(chartmuseumURL, uid, req.ChartRepo) + "/" + req.ChartName + "-" + req.ChartVer + ".tgz")
	if err != nil {
		log.Printf("cannot get chart %s from chartmuseum\n", req.ChartName, err.Error())
		return errors.New("internal error: cannot get chart from chartmuseum")
	}
	if queryOriginalChart.StatusCode != 200 {
		log.Printf("invalid input: original chart not exist \n")
		return errors.New("invalid input: original chart not exist")
	}

	defer queryOriginalChart.Body.Close()

	loadedChart, err := chartutil.LoadArchive(queryOriginalChart.Body)
	if err != nil {
		log.Printf("cannot load chart from the http get response from chartmuseum , %s \n", req.ChartName, err.Error())
		return errors.New("internal error: cannot load chart from the http get response from chartmuseum")
	}

	loadedChart.Metadata.Version = req.SaveVer
	loadedChart.Metadata.Name = req.SaveName
	loadedChart.Values = &chart.Config{
		Raw: string(req.ValuesFile),
	}

	dest, err := os.Getwd()
	if err != nil {
		log.Printf("cannot get chart outdir")
		return errors.New("internal error: cannot get chart outdir")
	}
	log.Printf("save to outdir: %s\n", dest)
	tarballName, err := chartutil.Save(loadedChart, dest)
	if err == nil {
		log.Printf("Successfully packaged chart and saved it to: %s\n", tarballName)
	} else {
		log.Printf("Failed to save: %s", err)
		return errors.New("internal error: Failed to save chart to outdir")
	}

	tarball, err := os.Open(tarballName)
	if err != nil {
		log.Printf("cannot open chart tar file")
		return errors.New("internal error: cannot open chart tar file")
	}

	chartReq, err := http.NewRequest("POST", getChartURL(chartmuseumURL+"/api", uid, req.SaveRepo), tarball)
	if err != nil {
		log.Printf("cannot open chart tar file, %s \n", err.Error())
		return errors.New("internal error: cannot open chart tar file")
	}

	chartRes, err := http.DefaultClient.Do(chartReq)
	if err != nil {
		log.Printf("cannot upload chart tar file, %s \n", err.Error())
		return errors.New("internal error: cannot upload chart tar file")
	}

	message, _ := ioutil.ReadAll(chartRes.Body)
	defer chartRes.Body.Close()
	log.Printf(string(message))

	if err := os.Remove(tarballName); err != nil {
		log.Printf("delete temp chart tarball failed, %s \n", err.Error())
	}
	return nil
}

// ChartList will return a list of charts from the specific chartmuseum repo
func (p *AppMgrHandler) ChartList(ctx context.Context, req *appmgr.ChartListRequest, rsp *appmgr.ChartListResponse) error {

	log.Println("Checking for charts...")

	uid := common_util.GetUserID(ctx)

	if chartmuseumURL == "" {
		chartmuseumURL = "http://chart-dev.dccn.ankr.com:8080"
	}
	chartRes, err := http.Get(getChartURL(chartmuseumURL+"/api", uid, req.ChartRepo))
	if err != nil {
		log.Printf("cannot get chart list, %s \n", err.Error())
		return errors.New("internal error: cannot get chart list")
	}

	defer chartRes.Body.Close()

	message, err := ioutil.ReadAll(chartRes.Body)
	if err != nil {
		log.Printf("cannot get chart list response body, %s \n", err.Error())
		return errors.New("internal error: cannot get chart list response body")
	}

	data := map[string][]Chart{}
	if err := json.Unmarshal([]byte(message), &data); err != nil {
		log.Printf("cannot unmarshal chart list, %s \n", err.Error())
		return errors.New("internal error: cannot unmarshal chart list")
	}

	charts := make([]*common_proto.Chart, 0)

	for _, v := range data {
		chart := common_proto.Chart{
			Name:             v[0].Name,
			Repo:             req.ChartRepo,
			Description:      v[0].Description,
			IconUrl:          v[0].Icon,
			LatestVersion:    v[0].Version,
			LatestAppVersion: v[0].AppVersion,
		}
		charts = append(charts, &chart)
	}

	rsp.Charts = charts

	return nil
}

// ChartDetail will return a list of specific chart versions from the specific chartmuseum repo
func (p *AppMgrHandler) ChartDetail(ctx context.Context, req *appmgr.ChartDetailRequest, rsp *appmgr.ChartDetailResponse) error {

	log.Println("Checking for chart details...")

	uid := common_util.GetUserID(ctx)

	if chartmuseumURL == "" {
		chartmuseumURL = "http://chart-dev.dccn.ankr.com:8080"
	}

	if req.Chart == nil {
		log.Printf("invalid input: null chart provided, %+v \n", req.Chart)
		return errors.New("invalid input: null chart provided")
	}

	chartRes, err := http.Get(getChartURL(chartmuseumURL+"/api", uid, req.Chart.Repo) + "/" + req.Chart.Name)
	if err != nil {
		log.Printf("cannot get chart details, %s \n", err.Error())
		return errors.New("internal error: cannot get chart details")
	}

	defer chartRes.Body.Close()

	message, err := ioutil.ReadAll(chartRes.Body)
	if err != nil {
		log.Printf("cannot get chart details response body, %s \n", err.Error())
		return errors.New("internal error: cannot get chart details response body")
	}

	data := []Chart{}
	if err := json.Unmarshal([]byte(message), &data); err != nil {
		log.Printf("cannot unmarshal chart details, %s \n", err.Error())
		return errors.New("internal error: cannot unmarshal chart details")
	}

	chartDetails := make([]*common_proto.ChartDetail, 0)
	for _, v := range data {
		chartdetail := common_proto.ChartDetail{
			Name:       v.Name,
			Repo:       req.Chart.Repo,
			Version:    v.Version,
			AppVersion: v.AppVersion,
		}
		chartDetails = append(chartDetails, &chartdetail)
	}
	rsp.Chartdetails = chartDetails

	tarballReq, err := http.NewRequest("GET", getChartURL(chartmuseumURL, uid, req.Chart.Repo)+"/"+req.Chart.Name+"-"+req.ShowVersion+".tgz", nil)
	if err != nil {
		log.Printf("cannot create show version tarball request, %s \n", err.Error())
		return errors.New("internal error: create show version tarball request")
	}

	tarballRes, err := http.DefaultClient.Do(tarballReq)
	if err != nil {
		log.Printf("cannot download chart tarball, %s \n", err.Error())
		return errors.New("internal error: cannot download chart tarball")
	}
	defer tarballRes.Body.Close()

	gzf, err := gzip.NewReader(tarballRes.Body)
	if err != nil {
		log.Printf("cannot open chart tarball, %s \n", err.Error())
		return errors.New("internal error: cannot open chart tarball")
	}
	defer gzf.Close()

	tarball := tar.NewReader(gzf)
	tarf := make(map[string]string)
	tarf[req.Chart.Name+"/README.md"] = ""
	tarf[req.Chart.Name+"/values.yaml"] = ""

	if err = extractFromTarfile(tarf, tarball); err != nil {
		log.Printf("cannot find readme/value in chart tarball, %s \n", err.Error())
		return errors.New("internal error: cannot find readme in chart tarball")
	}
	if tarf[req.Chart.Name+"/README.md"] == "" {
		log.Printf("cannot find readme in chart tarball, %s \n", err.Error())
	}
	if tarf[req.Chart.Name+"/values.yaml"] == "" {
		log.Printf("cannot find value in chart tarball, %s \n", err.Error())
	}

	rsp.ShowReadme = tarf[req.Chart.Name+"/README.md"]
	rsp.ShowValues = tarf[req.Chart.Name+"/values.yaml"]

	return nil
}

// DownloadChart will return a specific chart tarball from the specific chartmuseum repo
func (p *AppMgrHandler) DownloadChart(ctx context.Context, req *appmgr.DownloadChartRequest, rsp *appmgr.DownloadChartResponse) error {

	log.Println("Download chart tarball...")

	uid := common_util.GetUserID(ctx)

	if chartmuseumURL == "" {
		chartmuseumURL = "http://chart-dev.dccn.ankr.com:8080"
	}

	if req.ChartName == "" || req.ChartRepo == "" || req.ChartVer == "" {
		log.Printf("invalid input: null chart detail provided, %+v \n", req)
		return errors.New("invalid input: null chart detail provided")
	}

	tarballReq, err := http.NewRequest("GET", getChartURL(chartmuseumURL, uid, req.ChartRepo)+"/"+req.ChartName+"-"+req.ChartVer+".tgz", nil)
	if err != nil {
		log.Printf("cannot create show version tarball request, %s \n", err.Error())
		return errors.New("internal error: create show version tarball request")
	}

	tarballRes, err := http.DefaultClient.Do(tarballReq)
	if err != nil {
		log.Printf("cannot download chart tarball, %s \n", err.Error())
		return errors.New("internal error: cannot download chart tarball")
	}
	defer tarballRes.Body.Close()

	rsp.ChartFile, err = ioutil.ReadAll(tarballRes.Body)
	if err != nil {
		log.Printf("cannot read chart tarball, %s \n", err.Error())
		return errors.New("internal error: cannot read chart tarball")
	}

	return nil
}

func (p *AppMgrHandler) DeleteChart(ctx context.Context, req *appmgr.DeleteChartRequest, res *common_proto.Empty) error {
	log.Println("Deleting charts...")

	uid := common_util.GetUserID(ctx)
	if chartmuseumURL == "" {
		chartmuseumURL = "http://chart-dev.dccn.ankr.com:8080"
	}
	query, err := http.Get(getChartURL(chartmuseumURL+"/api", uid, req.ChartRepo) + "/" + req.ChartName + "/" + req.ChartVer)
	if query.StatusCode != 200 {
		log.Printf("chart not exist, delete failed.\n")
		return errors.New("chart not exist, delete failed")
	}

	delReq, err := http.NewRequest("DELETE", getChartURL(chartmuseumURL+"/api", uid, req.ChartRepo)+"/"+req.ChartName+"/"+req.ChartVer, nil)
	if err != nil {
		log.Printf("cannot create delete chart request, %s \n", err.Error())
		return errors.New("internal error: cannot create delete chart request")
	}

	delRes, err := http.DefaultClient.Do(delReq)
	if err != nil {
		log.Printf("cannot delete chart file, %s \n", err.Error())
		return errors.New("internal error: cannot delete chart file")
	}
	defer delRes.Body.Close()

	return nil
}

func getChartURL(url string, uid string, repo string) string {

	if repo == "user" {
		url += "/user/" + uid + "/charts"
	} else {
		url += "/public/" + repo + "/charts"
	}
	return url
}

func extractFromTarfile(tarf map[string]string, tarball *tar.Reader) error {

	count := len(tarf)
	for count > 0 {

		header, err := tarball.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		if _, ok := tarf[header.Name]; ok {
			var b bytes.Buffer
			io.Copy(&b, tarball)
			tarf[header.Name] = string(b.Bytes())
			count--
		}

	}

	return nil
}

func (p *AppMgrHandler) CreateNamespace(ctx context.Context, req *appmgr.CreateNamespaceRequest, rsp *appmgr.CreateNamespaceResponse) error {
	uid := common_util.GetUserID(ctx)

	log.Printf("app manager service CreateNamespace: %+v", req)

	req.Namespace.Id = "ns-" + uuid.New().String()

	event := common_proto.DCStream{
		OpType:    common_proto.DCOperation_NS_CREATE,
		OpPayload: &common_proto.DCStream_Namespace{Namespace: req.Namespace},
	}

	if err := p.deployApp.Publish(context.Background(), &event); err != nil {
		log.Println(ankr_default.ErrPublish)
		return ankr_default.ErrPublish
	} else {
		log.Println("app manager service send CreateNamespace MQ message to dc manager service (api)")
	}

	if err := p.db.CreateNamespace(req.Namespace, uid); err != nil {
		log.Println(err.Error())
		return err
	}

	rsp.Id = req.Namespace.Id

	return nil
}

func convertFromNamespaceRecord(namespace db.NamespaceRecord) common_proto.NamespaceReport {
	message := common_proto.Namespace{}
	message.Id = namespace.ID
	message.Name = namespace.Name
	message.ClusterId = namespace.ClusterID
	message.ClusterName = namespace.ClusterName
	message.CreationDate = namespace.CreationDate
	message.CpuLimit = namespace.CpuLimit
	message.MemLimit = namespace.MemLimit
	message.StorageLimit = namespace.StorageLimit
	namespaceReport := common_proto.NamespaceReport{
		Namespace: &message,
		NsEvent:   namespace.Event,
		NsStatus:  namespace.Status,
	}
	return namespaceReport
}

func (p *AppMgrHandler) NamespaceList(ctx context.Context, req *appmgr.NamespaceListRequest, rsp *appmgr.NamespaceListResponse) error {
	userId := common_util.GetUserID(ctx)
	log.Println("app service into NamespaceList")

	namespaceRecords, err := p.db.GetAllNamespace(userId)
	log.Printf(">>>>>>NamespaceMessage  %+v \n", namespaceRecords)
	if err != nil {
		log.Println(err.Error())
		return err
	}

	namespacesWithoutCancel := make([]*common_proto.NamespaceReport, 0)

	for i := 0; i < len(namespaceRecords); i++ {
		if namespaceRecords[i].Status != common_proto.NamespaceStatus_NS_CANCELED {
			NamespaceMessage := convertFromNamespaceRecord(namespaceRecords[i])
			log.Printf("NamespaceMessage  %+v \n", NamespaceMessage)
			namespacesWithoutCancel = append(namespacesWithoutCancel, &NamespaceMessage)
		}
	}

	rsp.NamespaceReports = namespacesWithoutCancel

	return nil
}

func (p *AppMgrHandler) UpdateNamespace(ctx context.Context, req *appmgr.UpdateNamespaceRequest, rsp *common_proto.Empty) error {
	userId := common_util.GetUserID(ctx)

	namespaceRecord, err := p.db.GetNamespace(req.Namespace.Id)
	if err != nil {
		log.Println(err.Error())
		return err
	}
	if err := checkId(userId, namespaceRecord.UID); err != nil {
		log.Println(err.Error())
		return err
	}

	if namespaceRecord.Status != common_proto.NamespaceStatus_NS_RUNNING &&
		namespaceRecord.Status != common_proto.NamespaceStatus_NS_UPDATE_FAILED {
		log.Println("namespace status is not running, cannot update")
		return errors.New("invalid input: namespace status is not running, cannot update")
	}

	namespaceReport := convertFromNamespaceRecord(namespaceRecord)
	namespaceReport.Namespace.CpuLimit = req.Namespace.CpuLimit
	namespaceReport.Namespace.MemLimit = req.Namespace.MemLimit
	namespaceReport.Namespace.StorageLimit = req.Namespace.StorageLimit
	namespaceReport.Namespace.Name = req.Namespace.Name

	event := common_proto.DCStream{
		OpType:    common_proto.DCOperation_NS_UPDATE,
		OpPayload: &common_proto.DCStream_Namespace{Namespace: namespaceReport.Namespace},
	}

	if err := p.deployApp.Publish(context.Background(), &event); err != nil {
		log.Println(err.Error())
		return err
	}
	// TODO: wait deamon notify
	if err := p.db.UpdateNamespace(req.Namespace); err != nil {
		log.Println(err.Error())
		return err
	}
	return nil
}

func (p *AppMgrHandler) DeleteNamespace(ctx context.Context, req *appmgr.DeleteNamespaceRequest, rsp *common_proto.Empty) error {
	userId := common_util.GetUserID(ctx)
	log.Println("Debug into DeleteNamespace")

	namespaceRecord, err := p.db.GetNamespace(req.Id)
	if err != nil {
		log.Println(err.Error())
		return err
	}

	if err := checkId(userId, namespaceRecord.UID); err != nil {
		log.Println(err.Error())
		return err
	}

	if namespaceRecord.Status == common_proto.NamespaceStatus_NS_CANCELED {
		return ankr_default.ErrCanceledTwice
	}

	namespaceReport := convertFromNamespaceRecord(namespaceRecord)

	event := common_proto.DCStream{
		OpType:    common_proto.DCOperation_NS_CANCEL,
		OpPayload: &common_proto.DCStream_Namespace{Namespace: namespaceReport.Namespace},
	}

	if err := p.deployApp.Publish(context.Background(), &event); err != nil {
		log.Println(err.Error())
		return err
	}

	if err := p.db.Update("namespace", req.Id, bson.M{"$set": bson.M{"status": common_proto.NamespaceStatus_NS_CANCELING}}); err != nil {
		log.Println(err.Error())
		return err
	}

	return nil
}
