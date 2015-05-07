/*
*	get the stats of the servers and containers
 */

package main

import (
	"encoding/json"
	"fmt"
	// "github.com/Sirupsen/logrus"
	"github.com/antonholmquist/jason"
	"github.com/fsouza/go-dockerclient"
	"io/ioutil"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

// var logger = logrus.New()
var ContainerMemCapacity = float64(20971520) //default container memory capacity is 20MB

var curClusterStats = make([]curServerStatus, 0, 5) //the global variable is current server stats and container stats

var curClusterLoad float64

type curServerStatus struct {
	machineStatus   serverStat      //the server stats
	containerStatus []containerStat // the container stats of this server

}

/*type containerAddr struct { // function dispatcherContainer will return this
	ServerIP   string
	ServerPost int
}*/
type serverConfig struct { // store data of ./metadata/config.json
	Server []struct {
		Host         string
		DockerPort   int
		CAdvisorPort int
	}
}

type serverAddress struct {
	Host         string
	DockerPort   int
	CAdvisorPort int
}

func getInitialServerAddr() serverConfig { // get default server info from ./metadata/config.json
	r, err := os.Open("../metadata/config.json")
	if err != nil {
		logger.Error(err)
	}
	decoder := json.NewDecoder(r)
	var c serverConfig
	err = decoder.Decode(&c)
	if err != nil {
		logger.Error(err)
	}
	return c

}

type serverStat struct {
	Host         string
	DockerPort   int
	CAdvisorPort int

	cpuUsage        float64 //百分比
	cpuFrequencyKHz int64   //kHz
	cpuCore         int64   //核心数,-1表示服务器不在线

	memUsageTotal float64 //内存容量，单位为Byte
	memUsageHot   float64 //当前活跃内存量
	memCapacity   float64 //内存总量
}

type containerStat struct {
	serverIP      string
	name          string  //image name
	id            string  //container id
	port          int     //暴露在外的端口
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

/*func serverStatSliceRemove(slice []serverStat, start, end int) []serverStat {
	return append(slice[:start], slice[end:]...)
}
func serverStatSliceRemoveAtIndex(slice []serverStat, index int) []serverStat {
	if index+1 >= len(slice) { // 末尾
		return slice[:index-1]
	} else if index-1 < 0 { //开头
		return slice[1:]
	} else {
		return serverStatSliceRemove(slice, index-1, index+1)
	}
}*/

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
		su[index].Host = serverList.Server[index].Host
		su[index].CAdvisorPort = serverList.Server[index].CAdvisorPort
		su[index].DockerPort = serverList.Server[index].DockerPort

	} // end of loop

	var temp []serverStat
	for _, v := range su {
		if v.cpuCore == -1 {
			continue
		}
		temp = append(temp, v)
	}
	return temp

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
func getImageNameByContainerName(serverUrl string, containerName string) map[string]string { //返回内容为:镜像名称，容器对外端口
	// http://ip:port， docker/id
	id := subSubstring(containerName, 8, 100) //猜测不会超过100个字符，实际等同于从第8个字符开始截取
	// fmt.Println("name is ", containerName)
	// fmt.Println("id is ", id)
	client, _ := docker.NewClient(serverUrl)
	// imageName,_ := client.InspectContainer
	containerInfo, err := client.InspectContainer(id)
	if err != nil {
		return make(map[string]string)
	}
	fmt.Println("imgs is", containerInfo.Config.Image)
	temp := containerInfo.NetworkSettings.Ports["8080/tcp"]
	if len(temp) == 0 {
		re := map[string]string{
			"ImageName":   containerInfo.Config.Image,
			"ExpostdPort": "",
		}
		return re
	}
	re := map[string]string{
		"ImageName":   containerInfo.Config.Image,
		"ExpostdPort": temp[0].HostPort,
		// containerInfo.NetworkSettings.Ports["8080/tcp"]["HostPort"]
	}

	// fmt.Println("怎么", temp[0].HostPort)
	return re
}
func getContainerStat(serverIP string, cadvisorPort int, dockerPort int, ContainerNameList []string) []containerStat {
	//serverUrl format is: http://server_ip:port
	cs := make([]containerStat, 0, 50)
	serverUrl := "http://" + serverIP + ":" + strconv.Itoa(cadvisorPort)
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
		cs[index].serverIP = serverIP
		cs[index].id = subSubstring(ContainerNameList[index], 8, 20)
		// cs[index].name = getImageNameByContainerName(cs[index].serverAddr, ContainerNameList[index])
		iif := getImageNameByContainerName("http://"+cs[index].serverIP+":"+"4243", ContainerNameList[index])
		cs[index].name = iif["ImageName"]
		tempPort, _ := (strconv.Atoi(iif["ExpostdPort"]))
		cs[index].port = int(tempPort)
		fmt.Println("serverip is ", cs[index].serverIP)
		fmt.Println("image name is ", iif["ImageName"])
		fmt.Println("container id is ", cs[index].id)
		fmt.Println("container port is ", cs[index].port)
		// cs[index].serverAddr = serverUrl
		// fmt.Println("container cpu is ", cs[index].cpuUsage)

	} // end of loop
	// fmt.Println("containerstat is gotten")
	// fmt.Println("len is ", len(ContainerNameList))
	return cs

}

func GetCurrentClusterStatus() []curServerStatus { // return current curClusterStatus
	return curClusterStats
}

func StartDeamon() { // load the initial server info from ./metadata/config.json
	// and update server and container status periodicly
	servers := getInitialServerAddr()

	// fmt.Println("servers is ", servers)
	// getImageNameByContainerName("http://192.168.0.33:4243", "/docker/0a69438e4d780629c9c8ef2b672d9aea03ccaf1b7b56dd97458174e59e47618c")
	timeSlot := time.NewTimer(time.Second * 1) // update status every second
	for {
		select {
		case <-timeSlot.C:
			//TODO the codes to update
			// fmt.Println(serverSStats)
			serverSStats := getServerStats(servers) //因为服务器可能 发生 在线\不在线的变化
			tempClusterStats := curClusterStats[0:0]

			for index := 0; index < len(serverSStats); index++ {
				var temp curServerStatus
				tempClusterStats = append(tempClusterStats, temp)
				if serverSStats[index].cpuCore == -1 {
					tempClusterStats[index].machineStatus.cpuCore = -1
					continue
				}
				//获取服务器状态
				// cs := getServerStats(serverSStats[index].Host + strconv.Itoa(serverSStats[index].CAdvisorPort))
				var tempServerConfig serverConfig
				var tempServerAddress serverAddress
				// tempServerConfig.Server = append(tempServerConfig.Server, tempServerAddress)
				tempServerConfig.Server = append(tempServerConfig.Server, tempServerAddress)
				// fmt.Println("长度是 ", len(tempServerConfig.Server))

				tempServerConfig.Server[0].Host = string(serverSStats[index].Host)
				tempServerConfig.Server[0].CAdvisorPort = int(serverSStats[index].CAdvisorPort)
				tempServerConfig.Server[0].DockerPort = int(serverSStats[index].DockerPort)

				var ss []serverStat
				ss = getServerStats(tempServerConfig)
				tempClusterStats[index].machineStatus = ss[0]

				//获取容器名字
				serverUrl := "http://" + serverSStats[index].Host + ":" + strconv.Itoa(serverSStats[index].CAdvisorPort) + "/api/v1.0/containers/docker"
				// fmt.Println("url is ", serverUrl)
				containerNames := getValidContainerName(serverUrl)
				// fmt.Println("container names is", containerNames)
				// cs := getContainerStat("http://"+serverSStats[index].Host+":"+strconv.Itoa(serverSStats[index].CAdvisorPort), containerNames)
				cs := getContainerStat(serverSStats[index].Host, serverSStats[index].CAdvisorPort, serverSStats[index].DockerPort, containerNames)
				// fmt.Println("containers is ", cs)
				tempClusterStats[index].containerStatus = append(tempClusterStats[index].containerStatus, cs...)

			}
			// return
			curClusterStats = curClusterStats[0:0]
			curClusterStats = append(curClusterStats, tempClusterStats...)

			// fmt.Println("当前状态是 ", curClusterStats)
			timeSlot.Reset(time.Second * 100)
		}
	}

}

func loadCurrentContainer() { //初始化时，将现有容器放入缓存区中
	delaySecond(5)
	for _, v := range curClusterStats {
		if v.machineStatus.cpuCore == -1 {
			continue
		}
		for _, vv := range v.containerStatus {
			var temp containerCreated
			temp.Status = 100 //开始时就装入的
			temp.Instance.ServerIP = vv.serverIP
			temp.Instance.ServerPort = vv.port
			temp.Instance.containerID = vv.id
			CacheContainer.Add(vv.id, temp)
		}
	}
	// fmt.Println("状态是", GetCurrentClusterStatus())
	t := CacheContainer.Keys()
	for _, v := range t {
		tt, _ := CacheContainer.Get(v)
		fmt.Println("值是", tt)
	}
	// return nil
}
