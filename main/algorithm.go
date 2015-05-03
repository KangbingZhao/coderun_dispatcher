/*
*	the algorithms for load balancing
 */
package main

import (
// "fmt"
// "encoding/json"
)

var callCount int
var CpuThreshold float64 = 0.9 // the threshold of overload
var MemThreshold float64 = 0.9

func isOverload(CpuUsage, MemUsage float64) bool {
	// cpu 大于 90% ，内存大于 90% 视为过载
	re := bool(false)
	if CpuUsage > CpuThreshold || MemUsage > MemThreshold {
		re = bool(true)
	}
	return re
}

func RR(currentServerStatus []curServerStatus) containerAddr { // a Round-Robin
	// 直接按照服务器轮流新建容器,只需要返回服务器IP
	// fmt.Println("我来自算法啊")
	temp := callCount % len(currentServerStatus)
	callCount = temp + 1
	ip := currentServerStatus[temp].machineStatus.Host
	tc := containerAddr{ip, 0}
	return tc

	// return currentServerStatus[temp].machineStatus.Host
}

func LCS(currentServerStatus []curServerStatus) containerAddr { // Lease-Connection Scheduling
	//选择当前运行容器最少的服务器，直接返回其IP
	var min int = 0
	// var minIP string
	serverNum := len(currentServerStatus) //当前在线的服务器数量
	if serverNum == 0 {                   //没有正常工作的服务器
		return containerAddr{"", 0}
	} else if serverNum == 1 { //只有一台服务器
		return containerAddr{currentServerStatus[0].machineStatus.Host, 0}
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
	return containerAddr{currentServerStatus[min].machineStatus.Host, 0}
}

func GetServerLoad(ss serverStat) float64 { //CPU和RAM使用率百分比的加权平均，暂定为0.5、0.5
	memUsage := ss.memUsageTotal / ss.memCapacity
	return (ss.cpuUsage + memUsage) / 2
}

func GetContainerLoad(cs containerStat) float64 { //CPU和RAM使用率百分比的加权平均，暂定为0.5、0.5
	memUsage := cs.memUsageTotal / cs.memCapacity
	return (cs.cpuUsage + memUsage) / 2
}

func ServerPriority(currentServerStatus []curServerStatus) containerAddr { //选择负载最低的服务器，新建容器,直接返回服务器IP
	serverNum := len(currentServerStatus)
	if serverNum == 0 {
		return containerAddr{"", 0}
	} else if serverNum == 1 {
		return containerAddr{currentServerStatus[0].machineStatus.Host, 0}
	} else { // 两台以上服务器在线
		var temp int = 0
		for i, v := range currentServerStatus {
			if GetServerLoad(currentServerStatus[temp].machineStatus) > GetServerLoad(v.machineStatus) {
				temp = i
			}
		}
		return containerAddr{currentServerStatus[temp].machineStatus.Host, 0}
	}
}

func findImagesInServer(currentServerStatus curServerStatus, imageName string) []int {
	// 从一台服务器选出所有符合条件的容器，并返回Slice
	re := make([]int, 0, 5)
	for i, v := range currentServerStatus.containerStatus {
		if imageName == v.name {
			re = append(re, i)
		}
	}
	return re
}

func sortServerByLoad(currentServerStatus []curServerStatus) []curServerStatus { //根据负载从轻到重排序,返回数组
	serverNum := len(currentServerStatus)
	if serverNum < 2 {
		return currentServerStatus
	}
	re := currentServerStatus
	for i1 := 0; i1 < len(re)-1; i1++ {
		for i2 := 0; i2 < len(currentServerStatus)-i1-1; i2++ {
			if GetServerLoad(re[i2].machineStatus) > GetServerLoad(re[i2+1].machineStatus) {
				temp := re[i2]
				re[i2] = re[i2+1]
				re[i2+1] = temp
			}
		}
	}
	return re
}

func ServerAndContainer(currentServerStatus []curServerStatus, imageName string) containerAddr { //优先选择已有的容器，容器及服务器都不过载则分配此容器，容器过载则重新分配容器；服务器过载则查找下一个服务器
	/*上述方案并不好，容器导致任务重的服务器负载越来越重，修改如下:
	*	先选择负载轻的服务器，在上面查找容器，选择负载最轻的容器分配(需要能够查出多个容器的函数)
	 */
	sortedServerStatus := sortServerByLoad(currentServerStatus)
	for _, v := range sortedServerStatus {
		//查找容器，找到且不过载则分配，找不到继续查找
		//循环结束后仍没有找到，则选择第一个（负载最轻的服务器分配）
		imageList := findImagesInServer(v, imageName)
		if len(imageList) == 0 { //不存在对应的容器
			continue
		} else {
			//todo 选择第一个镜像进行分配,同时return
			return containerAddr{v.containerStatus[imageList[0]].serverIP, v.containerStatus[imageList[0]].port}

		}
	}
	//执行到这里说明没有找到镜像，返回第一台服务器的ip即可
	return containerAddr{currentServerStatus[0].machineStatus.Host, 0}

}
