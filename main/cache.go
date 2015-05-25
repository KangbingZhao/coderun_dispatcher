/*
*Maintain the Cache of container
 */

package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/fsouza/go-dockerclient"
	"log"
	"os"
	"time"
)

// var serviceContainers = make([]containerAddr, 0, 20)
var CurrentServiceContainers = make([]serviceContainer, 0, 20)

type serviceContainer struct {
	ImageName string
}

type serviceContainers struct { //从serviceContainer.json中读取的持久服务容器的信息
	ServiceContainer []serviceContainer
}

func getInitialServiceContainers() (serviceContainers, error) { //从serviceContainer.json中读取持久服务容器的信息
	var c serviceContainers
	r, err := os.Open("../metadata/serviceContainer.json")
	if err != nil {
		logger.Error(err)
		return c, err
	}
	decoder := json.NewDecoder(r)

	errD := decoder.Decode(&c)
	if errD != nil {
		logger.Error(err)
		return c, errD
	}
	// fmt.Println("读取文件错误是", err)
	// fmt.Println("解码错误时", errD)
	return c, nil
}

func loadCurrentContainer() { //初始化时，将现有容器放入缓存区中,排除ServiceContainer
	// delaySecond(5)
	TcurClusterStats := GetCurrentClusterStatus()
	for _, v := range TcurClusterStats {
		if v.machineStatus.cpuCore == -1 {
			continue
		}
		for _, vv := range v.containerStatus {
			if isServiceContainer(vv.name) {
				continue
			}
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
		log.Println("装入缓存内容是", tt)
	}
	// return nil
}

func isServiceContainer(imageName string) bool { //检验一个容器是否是持久服务的容器
	if len(CurrentServiceContainers) == 0 {
		return false
	}
	for _, v := range CurrentServiceContainers {
		if v.ImageName == imageName {
			return true
		}
	}
	return false
}

func GetClusterLoad(currentServerStatus []curServerStatus) (float64, error) { // ServerLoad的算术平均，因为每台服务器都是等价的
	totalLoad := float64(0)
	if len(currentServerStatus) < 1 {
		return 0, errors.New("没有可用的服务器,无法获取集群负载")
	}
	for _, v := range currentServerStatus {
		totalLoad = totalLoad + GetServerLoad(v.machineStatus)
		// log.Println("机器", i, "的负载时", GetServerLoad(v.machineStatus))
	}
	// log.Println("长度", len(currentServerStatus), "总负载", totalLoad)
	return totalLoad / float64(len(currentServerStatus)), nil
}

func evictElement(cc containerCreated) error { //从缓存中清除一个容器时，向Docker发请求删除容器
	if cc.Status != 2 && cc.Status != 3 && cc.Status != 100 {
		return errors.New("该容器不存在")
	}

	//其实不应该这么写，因为服务器的Docker端口可能是变化的，未必是4243
	// url := cc.Instance.ServerIP + ":4243/containers/" + cc.Instance.containerID + "?v=1&force=1"
	serverUrl := "http://" + cc.Instance.ServerIP + ":4243"
	fmt.Println("docker地址是", serverUrl)
	client, errC := docker.NewClient(serverUrl)
	if errC != nil {
		return errors.New("无法连接Docker服务器")
	}
	opts := docker.RemoveContainerOptions{ID: cc.Instance.containerID, RemoveVolumes: false, Force: true}
	errR := client.RemoveContainer(opts)
	// fmt.Println(client)
	fmt.Println("删除函数的错误是", errR)
	log.Println("删除容器,ID是", cc.Instance.containerID, "错误信息是", errR)
	return errR
}

func RestrictContainer(currentServerStatus []curServerStatus) { //若集群负载高于90%，调用此函数清除最久未被使用的容器，清理五个容器
	// delaySecond(10)
	// fmt.Println("集群状态", currentServerStatus)
	// fmt.Println("执行了")
	load, errL := GetClusterLoad(currentServerStatus)
	if errL != nil {
		log.Fatalln("集群释放容器时，无法获取就集群负载")
		fmt.Println(errL)
		return
	}
	// fmt.Println("执行了2")
	cacheLength := CacheContainer.Len()
	if load < 0.9 && cacheLength < 150 { //不需要清楚容器
		log.Println("集群负载是", load, "缓存长度是", cacheLength, "不需要释放容器")
		return
	}
	// fmt.Println("执行了3")
	for j := 0; j < 5; j++ {
		id, err := CacheContainer.GetOldestKey()
		if err != nil {
			log.Fatalln("集群释放容器时，无法获取最后一个元素的值")
			fmt.Println("集群释放容器时，无法获取最后一个元素的值")
			return
		}
		// fmt.Println("执行了4")
		targetContainer, ok := CacheContainer.Get(id)
		if !ok {
			// fmt.Println("执行了5")
			log.Fatalln("无法获取对应的容器,ID是", id)
		} else {
			// fmt.Println("执行了6")
			tar, ok := targetContainer.(containerCreated)
			if ok {
				// fmt.Println("转换成功")
				removeResult := evictElement(tar)
				if removeResult != nil {
					log.Fatalln("清理容器时出错", removeResult)
					fmt.Println(removeResult)
				} else {
					log.Println("清理容器成功", tar.Instance.containerID)
				}
			} else {
				log.Fatalln("无法转换类型")
				// fmt.Println("转换不成功")
			}

		}

		// CacheContainer.Remove(id)
		if CacheContainer.Len() < 5 { //容器太少时也不再清理
			return
		}
		// l2, errL2 := GetClusterLoad(currentServerStatus)
		// if errL2 != nil {
		// 	log.Fatalln("集群释放容器时，无法获取集群负载")
		// }
		// if l2 < 0.7 { //服务器负载已经足够低
		// 	return
		// }

	}

}

func StartCacheDeamon() {
	delaySecond(5)
	timeSlot := time.NewTimer(time.Second * 1) // update status every second
	//读取持久化服务列表，供载入现有容器时过滤
	InitialServiceContainers, err := getInitialServiceContainers()
	if err != nil {
		logger.Errorln(err)
		return
	}
	CurrentServiceContainers = InitialServiceContainers.ServiceContainer
	loadCurrentContainer() //将当前运行中的容器载入Cache中
	for {
		select {
		case <-timeSlot.C:
			tempServiceContainers, err := getInitialServiceContainers()
			if err != nil {
				logger.Errorln(err)
				return
			}
			CurrentServiceContainers = tempServiceContainers.ServiceContainer
			// fmt.Println("初始化守护容器时", CurrentServiceContainers)

			// CurCLoad, errCCL := GetClusterLoad(curClusterStats)
			CurCLoad, errCCL := GetClusterLoad(GetCurrentClusterStatus())
			if errCCL != nil {
				log.Println("CacheDeamon中无法获取集群负载")
				log.Println("错误是", errCCL)
				log.Println("集群状态是", GetCurrentClusterStatus())
			} else if CurCLoad > 0.9 {
				RestrictContainer(GetCurrentClusterStatus()) //定期清理容器
				log.Println("开启了RestrictContainer,Load是", CurCLoad)
			}
			// fmt.Println("执行到deamon了")
			timeSlot.Reset(time.Second * 10)
		}
	}
}
