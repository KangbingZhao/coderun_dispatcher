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
	time.Sleep(n * time.Microsecond * 1000000)
}
func FindContainerInServer(ss ServerCapacity, containerID string) bool {
	for _, v := range ss.containers {
		if v.containerID == containerID {
			return true
		}
	}
	return false
}
func createNewContainerWithQuene(serverIP string, imageName string) (containerCreated, reError) {
	data := createContainerData{
		serverIP:  serverIP,
		imageName: imageName,
		addr:      make(chan *containerCreated),
		err:       make(chan *reError),
	}
	log.Println("哈哈锁住了")
	createBuf <- data
	log.Println("哈哈锁住了1")
	err := <-data.err
	addr := <-data.addr
	log.Println("哈哈锁住了2")
	log.Println("哈哈从创建队列中获取的信息是", *addr)

	return *addr, *err
}
func createNewContainer(serverIP string, imageName string) (containerAddr, reError) { //创建新的容器
	createURL := "http://" + serverIP + ":9090/createrunner/" + imageName

	for i := 0; i < 20; i++ { //发送创建容器的请求，最多10秒钟
		client := &http.Client{}
		req, reqError := http.NewRequest("POST", createURL, nil)
		log.Println("创建URL是", createURL)
		if reqError != nil {
			return containerAddr{"", 0, "", ""}, reError{"req初始化失败", reqError}
		}
		// log.Println("checkpoint")
		resq, req1Err := client.Do(req)
		if req1Err != nil {
			//发送失败,直接退出
			log.Println("发送请求失败", req1Err)
			return containerAddr{"", 0, "", ""}, reError{"发送创建请求失败", req1Err}
		}
		log.Println("checkpoint")
		data, err := ioutil.ReadAll(resq.Body)
		if err != nil {
			//数据读取失败，直接退出
			return containerAddr{"", 0, "", ""}, reError{"创建请求结果无法读取", err}
		}
		defer resq.Body.Close()
		log.Println("checkpoint")
		createResult, errR := jason.NewObjectFromBytes(data)
		if errR != nil {
			return containerAddr{"", 0, "", ""}, reError{"创建请求结果解析出错", errR}
		}
		log.Println("返回结果是", createResult)
		createStatus, errS := createResult.GetInt64("status")
		if errS != nil {
			return containerAddr{"", 0, "", ""}, reError{"创建请求结果无法解析", errS}
		}
		if createStatus == 3 { //创建成功
			log.Println("创建成功!")
			containerHost, errCH := createResult.GetString("hosts")
			if errCH != nil {
				return containerAddr{"", 0, "", ""}, reError{"创建请求结果无法解析出主机地址", errCH}
			}
			containerInfo, errCI := createResult.GetObject("instances")
			if errCI != nil {
				return containerAddr{"", 0, "", ""}, reError{"创建请求结果无法解析出容器信息", errCI}
			}
			containerPort, errCP := containerInfo.GetInt64("port")
			if errCP != nil {
				return containerAddr{"", 0, "", ""}, reError{"创建请求结果无法解析出容器端口", errCP}
			}
			containerId, errCID := containerInfo.GetString("container_id")
			if errCID != nil {
				return containerAddr{"", 0, "", ""}, reError{"创建请求结果无法解析出容器ID", errCID}
			}

			return containerAddr{containerHost, int(containerPort), containerId, imageName}, reError{"ok", nil}

		} else if createStatus == 1 { //延迟后重新请求
			log.Println("延迟后重新请求")
			delaySecond(10)
			continue

		} else if createStatus == 2 { //重新调用创建api
			log.Println("重新调用,id是", imageName, "Status is ", createStatus)
			delaySecond(1)
			continue
		} else if createStatus == 6 { //pull失败
			log.Println("Pull失败，id是", imageName)
			return containerAddr{"", 0, "", ""}, reError{"无法获取镜像", errors.New("无法获取镜像")}
		}

	}

	//timeout

	return containerAddr{"", 0, "", ""}, reError{"创建镜像超时", errors.New("创建镜像超时")}

}

func RR(currentServerStatus []curServerStatus) containerAddr { // a Round-Robin
	// 直接按照服务器轮流新建容器,只需要返回服务器IP
	// fmt.Println("我来自算法啊")
	temp := callCount % len(currentServerStatus)
	callCount = temp + 1
	ip := currentServerStatus[temp].machineStatus.Host
	tc := containerAddr{ip, 0, "", ""}
	return tc

	// return currentServerStatus[temp].machineStatus.Host
}

func LCS(currentServerStatus []curServerStatus) containerAddr { // Lease-Connection Scheduling
	//选择当前运行容器最少的服务器，直接返回其IP
	var min int = 0
	// var minIP string
	serverNum := len(currentServerStatus) //当前在线的服务器数量
	if serverNum == 0 {                   //没有正常工作的服务器
		return containerAddr{"", 0, "", ""}
	} else if serverNum == 1 { //只有一台服务器
		return containerAddr{currentServerStatus[0].machineStatus.Host, 0, "", ""}
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
	return containerAddr{currentServerStatus[min].machineStatus.Host, 0, "", ""}
}

func GetServerLoad(ss serverStat) float64 { //CPU和RAM使用率百分比的加权平均，暂定为0.5、0.5
	memUsage := ss.memUsageTotal / ss.memCapacity
	// log.Println("内存容器是", ss.memCapacity)
	// log.Println("内存用量是", ss.memUsageTotal)
	// log.Println("CPU用量是", ss.cpuUsage)
	if ss.cpuUsage > 0.9 || memUsage > 0.9 {
		return 1.0
	}
	if ss.cpuUsage > memUsage {
		return ss.cpuUsage
	} else {
		return memUsage
	}
	// return (ss.cpuUsage + memUsage) / 2
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
		return containerAddr{"", 0, "", ""}
	} else if serverNum == 1 {
		return containerAddr{currentServerStatus[0].machineStatus.Host, 0, "", ""}
	} else { // 两台以上服务器在线
		var temp int = 0
		for i, v := range currentServerStatus {
			if GetServerLoad(currentServerStatus[temp].machineStatus) > GetServerLoad(v.machineStatus) {
				temp = i
			}
		}
		return containerAddr{currentServerStatus[temp].machineStatus.Host, 0, "", ""}
	}
}

func FindImagesInServer(currentServerCapacity ServerCapacity, imageName string) []int {
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
			if currentServerCapacity.containers[re[i1]].capacityLeft > currentServerCapacity.containers[re[i2]].capacityLeft {
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
	for _, v := range re {
		log.Println("排序后的集群容量")
		log.Println(v)
	}
	// for _, v := range curClusterCapacity {
	// 	v.l.Unlock()
	// }
}

func SearchAvailableContainerInCluster(imageName string) *containerCreated {
	sortServerByLoad()
	overload75 := bool(false)
	overload90 := bool(false)
	if len(curClusterCapacity) < 1 {
		var re containerCreated
		re.Status = 6
		log.Println("请求镜像名", imageName, "无可用服务器")
		return &re
	}
	log.Println("排序结果", curClusterCapacity)
	var re containerCreated
	log.Println("大锁是", ClusterCapacityLock)
	ClusterCapacityLock.RLock()
	defer ClusterCapacityLock.RUnlock()
	for i, v := range curClusterCapacity {
		log.Println("第", i, "个小锁是", curClusterCapacity[i].l)
		curClusterCapacity[i].l.RLock()
		// fmt.Println("执行")
		//查找容器，找到且不过载则分配，找不到继续查找
		//循环结束后仍没有找到，则选择第一个（负载最轻的服务器分配）
		/*		if GetServerLoad(v.machineStatus) > 0.9 { //负载过高，不再分配
				continue
			}*/
		if v.CapacityLeft < DefaultServerCapacity/4 {
			//负载高于75%
			log.Println("选定的服务器已经过载,信息是", v)
			curClusterCapacity[i].l.RUnlock()
			if i == 0 {
				overload75 = true
				if v.CapacityLeft < DefaultServerCapacity/10 {
					overload90 = true
				}
			}

			break
		}
		imageList := FindImagesInServer(v, imageName)
		log.Println("镜像名", imageName)
		log.Println("列表", imageList)
		if len(imageList) == 0 { //不存在对应的容器
			// fmt.Println("执行22")
			curClusterCapacity[i].l.RUnlock()
			log.Println("分配时没有找到容器", imageName)
			continue
		} else {
			/*			for _, v11 := range imageList {
						log.Println("排序后的容器容量")
						log.Println(v.containers[v11])
					}*/
			//todo 选择第一个镜像进行分配,同时return
			/*			if GetContainerLoad(v.containerStatus[imageList[0]]) > 0.9 {
						log.Println("容器过载，不再分配此容器", v.containerStatus[imageList[0]].id)
						continue
					}*/
			if v.containers[imageList[0]].capacityLeft < DefaultContainerCapacify/20 {
				curClusterCapacity[i].l.RUnlock()
				log.Println("选定的容器过载，信息是", v.containers[imageList[0]])
				continue
			}

			re.Status = 3
			re.Instance.ServerIP = v.containers[imageList[0]].host
			re.Instance.ServerPort = v.containers[imageList[0]].port
			re.Instance.containerID = v.containers[imageList[0]].containerID
			re.Instance.imageName = imageName
			curClusterCapacity[i].l.RUnlock()
			log.Println("后面的小锁是", curClusterCapacity[i].l)
			curClusterCapacity[i].l.Lock()
			curClusterCapacity[i].CapacityLeft = v.CapacityLeft - 1
			curClusterCapacity[i].containers[imageList[0]].capacityLeft = v.containers[imageList[0]].capacityLeft - 1
			log.Println("分配信息是", v.containers[imageList[0]])
			curClusterCapacity[i].l.Unlock()
			return &re
		}
	}
	if !overload75 { //小于75%，新建
		return nil
	} else if overload75 && !overload90 { //75~90，分配
		for i, v := range curClusterCapacity {
			curClusterCapacity[i].l.RLock()
			// fmt.Println("执行")
			//查找容器，找到且不过载则分配，找不到继续查找
			//循环结束后仍没有找到，则选择第一个（负载最轻的服务器分配）
			/*		if GetServerLoad(v.machineStatus) > 0.9 { //负载过高，不再分配
					continue
				}*/

			imageList := FindImagesInServer(v, imageName)
			log.Println("镜像名", imageName)
			log.Println("列表", imageList)
			if len(imageList) == 0 { //不存在对应的容器
				// fmt.Println("执行22")
				curClusterCapacity[i].l.RUnlock()
				log.Println("分配时没有找到容器", imageName)
				continue
			} else {
				/*			for _, v11 := range imageList {
							log.Println("排序后的容器容量")
							log.Println(v.containers[v11])
						}*/
				//todo 选择第一个镜像进行分配,同时return
				/*			if GetContainerLoad(v.containerStatus[imageList[0]]) > 0.9 {
							log.Println("容器过载，不再分配此容器", v.containerStatus[imageList[0]].id)
							continue
						}*/

				re.Status = 3
				re.Instance.ServerIP = v.containers[imageList[0]].host
				re.Instance.ServerPort = v.containers[imageList[0]].port
				re.Instance.containerID = v.containers[imageList[0]].containerID
				re.Instance.imageName = imageName
				curClusterCapacity[i].l.RUnlock()
				log.Println("后面的小锁是", curClusterCapacity[i].l)
				curClusterCapacity[i].l.Lock()
				curClusterCapacity[i].CapacityLeft = v.CapacityLeft - 1
				curClusterCapacity[i].containers[imageList[0]].capacityLeft = v.containers[imageList[0]].capacityLeft - 1
				log.Println("分配信息是", v.containers[imageList[0]])
				curClusterCapacity[i].l.Unlock()
				return &re
			}
		}
		return nil //都超过了75，不存在所选容器，强行新建一个
	} else if overload75 && overload90 { //90~，拒绝服务
		var re containerCreated
		re.Status = 9
		return &re
	} else { //出错
		var re containerCreated
		re.Status = 9
		return &re
	}

}

func ServerAndContainer(imageName string) containerCreated { //优先选择已有的容器，容器及服务器都不过载则分配此容器，容器过载则重新分配容器；服务器过载则查找下一个服务器
	//status2表示容器创建成功，3表示分配现有的容器，6表示失败
	/*上述方案并不好，容器导致任务重的服务器负载越来越重，修改如下:
	*	先选择负载轻的服务器，在上面查找容器，选择负载最轻的容器分配(需要能够查出多个容器的函数)
	 */
	AvailableContainerPointer := SearchAvailableContainerInCluster(imageName) //75%
	if AvailableContainerPointer != nil {
		return *AvailableContainerPointer
	}
	// var re containerCreated
	//执行到这里说明没有找到镜像，返回第一台服务器的ip即可,也就是当前负载最低的服务器
	log.Println("没有合适的容器需要创建,服务器是", curClusterCapacity[0].host)
	// log.Println("sort长度是", len(sortedServerStatus))
	if curClusterCapacity[0].CapacityLeft < DefaultServerCapacity/10 {
		re := containerCreated{Status: 6}
		return re
	}
	curClusterCapacity[0].l.RLock()
	ServerIP := curClusterCapacity[0].host
	curClusterCapacity[0].l.RUnlock()
	temp, _ := createNewContainerWithQuene(ServerIP, imageName)

	// if err.err != nil { //出错
	// 	// re := containerCreated{6, {"", temp, 0}}
	// 	re.Status = 6
	// 	re.Instance = temp
	// 	log.Println("错误是", err.err)
	// 	return re
	// } else { //正确
	// 	// re := containerCreated{3, {temp.ServerIP, temp.ServerPort}}
	// 	re.Status = 2
	// 	re.Instance = temp

	// 	log.Println("创建容器成功", re.Instance)
	// 	log.Println("创建镜像名是", re.Instance.imageName, "请求镜像名是", imageName)
	// 	log.Println("当前集群状态是", curClusterCapacity)
	// 	return re
	// }
	if temp.Status == 2 {
		log.Println("创建容器成功", temp.Instance)
	}
	return temp
	// return createNewContainer(sortedServerStatus[0].machineStatus.Host, imageName)

	// return containerAddr{sortedServerStatus[0].machineStatus.Host, 0, ""}

}
