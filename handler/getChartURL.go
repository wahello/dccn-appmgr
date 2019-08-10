package handler

func getChartURL(url string, uid string, repo string) string {

	if repo == "user" {
		url += "/user/" + uid + "/charts"
	} else {
		url += "/public/" + repo + "/charts"
	}
	return url
}
