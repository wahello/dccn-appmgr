package handler

func getChartURL(url string, teamId string, repo string) string {

	if repo == "user" {
		url += "/user/" + teamId + "/charts"
	} else {
		url += "/public/" + repo + "/charts"
	}
	return url
}
