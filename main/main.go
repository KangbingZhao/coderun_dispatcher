package main

import (
	"encoding/json"
	"fmt"
	"github.com/Sirupsen/logrus"
	"github.com/go-martini/martini"
	"net/http"
)

type imageName struct { // function dispatchContainer receive this parameter
	ImageName string
}

type machineUsage struct {
	CPUUsage string
	MemUsage string
}

var logger = logrus.New()

type containerAddr struct { // function dispatcherContainer will return this
	ServerIP   string
	ServerPort int
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
	}*/

	// fmt.Println("镜像名称是", in.ImageName)
	// fmt.Println("当前状态", coderun_alog.GetCurrentClusterStatus())
	curClusterStat := GetCurrentClusterStatus()
	ip := RR(curClusterStat)
	// ip := LCS(curClusterStat)
	// ip := ServerPriority(curClusterStat)
	fmt.Println("分配的IP是", ip)
	fmt.Println("当前状态是", len(curClusterStat))
	// fmt.Println("执行了")
	w.Header().Set("content-type", "text/html")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(ip); err != nil {
		logger.Error(err)
	}
}

func main() {
	// getInitialServerInfo()
	// Test()
	go StartDeamon()
	// fmt.Println("在此测试一下")
	m := martini.Classic()
	m.Post("/api/dispatcher/v1.0/container/create", dispatchContainer)

	m.Run()

}
