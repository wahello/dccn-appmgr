package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"

	"k8s.io/helm/pkg/chartutil"
)

func main() {

	path, err := filepath.Abs("nginx-ingress")
	if err != nil {
		fmt.Println(err)
		log.Fatalf("err: %v", err)
	}

	chart, err := chartutil.LoadDir(path)
	if err != nil {
		fmt.Println(err)
		log.Fatalf("err: %v", err)
	}

	if filepath.Base(path) != chart.Metadata.Name {
		fmt.Printf("directory name (%s) and Chart.yaml name (%s) must match", filepath.Base(path), chart.Metadata.Name)
	}

	dest, err := os.Getwd()
	if err != nil {
		fmt.Println(err)
		log.Fatalf("err: %v", err)
	}

	name, err := chartutil.Save(chart, dest)
	if err == nil {
		fmt.Printf("Successfully packaged chart and saved it to: %s\n", name)
	} else {
		fmt.Printf("Failed to save: %s", err)
	}

	file, err := os.Open(name)
	check(err)

	req, err := http.NewRequest("POST", "http://chart-dev.dccn.ankr.network:8080/api/public/nginx/charts", file)
	check(err)

	res, err := http.DefaultClient.Do(req)

	message, _ := ioutil.ReadAll(res.Body)
	defer res.Body.Close()

	fmt.Printf(string(message))
}

func check(err error) {
	if err != nil {
		panic(err)
	}
}
