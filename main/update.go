package main

import (
	"encoding/json"
	"errors"
	"fmt"
	// "github.com/Sirupsen/logrus"
	// "github.com/antonholmquist/jason"
	// "github.com/fsouza/go-dockerclient"
	// "io/ioutil"
	"log"
	"math"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

var DefaultContainerCapacify int = 50       //一个完全空闲容器能承担的用户数
var DefaultServerCapacity int = 800         //一个完全空闲容器能承担的用户数
var ServerMemCapacity = float64(2147483648) //2*1024*1024*1024 byte
type updateInfo struct {                    //服务器主动发送的机器信息
	Host       string
	Cpu        float64
	Mem        float64
	Containers []struct {
		Image string
		Id    string
		Cpu   float64
		Mem   float64
		Port  string
	}
}
type serverConfig struct { // store data of ./metadata/config.json
	Server []struct {
		Host         string
		DockerPort   int
		CAdvisorPort int
	}
}

//对服务器状态和处理能力分离之后添加的数据结构
var UpdateInfoChannel = make(chan updateInfo, 100) //缓冲区大小100，存放更新的服务器状态
type ContainerCapacity struct {
	updateTime   time.Time
	host         string
	port         int
	containerID  string
	imageName    string
	capacityLeft int
}
type ServerCapacity struct {
	updateTime   time.Time
	l            sync.RWMutex
	host         string
	CapacityLeft int
	containers   []ContainerCapacity
}

var ContainerMemCapacity = float64(20971520) //default container memory capacity is 20MB
var curClusterCapacity = make([]ServerCapacity, 0, 5)
var ClusterCapacityLock sync.RWMutex

func getInitialServerAddrInUpdate() serverConfig { // get default server info from ./metadata/config.json
	r, err := os.Open("../metadata/config.json")
	if err != nil {
		log.Println(err)
	}
	decoder := json.NewDecoder(r)
	var c serverConfig
	err = decoder.Decode(&c)
	if err != nil {
		log.Println(err)
	}
	// fmt.Println("初始服务器地址是", c)
	return c

}

func FindServerByHostInUpdate(hostip string) (int, error) {
	// curCluster := GetCurrentClusterStatus()
	// l.RLock()
	// defer l.RUnlock()
	for i, v := range curClusterCapacity {
		curClusterCapacity[i].l.RLock()
		if v.host == hostip {
			curClusterCapacity[i].l.RUnlock()
			return i, nil
		}
		curClusterCapacity[i].l.RUnlock()
	}
	return -1, errors.New("没有对应的主机")
}

func GetUpdateInfoInupdate(w http.ResponseWriter, enc Encoder, r *http.Request) (int, string) { //接收信息，放入channel
	var receiveInfo updateInfo
	if err := json.NewDecoder(r.Body).Decode(&receiveInfo); err != nil {
		log.Println("接收的状态更新信息解码错误，", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		fmt.Println("接收数据错误", err)
		return http.StatusBadRequest, Must(enc.Encode(err))
	}
	if receiveInfo.Host == "" {
		receiveInfo.Host = r.RemoteAddr
		temp := strings.Split(receiveInfo.Host, ":")
		if len(temp) > 0 {
			receiveInfo.Host = temp[0]
		}
	}
	// fmt.Println("更新信息是", receiveInfo)
	UpdateInfoChannel <- receiveInfo
	log.Println("写入一个状态", receiveInfo)
	// log.Println("ID是", receiveInfo.Containers[0].Id, "第一个端口号", receiveInfo.Containers[1].Port)
	return http.StatusOK, Must(enc.Encode(""))
}

func DeleteServerInUpdate(index int) error { //delete element in curClusterCapacity
	if index < 0 {
		return errors.New("index小于0")
	} else if index > len(curClusterCapacity)-1 {
		return errors.New("index超出最大范围")
	}
	ClusterCapacityLock.RLock()
	temp := curClusterCapacity
	ClusterCapacityLock.RUnlock()
	var re []ServerCapacity
	if index == 0 {
		re = temp[1:]
	} else if index == len(curClusterCapacity)-1 {
		re = temp[:index]
	} else {
		p1 := temp[:index]
		p2 := temp[index+1:]
		re = append(p1, p2...)
	}
	ClusterCapacityLock.Lock()
	curClusterCapacity = re
	ClusterCapacityLock.Unlock()
	return nil
}
func DeleteContainerInUpdate(sindex int, cindex int) error { //delete element in ServerCapacity.containers
	if sindex < 0 || sindex > len(curClusterCapacity)-1 {
		return errors.New("服务器序号范围不符")
	}
	curClusterCapacity[sindex].l.Lock()
	defer curClusterCapacity[sindex].l.Unlock()
	if cindex < 0 || cindex > len(curClusterCapacity[sindex].containers)-1 {
		return errors.New("容器序号不符")
	}
	temp := curClusterCapacity[sindex].containers
	var re []ContainerCapacity
	if cindex == 0 {
		re = temp[1:]
	} else if cindex == len(curClusterCapacity[sindex].containers)-1 {
		re = temp[:cindex-1]
	} else {
		p1 := temp[:cindex-1]
		p2 := temp[cindex+1:]
		re = append(p1, p2...)
	}
	curClusterCapacity[sindex].containers = re
	return nil
}

func PurgeClusterCapacity() { //根据时间，清理curClusterCapacity中超时未更新的服务器和容器
	timeSlot := time.NewTimer(time.Second * 1)
	for {
		select {
		case <-timeSlot.C:
			for i := len(curClusterCapacity) - 1; i >= 0; i-- {
				if len(UpdateInfoChannel) > 90 {
					log.Println("更新信息通道被阻塞！！！")
					continue
				}
				if time.Now().Sub(curClusterCapacity[i].updateTime) > 5*1000*1000*1000 { //超过5s未更新,删除此服务器
					// log.Println("删除服务器,时长", curClusterCapacity[i].updateTime)
					log.Println("删除内容是", curClusterCapacity[i], "序号是", i)
					DeleteServerInUpdate(i)
					log.Println("删除后的集群是", curClusterCapacity)
					continue
				}
				//服务端已经做了容器生存检验，这里不再需要
				/*				for ii := len(curClusterCapacity[i].containers) - 1; ii >= 0; ii-- { //5s未更新删除此容器
								if time.Now().Sub(curClusterCapacity[i].containers[ii].updateTime) > 5*1000*1000*1000 {
									log.Println("删除容器", curClusterCapacity[i].containers[ii])
									DeleteContainerInUpdate(i, ii)
								}
							}*/

			}
			timeSlot.Reset(time.Second * 5)
		}
	}
}

func UpdateClusterCapacityInUpdate() { //每次更新一个服务器中的信息
	stat := <-UpdateInfoChannel //取出一个更新信息
	log.Println("取出一个状态")
	hostIndex, err := FindServerByHostInUpdate(stat.Host)
	if err != nil {
		// log.Println("找不到主机,更新信息是", stat)
		// return
		//新建主机
		log.Println("新建主机", stat.Host)
		var temp ServerCapacity
		temp.host = stat.Host
		ClusterCapacityLock.Lock()
		curClusterCapacity = append(curClusterCapacity, temp)
		ClusterCapacityLock.Unlock()
		hostIndex = len(curClusterCapacity) - 1
	}
	stat.Cpu = stat.Cpu / 100
	serverMemUsage := stat.Mem / ServerMemCapacity
	log.Println("CPU", stat.Cpu, "内存用量", stat.Mem, "内存百分比", serverMemUsage)
	var ServerCapacity int
	if stat.Cpu > serverMemUsage {
		ServerCapacity = int(math.Floor(float64(DefaultServerCapacity) * (1 - stat.Cpu)))
	} else {
		ServerCapacity = int(math.Floor(float64(DefaultServerCapacity) * (1 - serverMemUsage)))
	}
	// log.Println("checkpoint")
	if len(curClusterCapacity) > 0 {
		log.Println("锁", curClusterCapacity[0].l)
	}
	//更新服务器信息
	curClusterCapacity[hostIndex].l.Lock()
	curClusterCapacity[hostIndex].CapacityLeft = ServerCapacity
	curClusterCapacity[hostIndex].updateTime = time.Now()
	curClusterCapacity[hostIndex].l.Unlock()
	// log.Println("checkpoint")
	var tempContainers []ContainerCapacity
	for _, v := range stat.Containers { //更新容器信息
		if v.Id == "" {
			continue
		}
		if v.Mem < 0.0001 { //此时信息获取不完整
			log.Println("信息不完整未添加", v)
		}
		memUsage := v.Mem / ContainerMemCapacity
		v.Cpu = v.Cpu / 100
		var Capacity int
		if v.Cpu > memUsage {
			Capacity = int(math.Floor(float64(DefaultContainerCapacify) * (1 - v.Cpu)))
		} else {
			Capacity = int(math.Floor(float64(DefaultContainerCapacify) * (1 - memUsage)))
		}
		if Capacity < 0 {
			log.Println("容器错误", Capacity, "CPU是", v.Cpu, "内存是", memUsage)
		}

		var temp ContainerCapacity
		tINameArray := strings.Split(v.Image, "/")
		if len(tINameArray) > 1 {
			temp.imageName = tINameArray[1]
		} else {
			temp.imageName = v.Image
		}

		temp.capacityLeft = Capacity
		temp.containerID = v.Id
		temp.host = stat.Host
		// temp.imageName = v.Image
		Port, errPort := strconv.Atoi(v.Port)
		if errPort != nil {
			temp.port = 0
			log.Println("端口信息不完整未添加,", v)
			continue
		} else {
			temp.port = Port
		}

		temp.updateTime = time.Now()
		// curClusterCapacity[hostIndex].l.Lock()
		// curClusterCapacity[hostIndex].containers = append(curClusterCapacity[hostIndex].containers, temp)
		// curClusterCapacity[hostIndex].l.Unlock()
		tempContainers = append(tempContainers, temp)
		log.Println("添加容器", temp)

	}
	curClusterCapacity[hostIndex].l.Lock()
	curClusterCapacity[hostIndex].containers = curClusterCapacity[hostIndex].containers[0:0]
	curClusterCapacity[hostIndex].containers = append(curClusterCapacity[hostIndex].containers, tempContainers...)
	curClusterCapacity[hostIndex].l.Unlock()
	for i, v := range curClusterCapacity {
		log.Print("第", i, "个服务器容器", v.CapacityLeft, "内存最大", ServerMemCapacity)
		log.Print("地址", v.host)
		log.Print("更新时间", v.updateTime)
		log.Print("服务器容器", v.containers)
	}

}

func updateDeamon() {
	for {
		UpdateClusterCapacityInUpdate()
		// log.Println("长度是", len(UpdateInfoChannel))
	}

}
