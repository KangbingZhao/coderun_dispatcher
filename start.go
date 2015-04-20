package main

import (
	"encoding/json"
	"fmt"
	"github.com/Sirupsen/logrus"
	"github.com/go-martini/martini"
	"io/ioutil"
	"net/http"
	// "net/url"
	"os"
	// "reflect"
	"github.com/antonholmquist/jason"
	"strconv"
	"strings"
	// "strconv"
)

var logger = logrus.New()

var ContainerMemCapacity = float64(20971520)

type containerAddr struct { // function dispatcherContainer will return this
	ServerIP   string
	ServerPost int
}
type imageName struct { // function dispatchContainer receive this parameter
	iName string
}

type serverConfig struct { // store data of ./metadata/config.json
	Server []struct {
		Host         string
		DockerPort   int
		CAdvisorPort int
	}
}

type machineUsage struct {
	CPUUsage string
	MemUsage string
}

func getInitialServerAddr() serverConfig { // get default server info from ./metadata/config.json
	r, err := os.Open("./metadata/config.json")
	if err != nil {
		logger.Error(err)
	}
	decoder := json.NewDecoder(r)
	var c serverConfig
	err = decoder.Decode(&c)
	if err != nil {
		logger.Error(err)
	}
	for k, v := range c.Server {
		fmt.Println(k, v.Host, v.DockerPort, v.CAdvisorPort)
	}
	return c

}

type serverStat struct {
	cpuUsage        float64 //百分比
	cpuFrequencyKHz int64   //kHz
	cpuCore         int64   //核心数,-1表示服务器不在线

	memUsageTotal float64 //内存容量，单位为Byte
	memUsageHot   float64 //当前活跃内存量
	memCapacity   float64 //内存总量
}

type containerStat struct {
	cpuUsage      float64 //percent
	memUsageTotal float64 //Byte
	memeUsageHot  float64
	memCapacity   float64
}

func subSubstring(str string, start, end int) string { //截取字符串
	if start < 0 {
		start = 0
	}
	if end > len(str) {
		end = len(str)
	}

	return string(str[start:end])

}

func getServerStats(serverList serverConfig) []serverStat { // get current server stat

	su := make([]serverStat, 0, 10)
	for index := 0; index < len(serverList.Server); index++ {
		var temp serverStat
		su = append(su, temp)
		if serverList.Server[index].Host == "" {
			su[index].cpuCore = -1
			continue
		}

		// su[index].CPUUsage = "CPU" + strconv.Itoa(index)
		// su[index].MemUsage = "Mem" + strconv.Itoa(index)

		cadvisorUrl := "http://" + serverList.Server[index].Host + ":" + strconv.Itoa(serverList.Server[index].CAdvisorPort)
		posturl := cadvisorUrl + "/api/v1.0/containers"

		reqContent := "{\"num_stats\":2,\"num_samples\":0}"
		body := ioutil.NopCloser(strings.NewReader(reqContent))
		client := &http.Client{}
		req, _ := http.NewRequest("POST", posturl, body)
		resq, _ := client.Do(req)
		defer resq.Body.Close()
		data, _ := ioutil.ReadAll(resq.Body)
		// fmt.Println(string(data), err)

		t, _ := jason.NewObjectFromBytes(data)
		stats, _ := t.GetObjectArray("stats") //从cAdvisor获取的最近两个stat,1是最新的
		// fmt.Println("len is ", len(stats))
		t1, _ := stats[1].GetString("timestamp")
		t2, _ := stats[0].GetString("timestamp")
		// fmt.Println("timestamp1 is ", t1, "the timestamp 2 is", t2)
		t1Time, _ := strconv.ParseFloat(subSubstring(t1, 17, 29), 64) //从秒开始，舍弃最后一个字母Z，不知道Z什么意思
		t2Time, _ := strconv.ParseFloat(subSubstring(t2, 17, 29), 64)
		// t2Time, _ := strconv.ParseFloat(subSubstring(t1, 17, 50), 64)
		// fmt.Println("t1 time is ", t1Time)
		intervalInNs := (t1Time - t2Time) * 1000000000 //单位是纳秒
		// fmt.Println("interval is ", intervalInNs)
		t1CPUUsage, _ := stats[1].GetFloat64("cpu", "usage", "total")
		t2CPUUsage, _ := stats[0].GetFloat64("cpu", "usage", "total")
		// fmt.Println("tiCPU is ", t1CPUUsage, t2CPUUsage)
		su[index].cpuUsage = (t1CPUUsage - t2CPUUsage) / intervalInNs
		/*		if su[index].cpuUsage < 0 { //不知道为什么是负数，处理一下
				su[index].cpuUsage = -su[index].cpuUsage
			}*/
		// fmt.Println("usage isw ", su[index].cpuUsage)
		memoryUsageTotal, _ := stats[1].GetFloat64("memory", "usage")
		memoryUsageWorking, _ := stats[1].GetFloat64("memory", "working_set")
		su[index].memUsageTotal = memoryUsageTotal
		su[index].memUsageHot = memoryUsageWorking

		posturl2 := cadvisorUrl + "/api/v1.0/machine"
		client2 := &http.Client{}
		req2, _ := http.NewRequest("POST", posturl2, nil)
		resq2, _ := client2.Do(req2)
		defer resq2.Body.Close()
		data2, _ := ioutil.ReadAll(resq2.Body)

		tt2, _ := jason.NewObjectFromBytes(data2)
		num_cores, _ := tt2.GetInt64("num_cores")
		cpu_frequency_khz, _ := tt2.GetInt64("cpu_frequency_khz")
		mem_capacity, _ := tt2.GetFloat64("memory_capacity")
		su[index].cpuCore = num_cores
		su[index].cpuFrequencyKHz = cpu_frequency_khz
		su[index].memCapacity = mem_capacity

	} // end of loop
	/*	for i := len(su) - 1; i >= 0; i-- { //删去corenum=-1的服务器
		if su[i].cpuCore == -1 {
			su = delete(su, su[i])
		}
	}*/
	return su

}

func getValidContainerName(url string) []string { //返回字符串为docker名字,如/docker/0a69438e4……
	// reqContent := "{\"num_stats\":2,\"num_samples\":0}"
	// body := ioutil.NopCloser(strings.NewReader(reqContent))
	client := &http.Client{}
	req, _ := http.NewRequest("POST", url, nil)
	resq, _ := client.Do(req)
	defer resq.Body.Close()
	data, _ := ioutil.ReadAll(resq.Body)

	dataDecoded, _ := jason.NewObjectFromBytes(data)
	containerList, _ := dataDecoded.GetObjectArray("subcontainers")
	// var containerNameList []string
	containerNameList := make([]string, 0, 50)
	for i, v := range containerList {
		containerNameList = append(containerNameList, "")
		containerNameList[i], _ = v.GetString("name")
	}
	// containerNameList,_
	// fmt.Println("list is ", containerNameList)
	// return containerNameList
	return containerNameList

}

func getContainerStat(serverUrl string, ContainerNameList []string) []containerStat { //serverUrl format is: http://server_ip:port
	cs := make([]containerStat, 0, 50)
	for index := 0; index < len(ContainerNameList); index++ {
		var temp containerStat
		cs = append(cs, temp)

		posturl := serverUrl + "/api/v1.0/containers" + ContainerNameList[index]
		// fmt.Println("posturl is ", posturl)
		// continue
		reqContent := "{\"num_stats\":2,\"num_samples\":0}"
		body := ioutil.NopCloser(strings.NewReader(reqContent))
		client := &http.Client{}
		req, _ := http.NewRequest("POST", posturl, body)
		resq, _ := client.Do(req)
		defer resq.Body.Close()
		data, _ := ioutil.ReadAll(resq.Body)

		// fmt.Println("test")
		t, _ := jason.NewObjectFromBytes(data)
		// fmt.Println("t是神马", data)
		stats, _ := t.GetObjectArray("stats") //从cAdvisor获取的最近两个stat,1是最新的
		// fmt.Println("len is ", len(stats))
		t1, _ := stats[1].GetString("timestamp")
		t2, _ := stats[0].GetString("timestamp")
		// fmt.Println("test")
		// fmt.Println("timestamp1 is ", t1, "the timestamp 2 is", t2)
		t1Time, _ := strconv.ParseFloat(subSubstring(t1, 17, 29), 64) //从秒开始，舍弃最后一个字母Z，不知道Z什么意思
		t2Time, _ := strconv.ParseFloat(subSubstring(t2, 17, 29), 64)
		// t2Time, _ := strconv.ParseFloat(subSubstring(t1, 17, 50), 64)
		// fmt.Println("t1 time is ", t1Time)
		intervalInNs := (t1Time - t2Time) * 1000000000 //单位是纳秒
		// fmt.Println("test")
		// fmt.Println("interval is ", intervalInNs)
		t1CPUUsage, _ := stats[1].GetFloat64("cpu", "usage", "total")
		t2CPUUsage, _ := stats[0].GetFloat64("cpu", "usage", "total")
		// fmt.Println("tiCPU is ", t1CPUUsage, t2CPUUsage)
		cs[index].cpuUsage = (t1CPUUsage - t2CPUUsage) / intervalInNs

		memoryUsageTotal, _ := stats[1].GetFloat64("memory", "usage")
		memoryUsageWorking, _ := stats[1].GetFloat64("memory", "working_set")

		cs[index].memUsageTotal = memoryUsageTotal
		cs[index].memeUsageHot = memoryUsageWorking
		cs[index].memCapacity = ContainerMemCapacity
		// fmt.Println("container cpu is ", cs[index].cpuUsage)

	} // end of loop
	// fmt.Println("containerstat is gotten")
	// fmt.Println("len is ", len(ContainerNameList))
	return cs

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
