package main

import (
	"encoding/json"
	"fmt"
	"github.com/Sirupsen/logrus"
	"github.com/go-martini/martini"
	"net/http"
)

var logger = logrus.New()

var CacheContainer, _ = New(50)

type imageName struct { // function dispatchContainer receive this parameter
	ImageName string
}

type machineUsage struct {
	CPUUsage string
	MemUsage string
}

type containerAddr struct { // function dispatcherContainer will return this
	ServerIP    string
	ServerPort  int
	containerID string
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

	curClusterStat := GetCurrentClusterStatus()
	// ip := RR(curClusterStat)
	// ip := LCS(curClusterStat)
	// ip := ServerPriority(curClusterStat)
	ip := ServerAndContainer(curClusterStats, in.ImageName)
	fmt.Println("分配的IP是", ip)
	fmt.Println("当前状态是", len(curClusterStat))

	// CacheContainer.Add(, ip)

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
	createNewContainer("192.168.0.33", "test")
	//启动进程，检测并更新服务器状态
	go StartDeamon()

	CacheContainer.Add(1, "test")
	// fmt.Println("在此测试一下")
	m := martini.Classic()
	m.Post("/api/dispatcher/v1.0/container/create", dispatchContainer)

	m.Run()

}
