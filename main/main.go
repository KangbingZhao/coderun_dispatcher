package main

import (
	// "algorithm"
	"coderun_algo"
	"encoding/json"
	// "fmt"
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
	ServerPost int
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

	out := containerAddr{
		ServerIP:   "456789",
		ServerPost: 32,
	}

	w.Header().Set("content-type", "text/html")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(out); err != nil {
		logger.Error(err)
	}
	// fmt.Println("镜像名称是", in.ImageName)
	// fmt.Println("当前状态", coderun_alog.GetCurrentClusterStatus())
	/*	curClusterStat := coderun_alog.GetCurrentClusterStatus()
		ip := algorithm.RR(curClusterStat)*/
	// fmt.Println("分配的IP是", ip)
	// algorithm.RR()
	// fmt.Println("执行了")
}

func main() {
	// getInitialServerInfo()
	Test()
	go coderun_alog.StartDeamon()
	// fmt.Println("在此测试一下")
	m := martini.Classic()
	m.Post("/api/dispatcher/v1.0/container/create", dispatchContainer)

	m.Run()

}
