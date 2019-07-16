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
	"strings"
	"time"

	db "github.com/Ankr-network/dccn-appmgr/db_service"
	micro2 "github.com/Ankr-network/dccn-common/ankr-micro"
	ankr_default "github.com/Ankr-network/dccn-common/protos"
	appmgr "github.com/Ankr-network/dccn-common/protos/appmgr/v1/grpc"
	common_proto "github.com/Ankr-network/dccn-common/protos/common"
	common_util "github.com/Ankr-network/dccn-common/util"
	"github.com/Masterminds/semver"
	"github.com/golang/protobuf/ptypes/timestamp"
	"github.com/google/uuid"
	"google.golang.org/grpc/status"
	"gopkg.in/mgo.v2/bson"
	"k8s.io/helm/pkg/chartutil"
	"k8s.io/helm/pkg/proto/hapi/chart"
)

type AppMgrHandler struct {
	db        db.DBService
	deployApp *micro2.Publisher
}

func New(db db.DBService, deployApp *micro2.Publisher) *AppMgrHandler {

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

func (p *AppMgrHandler) CreateApp(ctx context.Context, req *appmgr.CreateAppRequest) (*appmgr.CreateAppResponse, error) {

	userID := common_util.GetUserID(ctx)
	log.Printf(">>>>>>>>>Debug into CreateApp %+v \nctx: %+v \n", req, ctx)

	if req.App == nil {
		log.Printf("invalid input: null app provided, %+v \n", req)
		return nil, ankr_default.ErrNoApp
	}

	appDeployment := &common_proto.AppDeployment{}
	appDeployment.AppId = "app-" + uuid.New().String()
	rsp := &appmgr.CreateAppResponse{
		AppId: appDeployment.AppId,
	}
	appDeployment.AppName = req.App.AppName
	if len(appDeployment.AppName) == 0 {
		log.Printf("invalid input: null app name provided, %+v \n", req.App.AppName)
		return rsp, ankr_default.ErrNoAppname
	}

	if req.App.NamespaceData == nil {
		log.Printf("invalid input: null namespace provided, %+v \n", req.App.NamespaceData)
		return rsp, ankr_default.ErrNsEmpty
	}

	switch req.App.NamespaceData.(type) {

	case *common_proto.App_NsId:

		namespaceRecord, err := p.db.GetNamespace(req.App.GetNsId())
		if err != nil {
			log.Printf("get namespace failed, %s", err.Error())
			return rsp, err
		}
		if namespaceRecord.Status != common_proto.NamespaceStatus_NS_RUNNING {
			log.Printf("namespace status not running")
			return rsp, ankr_default.ErrStatusNotSupportOperation
		}

		clusterConnection, err := p.db.GetClusterConnection(namespaceRecord.ClusterID)
		if err != nil || clusterConnection.Status != common_proto.DCStatus_AVAILABLE {
			log.Println("cluster connection not available, app can not be created")
			return rsp, errors.New("cluster connection not available, app can not be created")
		}

		appDeployment.Namespace = &common_proto.Namespace{
			NsId:             namespaceRecord.ID,
			NsName:           namespaceRecord.Name,
			ClusterId:        namespaceRecord.ClusterID,
			ClusterName:      namespaceRecord.ClusterName,
			CreationDate:     namespaceRecord.CreationDate,
			LastModifiedDate: namespaceRecord.LastModifiedDate,
			NsCpuLimit:       namespaceRecord.CpuLimit,
			NsMemLimit:       namespaceRecord.MemLimit,
			NsStorageLimit:   namespaceRecord.StorageLimit,
		}

	case *common_proto.App_Namespace:
		appDeployment.Namespace = req.App.GetNamespace()
		if appDeployment.Namespace == nil || appDeployment.Namespace.NsCpuLimit == 0 ||
			appDeployment.Namespace.NsMemLimit == 0 || appDeployment.Namespace.NsStorageLimit == 0 {
			log.Printf("invalid input: empty namespace properties not accepted \n")
			return rsp, ankr_default.ErrNsEmpty
		}
		if len(appDeployment.Namespace.ClusterId) > 0 {
			clusterConnection, err := p.db.GetClusterConnection(appDeployment.Namespace.ClusterId)
			if err != nil || clusterConnection.Status != common_proto.DCStatus_AVAILABLE {
				log.Println("cluster connection not available, app can not be created")
				return rsp, errors.New("cluster connection not available, app can not be created")
			}
		}
		appDeployment.Namespace.NsId = "ns-" + uuid.New().String()
		if err := p.db.CreateNamespace(appDeployment.Namespace, userID); err != nil {
			log.Println(err.Error())
			return rsp, err
		}
	}

	if req.App.ChartDetail == nil {
		log.Printf("invalid input: null chart detail provided, %+v \n", req.App)
		return rsp, ankr_default.ErrChartDetailEmpty
	}
	appDeployment.ChartDetail = req.App.ChartDetail
	appChart, err := http.Get(getChartURL(chartmuseumURL, userID,
		req.App.ChartDetail.ChartRepo) + "/" + req.App.ChartDetail.ChartName + "-" + req.App.ChartDetail.ChartVer + ".tgz")
	if err != nil {
		log.Printf("cannot get app chart %s from chartmuseum\nerror: %s\n", req.App.ChartDetail.ChartName, err.Error())
		return rsp, ankr_default.ErrChartMuseumGet
	}
	if appChart.StatusCode != 200 {
		log.Printf("invalid input: app chart not exist \n")
		return rsp, ankr_default.ErrChartNotExist
	}

	defer appChart.Body.Close()

	loadedChart, err := chartutil.LoadArchive(appChart.Body)
	if err != nil {
		log.Printf("cannot load chart from the http get response from chartmuseum , %s \nerror: %s",
			req.App.ChartDetail.ChartName, err.Error())
		return rsp, ankr_default.ErrChartMuseumGet
	}

	appDeployment.ChartDetail.ChartAppVer = loadedChart.Metadata.AppVersion
	appDeployment.ChartDetail.ChartIconUrl = loadedChart.Metadata.Icon

	appDeployment.Uid = userID

	event := common_proto.DCStream{
		OpType:    common_proto.DCOperation_APP_CREATE,
		OpPayload: &common_proto.DCStream_AppDeployment{AppDeployment: appDeployment},
	}

	if err := p.deployApp.Publish(&event); err != nil {
		log.Println(ankr_default.ErrPublish)
		return rsp, ankr_default.ErrPublish
	} else {
		log.Println("app manager service send CreateApp MQ message to dc manager service (api)")
	}

	if err := p.db.CreateApp(appDeployment, userID); err != nil {
		log.Println(err.Error())
		return rsp, err
	}

	return rsp, nil
}

// Must return nil for gRPC handler
func (p *AppMgrHandler) CancelApp(ctx context.Context, req *appmgr.AppID) (*common_proto.Empty, error) {
	userID := common_util.GetUserID(ctx)
	log.Printf(">>>>>>>>>Debug into CancelApp: %+v\nctx: %+v \n", req, ctx)

	if err := checkId(userID, req.AppId); err != nil {
		log.Println(err.Error())
		return &common_proto.Empty{}, err
	}

	app, err := p.checkOwner(userID, req.AppId)
	if err != nil {
		log.Println(err.Error())
		return &common_proto.Empty{}, err
	}

	if app.AppStatus == common_proto.AppStatus_APP_CANCELED {
		return &common_proto.Empty{}, ankr_default.ErrCanceledTwice
	}
	/*
		clusterConnection, err := p.db.GetClusterConnection(app.AppDeployment.Namespace.ClusterId)
		if err != nil || clusterConnection.Status != common_proto.DCStatus_AVAILABLE {
			log.Println("cluster connection not available, app can not be canceled")
			return errors.New("cluster connection not available, app can not be canceled")
		}
	*/
	event := common_proto.DCStream{
		OpType:    common_proto.DCOperation_APP_CANCEL,
		OpPayload: &common_proto.DCStream_AppDeployment{AppDeployment: app.AppDeployment},
	}

	if err := p.deployApp.Publish(&event); err != nil {
		log.Println(err.Error())
		return &common_proto.Empty{}, ankr_default.ErrPublish
	}

	if err := p.db.Update("app", req.AppId, bson.M{"$set": bson.M{"status": common_proto.AppStatus_APP_CANCELING}}); err != nil {
		log.Println(err.Error())
		return &common_proto.Empty{}, err
	}

	return &common_proto.Empty{}, nil
}

func convertToAppMessage(app db.AppRecord, pdb db.DBService) common_proto.AppReport {
	message := common_proto.AppDeployment{}
	message.AppId = app.ID
	message.AppName = app.Name
	message.Attributes = &common_proto.AppAttributes{
		CreationDate:     app.CreationDate,
		LastModifiedDate: app.LastModifiedDate,
	}
	message.Uid = app.UID
	message.ChartDetail = &app.ChartDetail
	namespaceRecord, err := pdb.GetNamespace(app.NamespaceID)
	if err != nil {
		log.Printf("get namespace record failed, %s", err.Error())
	}
	namespaceReport := convertFromNamespaceRecord(namespaceRecord)
	message.Namespace = namespaceReport.Namespace
	appReport := common_proto.AppReport{
		AppDeployment: &message,
		AppStatus:     app.Status,
		AppEvent:      app.Event,
		Detail:        app.Detail,
		Report:        app.Report,
	}
	if len(app.Detail) > 0 && len(message.Namespace.ClusterId) > 0 &&
		strings.Contains(app.Detail, app.ID+"."+message.Namespace.ClusterId+".ankr.com") {
		appReport.Endpoint = app.ID + "." + message.Namespace.ClusterId + ".ankr.com"
	}

	return appReport
}

func (p *AppMgrHandler) AppList(ctx context.Context, req *common_proto.Empty) (*appmgr.AppListResponse, error) {
	userId := common_util.GetUserID(ctx)
	log.Printf(">>>>>>>>>Debug into AppList, ctx: %+v \n", ctx)

	apps, err := p.db.GetAllApp(userId)
	rsp := &appmgr.AppListResponse{}
	log.Printf("appMessage  %+v \n", apps)
	if err != nil {
		log.Println(err.Error())
		return rsp, err
	}

	appsWithoutHidden := make([]*common_proto.AppReport, 0)

	for i := 0; i < len(apps); i++ {
		if !apps[i].Hidden && (apps[i].Status == common_proto.AppStatus_APP_CANCELED ||
			apps[i].Status == common_proto.AppStatus_APP_CANCELING) && (apps[i].LastModifiedDate == nil ||
			apps[i].LastModifiedDate.Seconds < (time.Now().Unix()-7200)) {
			log.Printf("Canceled App time exceeded 7200 seconds, Now: %+v \n", time.Now().Unix())
			apps[i].Hidden = true
			p.db.Update("app", apps[i].ID, bson.M{"$set": bson.M{"hidden": true,
				"lastmodifieddate": &timestamp.Timestamp{Seconds: time.Now().Unix()}}})
		}
		if !apps[i].Hidden {
			appMessage := convertToAppMessage(apps[i], p.db)
			log.Printf("appMessage  %+v \n", appMessage)
			if len(appMessage.AppDeployment.Namespace.ClusterId) > 0 {
				clusterConnection, err := p.db.GetClusterConnection(appMessage.AppDeployment.Namespace.ClusterId)
				if err != nil || clusterConnection.Status == common_proto.DCStatus_UNAVAILABLE {
					appMessage.AppStatus = common_proto.AppStatus_APP_UNAVAILABLE
					appMessage.AppEvent = common_proto.AppEvent_APP_HEARTBEAT_FAILED
				}
			}
			appsWithoutHidden = append(appsWithoutHidden, &appMessage)
			if appMessage.AppStatus == common_proto.AppStatus_APP_RUNNING {
				event := common_proto.DCStream{
					OpType:    common_proto.DCOperation_APP_DETAIL,
					OpPayload: &common_proto.DCStream_AppDeployment{AppDeployment: appMessage.AppDeployment},
				}
				if err := p.deployApp.Publish(&event); err != nil {
					log.Println(err.Error())
				}
			}
		}
	}

	rsp.AppReports = appsWithoutHidden

	return rsp, nil
}

func (p *AppMgrHandler) AppDetail(ctx context.Context, req *appmgr.AppID) (*appmgr.AppDetailResponse, error) {
	log.Printf(">>>>>>>>>Debug into AppDetail: %+v\nctx: %+v \n", req, ctx)

	rsp := &appmgr.AppDetailResponse{}
	appRecord, err := p.db.GetApp(req.AppId)
	if err != nil {
		log.Println(err.Error())
		return rsp, err
	}

	log.Printf("appMessage  %+v \n", appRecord)
	if err != nil {
		log.Println(err.Error())
		return rsp, err
	}

	if appRecord.Hidden == true {
		log.Printf("app id %s already purged \n", req.AppId)
		return rsp, ankr_default.ErrAlreadyPurged
	}

	appMessage := convertToAppMessage(appRecord, p.db)
	log.Printf("appMessage %+v \n", appMessage)
	rsp.AppReport = &appMessage

	event := common_proto.DCStream{
		OpType:    common_proto.DCOperation_APP_DETAIL,
		OpPayload: &common_proto.DCStream_AppDeployment{AppDeployment: appMessage.AppDeployment},
	}

	if err := p.deployApp.Publish(&event); err != nil {
		log.Println(err.Error())
		return rsp, ankr_default.ErrPublish
	}

	return rsp, nil
}

func (p *AppMgrHandler) AppCount(ctx context.Context,
	req *appmgr.AppCountRequest) (*appmgr.AppCountResponse, error) {
	log.Printf(">>>>>>>>>Debug into AppCount %+v\nctx: %+v \n", req, ctx)
	rsp := &appmgr.AppCountResponse{}
	if len(req.Uid) == 0 || len(req.ClusterId) == 0 {
		rsp.AppCount = 0
	}

	appRecord, err := p.db.GetAppCount(req.Uid, req.ClusterId)
	if err != nil {
		log.Println(err.Error())
		return rsp, err
	}

	rsp.AppCount = uint32(len(appRecord))

	return rsp, nil
}

func (p *AppMgrHandler) UpdateApp(ctx context.Context,
	req *appmgr.UpdateAppRequest) (*common_proto.Empty, error) {
	log.Printf(">>>>>>>>>Debug into UpdateApp: %+v\nctx: %+v\n", req, ctx)
	uid := common_util.GetUserID(ctx)

	if req.AppDeployment == nil || (req.AppDeployment.ChartDetail == nil ||
		len(req.AppDeployment.ChartDetail.ChartVer) == 0) && len(req.AppDeployment.AppName) == 0 {
		log.Printf("invalid input: no valid update app parameters, %+v \n", req.AppDeployment)
		return &common_proto.Empty{}, errors.New("invalid input: no valid update app parameters")
	}

	if err := checkId(uid, req.AppDeployment.AppId); err != nil {
		log.Println(err.Error())
		return &common_proto.Empty{}, err
	}

	appReport, err := p.checkOwner(uid, req.AppDeployment.AppId)
	if err != nil {
		log.Println(err.Error())
		return &common_proto.Empty{}, err
	}

	if appReport.AppStatus != common_proto.AppStatus_APP_RUNNING &&
		appReport.AppStatus != common_proto.AppStatus_APP_UPDATE_FAILED {
		log.Println("app status is not running, cannot update")
		return &common_proto.Empty{}, ankr_default.ErrStatusNotSupportOperation
	}

	appDeployment := appReport.AppDeployment

	clusterConnection, err := p.db.GetClusterConnection(appDeployment.Namespace.ClusterId)
	if err != nil || clusterConnection.Status != common_proto.DCStatus_AVAILABLE {
		log.Println("cluster connection not available, app can not be updated")
		return &common_proto.Empty{}, errors.New("cluster connection not available, app can not be updated")
	}

	if len(req.AppDeployment.AppName) > 0 {
		if err := p.db.Update("app", appDeployment.AppId,
			bson.M{"$set": bson.M{"name": req.AppDeployment.AppName}}); err != nil {
			log.Printf(err.Error())
			return &common_proto.Empty{}, err
		}
	}

	if req.AppDeployment.ChartDetail != nil && len(req.AppDeployment.ChartDetail.ChartVer) > 0 &&
		req.AppDeployment.ChartDetail.ChartVer != appDeployment.ChartDetail.ChartVer {
		appChart, err := http.Get(getChartURL(chartmuseumURL, uid,
			appDeployment.ChartDetail.ChartRepo) + "/" + appDeployment.ChartDetail.ChartName + "-" + req.AppDeployment.ChartDetail.ChartVer + ".tgz")
		if err != nil {
			log.Printf("cannot get app chart %s from chartmuseum\nerror: %s\n", req.AppDeployment.ChartDetail.ChartName, err.Error())
			return &common_proto.Empty{}, ankr_default.ErrChartMuseumGet
		}
		if appChart.StatusCode != 200 {
			log.Printf("invalid input: app chart not exist \n")
			return &common_proto.Empty{}, ankr_default.ErrChartNotExist
		}

		appDeployment.ChartDetail.ChartVer = req.AppDeployment.ChartDetail.ChartVer

		event := common_proto.DCStream{
			OpType:    common_proto.DCOperation_APP_UPDATE,
			OpPayload: &common_proto.DCStream_AppDeployment{AppDeployment: appDeployment},
		}

		if err := p.deployApp.Publish(&event); err != nil {
			log.Println(err.Error())
			return &common_proto.Empty{}, ankr_default.ErrPublish
		}

		// TODO: wait deamon notify
		if err := p.db.UpdateApp(appDeployment); err != nil {
			log.Println(err.Error())
			return &common_proto.Empty{}, err
		}
	}

	return &common_proto.Empty{}, nil
}

func (p *AppMgrHandler) AppOverview(ctx context.Context, req *common_proto.Empty) (*appmgr.AppOverviewResponse, error) {
	log.Printf(">>>>>>>>>Debug into AppOverview, ctx: %+v\n", ctx)
	rsp := &appmgr.AppOverviewResponse{}
	userId := common_util.GetUserID(ctx)

	apps, err := p.db.GetAllApp(userId)
	if err != nil {
		log.Println(err.Error())
		return rsp, err
	}

	nss, err := p.db.GetAllNamespace(userId)
	if err != nil {
		log.Println(err.Error())
		return rsp, err
	}

	clusters := map[string]bool{}
	rsp.CpuTotal = 0
	rsp.MemTotal = 0
	rsp.StorageTotal = 0

	for i := 0; i < len(nss); i++ {
		clusters[nss[i].ClusterID] = true
		rsp.CpuTotal += float32(nss[i].CpuLimit)
		rsp.MemTotal += float32(nss[i].MemLimit)
		rsp.StorageTotal += float32(nss[i].StorageLimit)
	}

	rsp.CpuUsage = rsp.CpuTotal / 4
	rsp.MemUsage = rsp.MemTotal / 3
	rsp.StorageUsage = rsp.StorageTotal / 2
	rsp.ClusterCount = uint32(len(clusters))
	rsp.NamespaceCount = uint32(len(nss))
	rsp.NetworkCount = uint32(len(clusters))
	rsp.TotalAppCount = uint32(len(apps))

	return rsp, nil
}

func (p *AppMgrHandler) PurgeApp(ctx context.Context, req *appmgr.AppID) (*common_proto.Empty, error) {
	log.Printf(">>>>>>>>>Debug into PurgeApp, ctx: %+v\n", ctx)
	appRecord, err := p.db.GetApp(req.AppId)

	if err != nil {
		log.Printf(" PurgeApp for app id %s not found \n", req.AppId)
		return &common_proto.Empty{}, ankr_default.ErrAppNotExist
	}

	if appRecord.Hidden {
		log.Printf(" app id %s already purged \n", req.AppId)
		return &common_proto.Empty{}, ankr_default.ErrAlreadyPurged
	}

	if appRecord.Status != common_proto.AppStatus_APP_CANCELED {

		if _, err := p.CancelApp(ctx, req); err != nil {
			log.Printf(err.Error())
			return &common_proto.Empty{}, err
		}
	}

	log.Printf(" PurgeApp  %s \n", req.AppId)
	if err := p.db.Update("app", req.AppId, bson.M{"$set": bson.M{"hidden": true}}); err != nil {
		log.Printf(err.Error())
		return &common_proto.Empty{}, err
	}

	return &common_proto.Empty{}, nil
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

func checkNsId(userId, nsId string) error {
	if userId == "" {
		return ankr_default.ErrUserNotExist
	}

	if nsId == "" || nsId != userId {
		return errors.New("User does not own this namespace")
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

// UploadChart will upload chart file to the chartmuseum in user catalog under "/user/userID"
func (p *AppMgrHandler) UploadChart(ctx context.Context, req *appmgr.UploadChartRequest) (*common_proto.Empty, error) {

	log.Printf(">>>>>>>>>Debug into UploadCharts...%+v\nctx: %+v\n", req, ctx)

	uid := common_util.GetUserID(ctx)

	if len(req.ChartName) == 0 || len(req.ChartRepo) == 0 || len(req.ChartVer) == 0 || len(req.ChartFile) == 0 {
		log.Printf("invalid input, create failed.\n")
		return &common_proto.Empty{}, ankr_default.ErrInvalidInput
	}

	_, err := semver.NewVersion(req.ChartVer)
	if err != nil {
		return &common_proto.Empty{}, errors.New("chart version is not a valid Semantic Version")
	}

	query, err := http.Get(getChartURL(chartmuseumURL+"/api", uid, req.ChartRepo) + "/" + req.ChartName + "/" + req.ChartVer)
	if query.StatusCode == 200 {
		log.Printf("chart already exist, create failed.\n")
		return &common_proto.Empty{}, ankr_default.ErrChartAlreadyExist
	}

	loadedChart, err := chartutil.LoadArchive(bytes.NewReader(req.ChartFile))
	if err != nil {
		log.Printf("cannot load chart from tar file, %s \nerror: %s\n", req.ChartName, err.Error())
		return &common_proto.Empty{}, ankr_default.ErrCannotLoadChart
	}

	loadedChart.Metadata.Version = req.ChartVer
	loadedChart.Metadata.Name = req.ChartName

	dest, err := os.Getwd()
	if err != nil {
		log.Printf("cannot get chart outdir")
		return &common_proto.Empty{}, ankr_default.ErrCannotGetChartOutdir
	}
	log.Printf("save to outdir: %s\n", dest)
	tarballName, err := chartutil.Save(loadedChart, dest)
	if err == nil {
		log.Printf("Successfully packaged chart and saved it to: %s\n", tarballName)
	} else {
		log.Printf("Failed to save: %s", err)
		return &common_proto.Empty{}, ankr_default.ErrCannotGetChartOutdir
	}

	tarball, err := os.Open(tarballName)
	if err != nil {
		log.Printf("cannot open chart tar file")
		return &common_proto.Empty{}, ankr_default.ErrCannotGetChartTar
	}

	chartReq, err := http.NewRequest("POST", getChartURL(chartmuseumURL+"/api", uid, req.ChartRepo), tarball)
	if err != nil {
		log.Printf("cannot open chart tar file, %s \n", err.Error())
		return &common_proto.Empty{}, ankr_default.ErrCannotGetChartTar
	}

	chartRes, err := http.DefaultClient.Do(chartReq)
	if err != nil {
		log.Printf("cannot upload chart tar file, %s \n", err.Error())
		return &common_proto.Empty{}, ankr_default.ErrCannotUploadChartTar
	}

	message, _ := ioutil.ReadAll(chartRes.Body)
	defer chartRes.Body.Close()
	log.Printf(string(message))

	if err := os.Remove(tarballName); err != nil {
		log.Printf("delete temp chart tarball failed, %s \n", err.Error())
	}
	return &common_proto.Empty{}, nil
}

// SaveAsChart will upload new version chart to the chartmuseum with new values.yaml
func (p *AppMgrHandler) SaveAsChart(ctx context.Context, req *appmgr.SaveAsChartRequest) (*common_proto.Empty, error) {

	log.Printf(">>>>>>>>>Debug into SaveAsChart...%+v\n ctx: %+v\n", req, ctx)

	uid := common_util.GetUserID(ctx)

	if len(req.ChartName) == 0 || len(req.ChartRepo) == 0 || len(req.ChartVer) == 0 ||
		len(req.SaveName) == 0 || len(req.SaveRepo) == 0 || len(req.SaveVer) == 0 {
		log.Printf("invalid input: empty chart properties not accepted \n")
		return &common_proto.Empty{}, ankr_default.ErrEmptyChartProperties
	}

	_, err := semver.NewVersion(req.SaveVer)
	if err != nil {
		return &common_proto.Empty{}, errors.New("chart version is not a valid Semantic Version")
	}

	querySaveChart, err := http.Get(getChartURL(chartmuseumURL+"/api", uid,
		req.SaveRepo) + "/" + req.SaveName + "/" + req.SaveVer)

	if err != nil {
		log.Printf("cannot get chart %s from chartmuseum\nerror: %s\n", req.SaveName, err.Error())
		return &common_proto.Empty{}, ankr_default.ErrChartMuseumGet
	}
	if querySaveChart.StatusCode == 200 {
		log.Printf("invalid input: save chart already exist \n")
		return &common_proto.Empty{}, ankr_default.ErrSaveChartAlreadyExist
	}

	queryOriginalChart, err := http.Get(getChartURL(chartmuseumURL, uid,
		req.ChartRepo) + "/" + req.ChartName + "-" + req.ChartVer + ".tgz")
	if err != nil {
		log.Printf("cannot get chart %s from chartmuseum\nerror: %s\n", req.ChartName, err.Error())
		return &common_proto.Empty{}, ankr_default.ErrChartMuseumGet
	}
	if queryOriginalChart.StatusCode != 200 {
		log.Printf("invalid input: original chart not exist \n")
		return &common_proto.Empty{}, ankr_default.ErrOriginalChartNotExist
	}

	defer queryOriginalChart.Body.Close()

	loadedChart, err := chartutil.LoadArchive(queryOriginalChart.Body)
	if err != nil {
		log.Printf("cannot load chart from the http get response from chartmuseum , %s \nerror: %s\n",
			req.ChartName, err.Error())
		return &common_proto.Empty{}, ankr_default.ErrChartMuseumGet
	}

	loadedChart.Metadata.Version = req.SaveVer
	loadedChart.Metadata.Name = req.SaveName
	loadedChart.Values = &chart.Config{
		Raw: string(req.ValuesYaml),
	}

	dest, err := os.Getwd()
	if err != nil {
		log.Printf("cannot get chart outdir")
		return &common_proto.Empty{}, ankr_default.ErrCannotGetChartOutdir
	}
	log.Printf("save to outdir: %s\n", dest)
	tarballName, err := chartutil.Save(loadedChart, dest)
	if err == nil {
		log.Printf("Successfully packaged chart and saved it to: %s\n", tarballName)
	} else {
		log.Printf("Failed to save: %s", err)
		return &common_proto.Empty{}, ankr_default.ErrCannotGetChartOutdir
	}

	tarball, err := os.Open(tarballName)
	if err != nil {
		log.Printf("cannot open chart tar file")
		return &common_proto.Empty{}, ankr_default.ErrCannotGetChartTar
	}

	chartReq, err := http.NewRequest("POST", getChartURL(chartmuseumURL+"/api",
		uid, req.SaveRepo), tarball)
	if err != nil {
		log.Printf("cannot open chart tar file, %s \n", err.Error())
		return &common_proto.Empty{}, ankr_default.ErrCannotGetChartTar
	}

	chartRes, err := http.DefaultClient.Do(chartReq)
	if err != nil {
		log.Printf("cannot upload chart tar file, %s \n", err.Error())
		return &common_proto.Empty{}, ankr_default.ErrCannotUploadChartTar
	}

	message, _ := ioutil.ReadAll(chartRes.Body)
	defer chartRes.Body.Close()
	log.Printf(string(message))

	if err := os.Remove(tarballName); err != nil {
		log.Printf("delete temp chart tarball failed, %s \n", err.Error())
	}
	return &common_proto.Empty{}, nil
}

// ChartList will return a list of charts from the specific chartmuseum repo
func (p *AppMgrHandler) ChartList(ctx context.Context, req *appmgr.ChartListRequest) (*appmgr.ChartListResponse, error) {

	log.Printf(">>>>>>>>>Debug into ChartList...%+v\nctx: %+v\n", req, ctx)

	uid := common_util.GetUserID(ctx)
	rsp := &appmgr.ChartListResponse{}

	if len(req.ChartRepo) == 0 {
		req.ChartRepo = "stable"
	}
	chartRes, err := http.Get(getChartURL(chartmuseumURL+"/api", uid, req.ChartRepo))
	if err != nil {
		log.Printf("cannot get chart list, %s \n", err.Error())
		return rsp, ankr_default.ErrCannotGetChartList
	}

	defer chartRes.Body.Close()

	message, err := ioutil.ReadAll(chartRes.Body)
	if err != nil {
		log.Printf("cannot get chart list response body, %s \n", err.Error())
		return rsp, ankr_default.ErrCannotReadChartList
	}

	data := map[string][]Chart{}
	if err := json.Unmarshal([]byte(message), &data); err != nil {
		log.Printf("cannot unmarshal chart list, %s \n", err.Error())
		return rsp, ankr_default.ErrUnMarshalChartList
	}

	charts := make([]*common_proto.Chart, 0)

	for _, v := range data {
		chart := common_proto.Chart{
			ChartName:             v[0].Name,
			ChartRepo:             req.ChartRepo,
			ChartDescription:      v[0].Description,
			ChartIconUrl:          v[0].Icon,
			ChartLatestVersion:    v[0].Version,
			ChartLatestAppVersion: v[0].AppVersion,
		}
		charts = append(charts, &chart)
	}

	rsp.Charts = charts

	return rsp, nil
}

// ChartDetail will return a list of specific chart versions from the chartmuseum repo
func (p *AppMgrHandler) ChartDetail(ctx context.Context,
	req *appmgr.ChartDetailRequest) (*appmgr.ChartDetailResponse, error) {

	log.Printf(">>>>>>>>>Debug into ChartDetail... %+v\nctx: %+v\n", req, ctx)

	uid := common_util.GetUserID(ctx)
	rsp := &appmgr.ChartDetailResponse{}
	if req.Chart == nil || len(req.Chart.ChartName) == 0 || len(req.Chart.ChartRepo) == 0 {
		log.Printf("invalid input: null chart provided, %+v \n", req.Chart)
		return rsp, ankr_default.ErrChartNotExist
	}

	chartRes, err := http.Get(getChartURL(chartmuseumURL+"/api",
		uid, req.Chart.ChartRepo) + "/" + req.Chart.ChartName)
	if err != nil {
		log.Printf("cannot get chart details, %s \n", err.Error())
		return rsp, ankr_default.ErrChartDetailGet
	}

	defer chartRes.Body.Close()

	message, err := ioutil.ReadAll(chartRes.Body)
	if err != nil {
		log.Printf("cannot get chart details response body, %s \n", err.Error())
		return rsp, ankr_default.ErrCannotReadChartDetails
	}

	data := []Chart{}
	if err := json.Unmarshal([]byte(message), &data); err != nil {
		log.Printf("cannot unmarshal chart details, %s \n", err.Error())
		return rsp, ankr_default.ErrUnMarshalChartDetail
	}

	rsp.ChartName = req.Chart.ChartName
	rsp.ChartRepo = req.Chart.ChartRepo

	versionDetails := make([]*common_proto.ChartVersionDetail, 0)
	for _, v := range data {
		versiondetail := common_proto.ChartVersionDetail{
			ChartVer:    v.Version,
			ChartAppVer: v.AppVersion,
		}
		versionDetails = append(versionDetails, &versiondetail)
	}
	rsp.ChartVersionDetails = versionDetails

	tarballReq, err := http.NewRequest("GET", getChartURL(chartmuseumURL, uid,
		req.Chart.ChartRepo)+"/"+req.Chart.ChartName+"-"+req.ShowVersion+".tgz", nil)
	if err != nil {
		log.Printf("cannot create show version tarball request, %s \n", err.Error())
		return rsp, ankr_default.ErrCreateRequest
	}

	tarballRes, err := http.DefaultClient.Do(tarballReq)
	if err != nil {
		log.Printf("cannot download chart tarball, %s \n", err.Error())
		return rsp, errors.New(ankr_default.DialError + "Cannot download chart tarball" + err.Error())
	}
	defer tarballRes.Body.Close()

	gzf, err := gzip.NewReader(tarballRes.Body)
	if err != nil {
		log.Printf("cannot open chart tarball, %s \n", err.Error())
		return rsp, ankr_default.ErrCannotReadDownload
	}
	defer gzf.Close()

	tarball := tar.NewReader(gzf)
	tarf := make(map[string]string)
	tarf[req.Chart.ChartName+"/README.md"] = ""
	tarf[req.Chart.ChartName+"/values.yaml"] = ""

	if err := extractFromTarfile(tarf, tarball); err != nil {
		log.Printf("cannot find readme/value in chart tarball, %s \n", err.Error())
		return rsp, ankr_default.ErrNoChartReadme
	}
	if tarf[req.Chart.ChartName+"/README.md"] == "" {
		log.Printf("cannot find readme in chart tarball\n")
	}
	if tarf[req.Chart.ChartName+"/values.yaml"] == "" {
		log.Printf("cannot find value in chart tarball\n")
	}

	rsp.ReadmeMd = tarf[req.Chart.ChartName+"/README.md"]
	rsp.ValuesYaml = tarf[req.Chart.ChartName+"/values.yaml"]

	return rsp, nil
}

// DownloadChart will return a specific chart tarball from the specific chartmuseum repo
func (p *AppMgrHandler) DownloadChart(ctx context.Context,
	req *appmgr.DownloadChartRequest) (*appmgr.DownloadChartResponse, error) {

	log.Printf(">>>>>>>>>Debug into DownloadChart...%+v\nctx: %+v\n", req, ctx)

	uid := common_util.GetUserID(ctx)
	rsp := &appmgr.DownloadChartResponse{}
	if len(req.ChartName) == 0 || len(req.ChartRepo) == 0 || len(req.ChartVer) == 0 {
		log.Printf("invalid input: null chart detail provided, %+v \n", req)
		return rsp, ankr_default.ErrChartDetailEmpty
	}

	tarballReq, err := http.NewRequest("GET", getChartURL(chartmuseumURL,
		uid, req.ChartRepo)+"/"+req.ChartName+"-"+req.ChartVer+".tgz", nil)
	if err != nil {
		log.Printf("cannot create download tarball request, %s \n", err.Error())
		return rsp, ankr_default.ErrCreateRequest
	}

	tarballRes, err := http.DefaultClient.Do(tarballReq)
	if err != nil {
		log.Printf("cannot download chart tarball, %s \n", err.Error())
		return rsp, errors.New(ankr_default.DialError + "Cannot download chart tarball" + err.Error())
	}
	defer tarballRes.Body.Close()

	chartFile, err := ioutil.ReadAll(tarballRes.Body)
	if err != nil {
		log.Printf("cannot read chart tarball, %s \n", err.Error())
		return rsp, ankr_default.ErrCannotReadDownload
	}

	rsp.ChartFile = chartFile

	return rsp, nil
}

// DeleteChart delete a specific chart version from the specific chartmuseum repo
func (p *AppMgrHandler) DeleteChart(ctx context.Context,
	req *appmgr.DeleteChartRequest) (*common_proto.Empty, error) {

	log.Printf(">>>>>>>>>Debug into DeleteChart...%+v\nctx: %+v\n", req, ctx)

	uid := common_util.GetUserID(ctx)

	query, err := http.Get(getChartURL(chartmuseumURL+"/api", uid,
		req.ChartRepo) + "/" + req.ChartName + "/" + req.ChartVer)
	if query.StatusCode != 200 {
		log.Printf("chart not exist, delete failed.\n")
		return &common_proto.Empty{}, ankr_default.ErrChartNotExist
	}

	delReq, err := http.NewRequest("DELETE", getChartURL(chartmuseumURL+"/api",
		uid, req.ChartRepo)+"/"+req.ChartName+"/"+req.ChartVer, nil)
	if err != nil {
		log.Printf("cannot create delete chart request, %s \n", err.Error())
		return &common_proto.Empty{}, ankr_default.ErrCreateRequest
	}

	delRes, err := http.DefaultClient.Do(delReq)
	if err != nil {
		log.Printf("cannot delete chart file, %s \n", err.Error())
		return &common_proto.Empty{}, errors.New(ankr_default.LogicError + "Cannot delete chart file" + err.Error())
	}
	defer delRes.Body.Close()

	return &common_proto.Empty{}, nil
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

// CreateNamespace will create a namespace on cluster which is desinated by dcmgr
func (p *AppMgrHandler) CreateNamespace(ctx context.Context,
	req *appmgr.CreateNamespaceRequest) (*appmgr.CreateNamespaceResponse, error) {

	rsp := &appmgr.CreateNamespaceResponse{}
	uid := common_util.GetUserID(ctx)
	if len(uid) == 0 {
		return rsp, errors.New("user id not found in context")
	}
	log.Printf(">>>>>>>>>Debug into CreateNamespace: %+v\nctx: %+v\n", req, ctx)

	if req.Namespace == nil || req.Namespace.NsCpuLimit == 0 ||
		req.Namespace.NsMemLimit == 0 || req.Namespace.NsStorageLimit == 0 {
		log.Printf("invalid input: empty namespace properties not accepted \n")
		return rsp, ankr_default.ErrNsEmpty
	}

	if len(req.Namespace.ClusterId) > 0 {
		clusterConnection, err := p.db.GetClusterConnection(req.Namespace.ClusterId)
		if err != nil || clusterConnection.Status != common_proto.DCStatus_AVAILABLE {
			log.Println("cluster connection not available, namespace can not be created")
			return rsp, errors.New("cluster connection not available, namespace can not be created")
		}
	}

	req.Namespace.NsId = "ns-" + uuid.New().String()

	event := common_proto.DCStream{
		OpType:    common_proto.DCOperation_NS_CREATE,
		OpPayload: &common_proto.DCStream_Namespace{Namespace: req.Namespace},
	}

	if err := p.deployApp.Publish(&event); err != nil {
		log.Println(ankr_default.ErrPublish)
		return rsp, ankr_default.ErrPublish
	} else {
		log.Println("app manager service send CreateNamespace MQ message to dc manager service (api)")
	}

	if err := p.db.CreateNamespace(req.Namespace, uid); err != nil {
		log.Println(err.Error())
		return rsp, err
	}

	rsp.NsId = req.Namespace.NsId

	return rsp, nil
}

func convertFromNamespaceRecord(namespace db.NamespaceRecord) common_proto.NamespaceReport {
	message := common_proto.Namespace{}
	message.NsId = namespace.ID
	message.NsName = namespace.Name
	message.ClusterId = namespace.ClusterID
	message.ClusterName = namespace.ClusterName
	message.CreationDate = namespace.CreationDate
	message.LastModifiedDate = namespace.LastModifiedDate
	message.NsCpuLimit = namespace.CpuLimit
	message.NsMemLimit = namespace.MemLimit
	message.NsStorageLimit = namespace.StorageLimit
	namespaceReport := common_proto.NamespaceReport{
		Namespace: &message,
		NsEvent:   namespace.Event,
		NsStatus:  namespace.Status,
	}
	return namespaceReport
}

// NamespaceList will return a namespace list for certain user
func (p *AppMgrHandler) NamespaceList(ctx context.Context,
	req *common_proto.Empty) (*appmgr.NamespaceListResponse, error) {

	rsp := &appmgr.NamespaceListResponse{}
	userId := common_util.GetUserID(ctx)
	if len(userId) == 0 {
		return rsp, errors.New("user id not found in context")
	}
	log.Printf(">>>>>>>>>Debug into NamespaceList, ctx: %+v\n", ctx)

	namespaceRecords, err := p.db.GetAllNamespace(userId)
	log.Printf("NamespaceMessage  %+v \n", namespaceRecords)
	if err != nil {
		log.Println(err.Error())
		return rsp, err
	}

	namespacesWithoutCancel := make([]*common_proto.NamespaceReport, 0)

	for i := 0; i < len(namespaceRecords); i++ {

		if !namespaceRecords[i].Hidden && (namespaceRecords[i].Status == common_proto.NamespaceStatus_NS_CANCELED ||
			namespaceRecords[i].Status == common_proto.NamespaceStatus_NS_CANCELING) &&
			(namespaceRecords[i].LastModifiedDate == nil || namespaceRecords[i].LastModifiedDate.Seconds < (time.Now().Unix()-7200)) {
			namespaceRecords[i].Hidden = true
			p.db.Update("namespace", namespaceRecords[i].ID, bson.M{"$set": bson.M{"hidden": true,
				"lastmodifieddate": &timestamp.Timestamp{Seconds: time.Now().Unix()}}})
		}

		if !namespaceRecords[i].Hidden {
			namespaceMessage := convertFromNamespaceRecord(namespaceRecords[i])
			log.Printf("NamespaceMessage  %+v \n", namespaceMessage)
			clusterConnection, err := p.db.GetClusterConnection(namespaceRecords[i].ClusterID)
			if err != nil || clusterConnection.Status == common_proto.DCStatus_UNAVAILABLE {
				namespaceMessage.NsStatus = common_proto.NamespaceStatus_NS_UNAVAILABLE
				namespaceMessage.NsEvent = common_proto.NamespaceEvent_NS_HEARBEAT_FAILED
			}
			namespacesWithoutCancel = append(namespacesWithoutCancel, &namespaceMessage)
		}
	}

	rsp.NamespaceReports = namespacesWithoutCancel

	return rsp, nil
}

// UpdateNamespace will update a namespace with cpu/mem/storage limit
func (p *AppMgrHandler) UpdateNamespace(ctx context.Context,
	req *appmgr.UpdateNamespaceRequest) (*common_proto.Empty, error) {

	log.Printf(">>>>>>>>>Debug into UpdateNamespace: %+v\nctx: %+v\n", req, ctx)
	userId := common_util.GetUserID(ctx)

	if req.Namespace == nil || (req.Namespace.NsCpuLimit == 0 ||
		req.Namespace.NsMemLimit == 0 || req.Namespace.NsStorageLimit == 0) {
		log.Printf("invalid input: empty namespace properties not accepted \n")
		return &common_proto.Empty{}, ankr_default.ErrNsEmpty
	}

	namespaceRecord, err := p.db.GetNamespace(req.Namespace.NsId)
	if err != nil {
		log.Println(err.Error())
		return &common_proto.Empty{}, err
	}
	if err := checkNsId(userId, namespaceRecord.UID); err != nil {
		log.Println(err.Error())
		return &common_proto.Empty{}, err
	}

	if namespaceRecord.Status != common_proto.NamespaceStatus_NS_RUNNING &&
		namespaceRecord.Status != common_proto.NamespaceStatus_NS_UPDATE_FAILED {
		log.Println("namespace status is not running, cannot update")
		return &common_proto.Empty{}, ankr_default.ErrNSStatusCanNotUpdate
	}

	clusterConnection, err := p.db.GetClusterConnection(namespaceRecord.ClusterID)
	if err != nil || clusterConnection.Status != common_proto.DCStatus_AVAILABLE {
		log.Println("cluster connection not available, namespace can not be updated")
		return &common_proto.Empty{}, errors.New("cluster connection not available, namespace can not be updated")
	}

	namespaceReport := convertFromNamespaceRecord(namespaceRecord)

	if req.Namespace.NsCpuLimit > 0 && req.Namespace.NsMemLimit > 0 && req.Namespace.NsStorageLimit > 0 {
		namespaceReport.Namespace.NsCpuLimit = req.Namespace.NsCpuLimit
		namespaceReport.Namespace.NsMemLimit = req.Namespace.NsMemLimit
		namespaceReport.Namespace.NsStorageLimit = req.Namespace.NsStorageLimit
	}

	event := common_proto.DCStream{
		OpType:    common_proto.DCOperation_NS_UPDATE,
		OpPayload: &common_proto.DCStream_Namespace{Namespace: namespaceReport.Namespace},
	}

	if err := p.deployApp.Publish(&event); err != nil {
		log.Println(err.Error())
		return &common_proto.Empty{}, errors.New(ankr_default.PublishError + err.Error())
	}
	// TODO: wait deamon notify
	if err := p.db.UpdateNamespace(req.Namespace); err != nil {
		log.Printf("Err Status: %s, Err Message: %s", status.Code(err), err.Error())
		return &common_proto.Empty{}, err
	}
	return &common_proto.Empty{}, nil
}

// DeleteNamespace will delete a namespace with no resource owned
func (p *AppMgrHandler) DeleteNamespace(ctx context.Context,
	req *appmgr.DeleteNamespaceRequest) (*common_proto.Empty, error) {

	userId := common_util.GetUserID(ctx)
	log.Printf(">>>>>>>>>Debug into DeleteNamespace %+v\nctx: %+v\n", req, ctx)

	namespaceRecord, err := p.db.GetNamespace(req.NsId)
	if err != nil {
		log.Println(err.Error())
		return &common_proto.Empty{}, err
	}

	if err := checkNsId(userId, namespaceRecord.UID); err != nil {
		log.Println(err.Error())
		return &common_proto.Empty{}, err
	}

	if namespaceRecord.Status == common_proto.NamespaceStatus_NS_CANCELED {
		return &common_proto.Empty{}, ankr_default.ErrCanceledTwice
	}
	/*
		clusterConnection, err := p.db.GetClusterConnection(namespaceRecord.ClusterID)
		if err != nil || clusterConnection.Status != common_proto.DCStatus_AVAILABLE {
			log.Println("cluster connection not available, namespace can not be deleted")
			return errors.New("cluster connection not available, namespace can not be deleted")
		}
	*/
	apps, err := p.db.GetAllAppByNamespaceId(req.NsId)
	if err != nil {
		log.Println(err.Error())
		return &common_proto.Empty{}, err
	}
	for _, app := range apps {
		if app.Status != common_proto.AppStatus_APP_CANCELED &&
			app.Status != common_proto.AppStatus_APP_CANCELING &&
			app.Status != common_proto.AppStatus_APP_FAILED {
			return &common_proto.Empty{}, errors.New("namespace still got running app, can not delete.")
		}
	}

	namespaceReport := convertFromNamespaceRecord(namespaceRecord)

	event := common_proto.DCStream{
		OpType:    common_proto.DCOperation_NS_CANCEL,
		OpPayload: &common_proto.DCStream_Namespace{Namespace: namespaceReport.Namespace},
	}

	if err := p.deployApp.Publish(&event); err != nil {
		log.Println(err.Error())
		return &common_proto.Empty{}, errors.New(ankr_default.PublishError + err.Error())
	}

	if err := p.db.Update("namespace", req.NsId, bson.M{
		"$set": bson.M{"status": common_proto.NamespaceStatus_NS_CANCELING}}); err != nil {
		log.Println(err.Error())
		return &common_proto.Empty{}, err
	}

	return &common_proto.Empty{}, nil
}
