package coderun_dispatcher

import (
	"encoding/json"
	"fmt"
	// "github.com/Sirupsen/logrus"
	// "github.com/antonholmquist/jason"
	"github.com/go-martini/martini"
	// "io/ioutil"
	"net/http"
	// "os"
	// "strconv"
	// "strings"
	// "strconv"
)

type imageName struct { // function dispatchContainer receive this parameter
	iName string
}

type machineUsage struct {
	CPUUsage string
	MemUsage string
}

func dispatchContainer(w http.ResponseWriter, r *http.Request) {
	// 接受image-name，返回json（server-ip，server-port）
	// return "hello world " + param["word"]
	// var ca containerAddr
	var in imageName //the image name received
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		logger.Warnf("error decoding image: %s", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	/*	out := containerAddr{
			ServerIP:   "456789",
			ServerPost: 32,
		}
		w.Header().Set("content-type", "application/json")
		w.WriteHeader(http.StatusOK)
		if err := json.NewEncoder(w).Encode(out); err != nil {
			logger.Error(err)
		}*/
	testContainerList := getValidContainerName("http://192.168.0.33:8080/api/v1.0/containers/docker")
	// fmt.Println("conter list is ", testContainerList)
	cs := getContainerStat("http://192.168.0.33:8080", testContainerList)
	fmt.Println("容器状态", cs)
	// fmt.Println("test is ", test)
	server := getInitialServerAddr()
	str := getServerStats(server) //服务器状态列表
	for _, v := range str {
		fmt.Println("str is ", v)
	}
	out := containerAddr{
		ServerIP:   "str",
		ServerPost: 32,
	}
	/*	out := containerAddr{
		ServerIP:   "456789",
		ServerPost: 32,
	}*/
	// logger.Debug("halog")
	// logger.Debug(str)
	// fmt.Println(str)

	w.Header().Set("content-type", "text/html")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(out); err != nil {
		logger.Error(err)
	}
	fmt.Println(str)

}

func main() {
	// getInitialServerInfo()
	m := martini.Classic()
	m.Post("/api/dispatcher/v1.0/container/create", dispatchContainer)

	m.Run()

}
