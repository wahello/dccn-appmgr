package handler

import (
	"bytes"
	ankr_default "github.com/Ankr-network/dccn-common/protos"
	appmgr "github.com/Ankr-network/dccn-common/protos/appmgr/v1/grpc"
	common_proto "github.com/Ankr-network/dccn-common/protos/common"
	common_util "github.com/Ankr-network/dccn-common/util"
	"github.com/Masterminds/semver"
	"io/ioutil"
	"k8s.io/helm/pkg/chartutil"
	"log"
	"net/http"
	"os"
	"errors"
	"context"
)

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
