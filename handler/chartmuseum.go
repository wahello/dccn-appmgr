package handler

import (
	"encoding/json"
	ankr_default "github.com/Ankr-network/dccn-common/protos"
	"io/ioutil"
	"log"
	"net/http"
)

func getCharts(teamId, repo string) (map[string][]Chart, error) {
	res := map[string][]Chart{}
	chartRes, err := http.Get(getChartURL(chartmuseumURL+"/api", teamId, repo))
	if err != nil {
		log.Printf("cannot get chart list, %v", err)
		return res, ankr_default.ErrCannotGetChartList
	}

	defer func() { _ = chartRes.Body.Close() }()

	message, err := ioutil.ReadAll(chartRes.Body)
	if err != nil {
		log.Printf("cannot get chart list response body, %v", err)
		return res, ankr_default.ErrCannotReadChartList
	}

	if err := json.Unmarshal([]byte(message), &res); err != nil {
		log.Printf("cannot unmarshal chart list, %s \n", err.Error())
		return res, ankr_default.ErrUnMarshalChartList
	}

	return res, nil
}

func getChartURL(url string, teamId string, repo string) string {

	if repo == "user" {
		url += "/user/" + teamId + "/charts"
	} else {
		url += "/public/" + repo + "/charts"
	}
	return url
}
