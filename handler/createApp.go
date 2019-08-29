package handler

import (
	"context"
	"errors"
	"log"
	"net/http"

	ankr_default "github.com/Ankr-network/dccn-common/protos"
	appmgr "github.com/Ankr-network/dccn-common/protos/appmgr/v1/grpc"
	common_proto "github.com/Ankr-network/dccn-common/protos/common"
	common_util "github.com/Ankr-network/dccn-common/util"
	"github.com/google/uuid"
	"k8s.io/helm/pkg/chartutil"
)

func (p *AppMgrHandler) CreateApp(ctx context.Context, req *appmgr.CreateAppRequest) (*appmgr.CreateAppResponse, error) {

	creator, teamId := common_util.GetUserIDAndTeamID(ctx)
	log.Printf(">>>>>>>>>Debug into CreateApp %+v \nctx: %+v , creator %s  team_id %s \n", req, ctx, creator, teamId)

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
		if err := p.db.CreateNamespace(appDeployment.Namespace, teamId, creator); err != nil {
			log.Println(err.Error())
			return rsp, err
		}
	}

	if req.App.ChartDetail == nil {
		log.Printf("invalid input: null chart detail provided, %+v \n", req.App)
		return rsp, ankr_default.ErrChartDetailEmpty
	}
	appDeployment.ChartDetail = req.App.ChartDetail
	appChart, err := http.Get(getChartURL(chartmuseumURL, teamId,
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
	appDeployment.ChartDetail.ChartDescription = loadedChart.Metadata.Description

	appDeployment.TeamId = teamId
	if req.App.CustomValues != nil {
		for _, customValue := range req.App.CustomValues {
			appDeployment.CustomValues = append(appDeployment.CustomValues, &common_proto.CustomValue{Key: "ankrCustomValues." + customValue.Key, Value: customValue.Value})
		}
	}

	event := common_proto.DCStream{
		OpType:    common_proto.DCOperation_APP_CREATE,
		OpPayload: &common_proto.DCStream_AppDeployment{AppDeployment: appDeployment},
	}

	if err := p.deployApp.Publish(&event); err != nil {
		log.Println(ankr_default.ErrPublish)
		return rsp, ankr_default.ErrPublish
	}
	log.Println("app manager service send CreateApp MQ message to dc manager service (api)")

	if err := p.db.CreateApp(appDeployment, teamId, creator); err != nil {
		log.Println(err.Error())
		return rsp, err
	}

	return rsp, nil
}
