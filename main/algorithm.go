/*
*	the algorithms for load balancing
 */
package main

import (
	"fmt"
	// "encoding/json"
	"github.com/antonholmquist/jason"
	"io/ioutil"
	"net/http"
	// "strconv"
	"errors"
	"log"
	// "math"
	// "strings"
	"time"
)

var callCount int
var CpuThreshold float64 = 0.9 // the threshold of overload
var MemThreshold float64 = 0.9

type reError struct { //函数返回的结果
	msg string
	err error
}

func isOverload(CpuUsage, MemUsage float64) bool {
	// cpu 大于 90% ，内存大于 90% 视为过载
	re := bool(false)
	if CpuUsage > CpuThreshold || MemUsage > MemThreshold {
		re = bool(true)
	}
	return re
}

func delaySecond(n time.Duration) {
	// func delaySecond(n int) {
	time.Sleep(n * time.Second)
}

func createNewContainer(serverIP string, imageName string) (containerAddr, reError) { //创建新的容器
	createURL := "http://" + serverIP + ":9090/createrunner/" + imageName

	for i := 0; i < 20; i++ { //发送创建容器的请求，最多10秒钟
		client := &http.Client{}
		req, reqError := http.NewRequest("POST", createURL, nil)
		log.Println("创建URL是", createURL)
		if reqError != nil {
			return containerAddr{"", 0, ""}, reError{"req初始化失败", reqError}
		}
		resq, req1Err := client.Do(req)
		if req1Err != nil {
			//发送失败,直接退出
			return containerAddr{"", 0, ""}, reError{"发送创建请求失败", req1Err}
		}
		data, err := ioutil.ReadAll(resq.Body)
		if err != nil {
			//数据读取失败，直接退出
			return containerAddr{"", 0, ""}, reError{"创建请求结果无法读取", err}
		}
		defer resq.Body.Close()
		createResult, errR := jason.NewObjectFromBytes(data)
		if errR != nil {
			return containerAddr{"", 0, ""}, reError{"创建请求结果解析出错", errR}
		}
		log.Println("返回结果是", createResult)
		createStatus, errS := createResult.GetInt64("status")
		if errS != nil {
			return containerAddr{"", 0, ""}, reError{"创建请求结果无法解析", errS}
		}
		if createStatus == 3 { //创建成功
			log.Println("创建成功!")
			containerHost, errCH := createResult.GetString("hosts")
			if errCH != nil {
				return containerAddr{"", 0, ""}, reError{"创建请求结果无法解析出主机地址", errCH}
			}
			containerInfo, errCI := createResult.GetObject("instances")
			if errCI != nil {
				return containerAddr{"", 0, ""}, reError{"创建请求结果无法解析出容器信息", errCI}
			}
			containerPort, errCP := containerInfo.GetInt64("port")
			if errCP != nil {
				return containerAddr{"", 0, ""}, reError{"创建请求结果无法解析出容器端口", errCP}
			}
			containerId, errCID := containerInfo.GetString("container_id")
			if errCID != nil {
				return containerAddr{"", 0, ""}, reError{"创建请求结果无法解析出容器ID", errCID}
			}

			return containerAddr{containerHost, int(containerPort), containerId}, reError{"", nil}

		} else if createStatus == 1 { //延迟后重新请求
			log.Println("延迟后重新请求")
			delaySecond(1)
			continue

		} else if createStatus == 2 { //重新调用创建api
			log.Println("重新调用,id是", imageName, "Status is ", createStatus)
			delaySecond(1)
			continue
		} else if createStatus == 6 { //pull失败
			log.Println("Pull失败，id是", imageName)
			return containerAddr{"", 0, ""}, reError{"无法获取镜像", errors.New("无法获取镜像")}
		}

	}

	//timeout

	return containerAddr{"", 0, ""}, reError{"创建镜像超时", errors.New("创建镜像超时")}

}

func AbandonedCreateNewContainer(serverIP string, imageName string) containerAddr { //创建新的人容器
	// fmt.Println("我是新建的函数")
	createURL := "http://" + serverIP + ":9090/createrunner/" + imageName
	// reqContent := ""
	client := &http.Client{}
	req, _ := http.NewRequest("POST", createURL, nil)
	resq, req1Err := client.Do(req)
	// fmt.Println("daowolw")
	if req1Err != nil {
		fmt.Println("发送创建请求失败")
		fmt.Println(createURL)
		return containerAddr{"", 0, ""}
	}
	data, _ := ioutil.ReadAll(resq.Body)
	// fmt.Println("daowolwsdfdsf")
	if data == nil {
		return containerAddr{"", 0, ""}
	}
	defer resq.Body.Close()
	/*	dataDecode,_ := jason.NewObjectFromBytes(data)
		createResult,_ := dataDecode.GetString("message")
		if createResult!= ""*/
	for i := 0; i < 10; i++ { //循环直到请求成功,否则超时

		findURL := "http://" + serverIP + ":9090/findrunner/" + imageName
		resq2, err := http.Get(findURL)
		if err != nil {
			//错误处理
		}

		data2, _ := ioutil.ReadAll(resq2.Body)
		defer resq2.Body.Close()
		findResult, _ := jason.NewObjectFromBytes(data2)

		containerHost, err1 := findResult.GetString("hosts")
		containerArray, err2 := findResult.GetObjectArray("instances")
		fmt.Println("地址是", findURL)
		fmt.Println("返回内容是", findResult)
		containerId, err3 := containerArray[len(containerArray)-1].GetString("container_id")
		containerPort, err4 := containerArray[len(containerArray)-1].GetInt64("port")
		// containerPort, err5 := strconv.Atoi(containerPort)

		if err1 == nil && err2 == nil && err3 == nil && err4 == nil {
			// fmt.Println("都是空啊")
			/*		fmt.Println("我是主机", containerHost)
					fmt.Println("我是ID", containerId)
					fmt.Println("我是信息", containerPort)*/
			// fmt.Println(err1, err2, err3, err4)
			fmt.Println("获取的id是", containerId)
			return containerAddr{containerHost, int(containerPort), containerId}
			break

		}
		delaySecond(1)

	}
	//超时了
	return containerAddr{"", 0, ""}

}

func RR(currentServerStatus []curServerStatus) containerAddr { // a Round-Robin
	// 直接按照服务器轮流新建容器,只需要返回服务器IP
	// fmt.Println("我来自算法啊")
	temp := callCount % len(currentServerStatus)
	callCount = temp + 1
	ip := currentServerStatus[temp].machineStatus.Host
	tc := containerAddr{ip, 0, ""}
	return tc

	// return currentServerStatus[temp].machineStatus.Host
}

func LCS(currentServerStatus []curServerStatus) containerAddr { // Lease-Connection Scheduling
	//选择当前运行容器最少的服务器，直接返回其IP
	var min int = 0
	// var minIP string
	serverNum := len(currentServerStatus) //当前在线的服务器数量
	if serverNum == 0 {                   //没有正常工作的服务器
		return containerAddr{"", 0, ""}
	} else if serverNum == 1 { //只有一台服务器
		return containerAddr{currentServerStatus[0].machineStatus.Host, 0, ""}
	} else {
		for i, v := range currentServerStatus {
			if len(v.containerStatus) > len(currentServerStatus[i+1].containerStatus) {
				min = i + 1
			}
			if i+1+1 == serverNum { // i+1是最后一个元素
				break
			}
		}
	}
	return containerAddr{currentServerStatus[min].machineStatus.Host, 0, ""}
}

func GetServerLoad(ss serverStat) float64 { //CPU和RAM使用率百分比的加权平均，暂定为0.5、0.5
	memUsage := ss.memUsageTotal / ss.memCapacity
	// log.Println("内存容器是", ss.memCapacity)
	// log.Println("内存用量是", ss.memUsageTotal)
	// log.Println("CPU用量是", ss.cpuUsage)
	if ss.cpuUsage > 0.9 || memUsage > 0.9 {
		return 1.0
	}
	return (ss.cpuUsage + memUsage) / 2
}

func GetContainerLoad(cs containerStat) float64 { //CPU和RAM使用率百分比的加权平均，暂定为0.5、0.5
	memUsage := cs.memUsageTotal / cs.memCapacity
	fmt.Println("内存占用率过高", cs.memUsageTotal)
	if cs.cpuUsage > 0.9 || memUsage > 0.9 {

		return 1.0
	}
	return (cs.cpuUsage + memUsage) / 2
}

func ServerPriority(currentServerStatus []curServerStatus) containerAddr { //选择负载最低的服务器，新建容器,直接返回服务器IP
	serverNum := len(currentServerStatus)
	if serverNum == 0 {
		return containerAddr{"", 0, ""}
	} else if serverNum == 1 {
		return containerAddr{currentServerStatus[0].machineStatus.Host, 0, ""}
	} else { // 两台以上服务器在线
		var temp int = 0
		for i, v := range currentServerStatus {
			if GetServerLoad(currentServerStatus[temp].machineStatus) > GetServerLoad(v.machineStatus) {
				temp = i
			}
		}
		return containerAddr{currentServerStatus[temp].machineStatus.Host, 0, ""}
	}
}

func findImagesInServer(currentServerCapacity ServerCapacity, imageName string) []int {
	// fmt.Println("当前容器", currentServerStatus.containerStatus)
	// 从一台服务器选出所有符合条件的容器，，按使用率从小到大排序后返回Slice
	re := make([]int, 0, 5)
	for i, v := range currentServerCapacity.containers {
		if imageName == v.imageName {
			re = append(re, i)
		}
	}
	// fmt.Println("执行zz")
	if len(re) < 2 {
		return re
	}
	// fmt.Println("执行xx")
	// fmt.Println("RE是", len(re))
	for i1 := 0; i1 < len(re)-1; i1++ { //对选出的容器，按使用率从小到大排序
		for i2 := 0; i2 < len(re)-i1-1; i2++ {
			// if GetContainerLoad(currentServerStatus.containerStatus[re[i1]]) > GetContainerLoad(currentServerStatus.containerStatus[re[i2]]) {
			if currentServerCapacity.containers[re[i1]].capacityLeft < currentServerCapacity.containers[re[i2]].capacityLeft {
				/*temp := i2
				i2 = i1
				i1 = temp*/
				temp := re[i2]
				re[i2] = re[i1]
				re[i1] = temp
				// fmt.Println("i1:i2", i1, " ", i2)
			}
		}
	}
	// fmt.Println("执行za")
	return re
}

// func sortServerByLoad(currentClusterCapacity []ServerCapacity) []ServerCapacity { //剩余容量从大到小
func sortServerByLoad() {

	serverNum := len(curClusterCapacity)
	if serverNum < 2 {
		return
	}

	re := curClusterCapacity
	for i1 := 0; i1 < len(re)-1; i1++ {
		// curClusterCapacity[0].l.Lock()
		for i2 := 0; i2 < len(re)-i1-1; i2++ {
			// if GetServerLoad(re[i2].machineStatus) > GetServerLoad(re[i2+1].machineStatus) {
			// temp := re[i2]
			// re[i2] = re[i2+1]
			// re[i2+1] = temp
			// if re[i2].
			if re[i2].CapacityLeft < re[i2+1].CapacityLeft {
				temp := re[i2]
				re[i2] = re[i2+1]
				re[i2+1] = temp
			}
		}
		// curClusterCapacity[0].l.Unlock()
	}
	// for _, v := range curClusterCapacity {
	// 	v.l.Lock()
	// }
	curClusterCapacity = re
	// for _, v := range curClusterCapacity {
	// 	v.l.Unlock()
	// }
}

func ServerAndContainer(imageName string) containerCreated { //优先选择已有的容器，容器及服务器都不过载则分配此容器，容器过载则重新分配容器；服务器过载则查找下一个服务器
	//status2表示容器创建成功，3表示分配现有的容器，6表示失败
	/*上述方案并不好，容器导致任务重的服务器负载越来越重，修改如下:
	*	先选择负载轻的服务器，在上面查找容器，选择负载最轻的容器分配(需要能够查出多个容器的函数)
	 */
	// sortedClusterCapacity := sortServerByLoad(currentClusterCapacity)
	sortServerByLoad()
	var re containerCreated
	for _, v := range curClusterCapacity {
		v.l.RLock()
		// fmt.Println("执行")
		//查找容器，找到且不过载则分配，找不到继续查找
		//循环结束后仍没有找到，则选择第一个（负载最轻的服务器分配）
		/*		if GetServerLoad(v.machineStatus) > 0.9 { //负载过高，不再分配
				continue
			}*/
		if v.CapacityLeft < DefaultContainerCapacify/10 { //负载高于90%
			v.l.RUnlock()
			continue
		}
		imageList := findImagesInServer(v, imageName)
		// fmt.Println("镜像名", imageName)
		// fmt.Println("列表", imageList)
		if len(imageList) == 0 { //不存在对应的容器
			// fmt.Println("执行22")
			log.Println("分配时没有找到容器", imageName)
			continue
		} else {
			//todo 选择第一个镜像进行分配,同时return
			/*			if GetContainerLoad(v.containerStatus[imageList[0]]) > 0.9 {
						log.Println("容器过载，不再分配此容器", v.containerStatus[imageList[0]].id)
						continue
					}*/
			if v.containers[imageList[0]].capacityLeft < DefaultContainerCapacify/20 {
				v.l.RUnlock()
				continue
			}

			re.Status = 3
			re.Instance.ServerIP = v.containers[imageList[0]].host
			re.Instance.ServerPort = v.containers[imageList[0]].port
			re.Instance.containerID = v.containers[imageList[0]].containerID
			v.l.RUnlock()
			v.l.Lock()
			v.CapacityLeft = v.CapacityLeft - 1
			v.containers[imageList[0]].capacityLeft = v.containers[imageList[0]].capacityLeft - 1
			v.l.Unlock()
			return re
		}
	}
	//执行到这里说明没有找到镜像，返回第一台服务器的ip即可,也就是当前负载最低的服务器
	log.Println("没有合适的容器需要创建")
	// log.Println("sort长度是", len(sortedServerStatus))
	curClusterCapacity[0].l.RLock()
	ServerIP := curClusterCapacity[0].host
	curClusterCapacity[0].l.RUnlock()
	temp, err := createNewContainer(ServerIP, imageName)

	var newContainer ContainerCapacity //添加新容器
	newContainer.capacityLeft = DefaultContainerCapacify - 1
	newContainer.containerID = temp.containerID
	newContainer.host = ServerIP
	newContainer.imageName = imageName
	newContainer.port = temp.ServerPort

	curClusterCapacity[0].l.Lock()
	curClusterCapacity[0].CapacityLeft = curClusterCapacity[0].CapacityLeft - 1
	curClusterCapacity[0].containers = append(curClusterCapacity[0].containers, newContainer)
	curClusterCapacity[0].l.Unlock()
	if err.err != nil { //出错
		// re := containerCreated{6, {"", temp, 0}}
		re.Status = 6
		re.Instance = temp
		log.Println("错误是", err.err)
		return re
	} else { //正确
		// re := containerCreated{3, {temp.ServerIP, temp.ServerPort}}
		re.Status = 2
		re.Instance = temp
		log.Println("创建容器成功", re.Instance)
		return re
	}

	// return temp
	// return createNewContainer(sortedServerStatus[0].machineStatus.Host, imageName)

	// return containerAddr{sortedServerStatus[0].machineStatus.Host, 0, ""}

}
