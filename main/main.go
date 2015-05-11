package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"github.com/Sirupsen/logrus"
	"github.com/go-martini/martini"
	"log"
	"net/http"
	"os"
	"regexp"
	"strings"
)

var logger = logrus.New()
var rxExt = regexp.MustCompile(`(\.(?:xml|text|json))\/?$`)
var CacheContainer, _ = New(150) //缓存的容器

var ( //日志文件
	logFileName = flag.String("log", "server.log", "Log file name")
)

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

func dispatchContainer(w http.ResponseWriter, enc Encoder, r *http.Request) (int, string) {
	// 接受image-name，返回json（server-ip，server-port）
	// return "hello world " + param["word"]
	// var ca containerAddr
	var in imageName //the image name received
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		logger.Warnf("error decoding image: %s", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		fmt.Println("接收数据错误", err)
		return http.StatusBadRequest, ""
	}

	curClusterStat := GetCurrentClusterStatus()

	ip := ServerAndContainer(curClusterStat, in.ImageName)
	if ip.Status == 6 { //分配容器出错了

	} else if ip.Status == 3 { //使用了现有的容器
		fmt.Println("现有容器")
		name := getImageNameByContainerName("http://"+ip.Instance.ServerIP+":4243", ip.Instance.containerID)
		if !isServiceContainer(name["ImageName"]) {
			_, ok := CacheContainer.Get(ip.Instance)
			if ok == true { //成功放到最前面
				logger.Debug("成功")
			} else { //出错了
				logger.Errorln("未成功更新容器调用记录！")
			}
		}

	} else if ip.Status == 2 { //新创建的容器
		fmt.Println("新建容器")
		ip.Status = 3 //因为只接受3作为正确结果
		name := getImageNameByContainerName("http://"+ip.Instance.ServerIP+":4243", ip.Instance.containerID)
		if !isServiceContainer(name["ImageName"]) {
			CacheContainer.Add(ip.Instance.containerID, ip)
		}
	} else { //错误的数据
		logger.Errorln("无法解析的容器状态")
	}
	// CacheContainer.Add(ip.Instance.containerID, ip)
	// fmt.Println("分配的IP是", ip.Instance.ServerIP)
	// fmt.Println("当前id是", ip.Instance.containerID)
	fmt.Println("分配信息是", ip)

	log.Println("分配成功！请求容器是", in.ImageName, "分配结果是", ip)
	// RestrictContainer(curClusterStat)
	/*	t := evictElement(ip)
		if t != nil {
			fmt.Println("删除初出错", t)
		} else {
			fmt.Println("删除成功")
		}*/
	// t, _ := CacheContainer.Get(ip.Instance.containerID)
	// fmt.Println("当前缓存是", t)
	// CacheContainer.Add(, ip)
	// lruTest()
	// fmt.Println("执行了")
	/*	w.Header().Set("content-type", "application/json")
		w.WriteHeader(http.StatusOK)
		if err := json.NewEncoder(w).Encode(ip); err != nil {
			logger.Error(err)
		}*/
	return http.StatusOK, Must(enc.Encode(ip))
}
func MapEncoder(c martini.Context, w http.ResponseWriter, r *http.Request) {
	// Get the format extension
	matches := rxExt.FindStringSubmatch(r.URL.Path)
	ft := ".json"
	if len(matches) > 1 {
		// Rewrite the URL without the format extension
		l := len(r.URL.Path) - len(matches[1])
		if strings.HasSuffix(r.URL.Path, "/") {
			l--
		}
		r.URL.Path = r.URL.Path[:l]
		ft = matches[1]
	}
	// Inject the requested encoder
	switch ft {
	case ".xml":
		c.MapTo(xmlEncoder{}, (*Encoder)(nil))
		w.Header().Set("Content-Type", "application/xml")
	case ".text":
		c.MapTo(textEncoder{}, (*Encoder)(nil))
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	case ".html":
		c.MapTo(textEncoder{}, (*Encoder)(nil))
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	default:
		c.MapTo(jsonEncoder{}, (*Encoder)(nil))
		w.Header().Set("Content-Type", "application/json")
	}
}
func main() {

	//set logfile Stdout
	logFile, logErr := os.OpenFile(*logFileName, os.O_CREATE|os.O_RDWR|os.O_APPEND, 0666)
	if logErr != nil {
		fmt.Println("Fail to find", *logFile, "cServer start Failed")
		os.Exit(1)
	}
	log.SetOutput(logFile)
	// 默认会有log.Ldate | log.Ltime（日期 时间），这里重写为 日 时 文件名
	log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile) //2015/04/22 11:28:41 test.go:29: content

	go StartDeamon()
	go StartCacheDeamon()

	// fmt.Println("执行到我来")
	m := martini.Classic()
	m.Use(MapEncoder)
	m.Post("/api/dispatcher/v1.0/container/create", dispatchContainer)

	m.Run()

}
