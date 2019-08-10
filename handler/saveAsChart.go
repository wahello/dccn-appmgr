package handler

import (
	ankr_default "github.com/Ankr-network/dccn-common/protos"
	appmgr "github.com/Ankr-network/dccn-common/protos/appmgr/v1/grpc"
	common_proto "github.com/Ankr-network/dccn-common/protos/common"
	common_util "github.com/Ankr-network/dccn-common/util"
	"github.com/Masterminds/semver"
	"io/ioutil"
	"k8s.io/helm/pkg/chartutil"
	"k8s.io/helm/pkg/proto/hapi/chart"
	"log"
	"net/http"
	"os"
	"context"
	"errors"
)

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