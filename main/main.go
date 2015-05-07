package main

import (
	"encoding/json"
	"fmt"
	"github.com/Sirupsen/logrus"
	"github.com/go-martini/martini"
	"net/http"
)

var logger = logrus.New()

var CacheContainer, _ = New(150)

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
	containerID string //小写开头，并没有返回给用户
}

type containerCreated struct {
	Status   int
	Instance containerAddr
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
	// fmt.Println("集群状态是啥啊", curClusterStat)
	// ip := RR(curClusterStat)
	// ip := LCS(curClusterStat)
	// ip := ServerPriority(curClusterStat)
	ip := ServerAndContainer(curClusterStat, in.ImageName)
	if ip.Status == 6 { //分配容器出错了

	} else if ip.Status == 2 { //新创建的容器
		CacheContainer.Add(ip.Instance.containerID, ip)
	} else if ip.Status == 3 { //使用了现有的容器
		_, ok := CacheContainer.Get(ip.Instance)
		if ok == true { //成功放到最前面
			logger.Debug("成功")
		} else { //出错了
			logger.Errorln("未成功更新容器调用记录！")
		}
	} else { //错误的数据
		logger.Errorln("无法解析的容器状态")
	}
	// CacheContainer.Add(ip.Instance.containerID, ip)
	fmt.Println("分配的IP是", ip.Instance.ServerIP)
	fmt.Println("当前id是", ip.Instance.containerID)
	// t, _ := CacheContainer.Get(ip.Instance.containerID)
	// fmt.Println("当前缓存是", t)
	// CacheContainer.Add(, ip)
	// lruTest()
	// fmt.Println("执行了")
	w.Header().Set("content-type", "text/html")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(ip); err != nil {
		logger.Error(err)
	}
}

/*func lruTest() {
	CacheContainer.Add("ha", "test content")
	text, err := CacheContainer.Get("ha")
	fmt.Println("状态是", err, "内容是", text)
}*/

func main() {
	// getInitialServerInfo()
	// Test()
	// createNewContainer("192.168.0.33", "test")
	//启动进程，检测并更新服务器状态
	go StartDeamon()

	// delaySecond(1)
	fmt.Println("到了")
	loadCurrentContainer()
	// CacheContainer.Add(1, "test")
	// fmt.Println("在此测试一下")
	m := martini.Classic()
	m.Post("/api/dispatcher/v1.0/container/create", dispatchContainer)
	// m.Post("/api/dispatcher/v1.0/lru/test", lruTest)

	m.Run()

}
