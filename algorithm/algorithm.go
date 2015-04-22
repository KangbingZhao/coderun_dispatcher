package algorithm

import (
// "fmt"
)

// var curClusterStatus []curServerStatus

type curServerStatus struct {
	machineStatus   serverStat      //the server stats
	containerStatus []containerStat // the container stats of this server

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
	port          int     //暴露在外的端口
	cpuUsage      float64 //percent
	memUsageTotal float64 //Byte
	memeUsageHot  float64
	memCapacity   float64
}

var callCount int = 0

func findImageLocation() {

}

func RR(currentServerStatus []curServerStatus) string { // a Round-Robin
	// 直接按照服务器轮流新建容器,只需要返回服务器IP
	// fmt.Println("我来自算法啊")
	temp := callCount % len(currentServerStatus)
	if callCount > 20000 {
		callCount = temp
	}
	return currentServerStatus[temp].machineStatus.Host
}
