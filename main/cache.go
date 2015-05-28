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
	// TcurClusterStats := GetCurrentClusterStatus()
	TcurCluster := curClusterCapacity
	for _, v := range TcurCluster {
		for _, vv := range v.containers {
			if isServiceContainer(vv.imageName) {
				continue
			}
			var temp containerCreated
			temp.Status = 100 //开始时就装入的
			temp.Instance.ServerIP = vv.host
			temp.Instance.ServerPort = vv.port
			temp.Instance.containerID = vv.containerID
			CacheContainer.Add(vv.containerID, temp)
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

	//接下来从集群状态中删除此容器
	for i, v := range curClusterCapacity {
		for ii, vv := range v.containers {
			if vv.containerID == cc.Instance.containerID {
				err := DeleteContainerInUpdate(i, ii)
				if err != nil {
					log.Println("删除容器时，删除容量信息出错")
					return err
				} else {
					log.Println("删除容器时，删除容量信息成功")
				}
			}
		}
	}
	return errR
}

func getClusterCapacityLeft() int {
	var totalCapacityLeft int = 0
	for i, v := range curClusterCapacity {
		curClusterCapacity[i].l.RLock()
		totalCapacityLeft = totalCapacityLeft + v.CapacityLeft
		curClusterCapacity[i].l.RUnlock()
	}
	return totalCapacityLeft
}

func RestrictContainer() { //若集群负载高于90%，调用此函数清除最久未被使用的容器，清理五个容器

	for j := 0; j < 5; j++ {
		id, err := CacheContainer.GetOldestKey()
		if err != nil {
			log.Fatalln("集群释放容器时，无法获取最后一个元素的值")
			// fmt.Println("集群释放容器时，无法获取最后一个元素的值")
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

	}

}
func StartCacheDeamon() {
	delaySecond(5)
	// log.Println("Cache启动了")
	timeSlot := time.NewTimer(time.Second * 1) // update status every second
	//读取持久化服务列表，供载入现有容器时过滤
	InitialServiceContainers, err := getInitialServiceContainers()
	if err != nil {
		log.Println("获取持久容器列表错误", err)
		return
	}
	CurrentServiceContainers = InitialServiceContainers.ServiceContainer
	loadCurrentContainer() //将当前运行中的容器载入Cache中
	for {
		select {
		case <-timeSlot.C:
			tempServiceContainers, err := getInitialServiceContainers()
			if err != nil {
				log.Println("获取持久容器列表错误", err)
				return
			}
			CurrentServiceContainers = tempServiceContainers.ServiceContainer
			// fmt.Println("初始化守护容器时", CurrentServiceContainers)

			// CurCLoad, errCCL := GetClusterLoad(curClusterStats)
			/*			CurCLoad, errCCL := GetClusterLoad(GetCurrentClusterStatus())
						if errCCL != nil {
							log.Println("CacheDeamon中无法获取集群负载")
							log.Println("错误是", errCCL)
							log.Println("集群状态是", GetCurrentClusterStatus())
						} else if CurCLoad > 0.9 {
							RestrictContainer(GetCurrentClusterStatus()) //定期清理容器
							log.Println("开启了RestrictContainer,Load是", CurCLoad)
						}*/
			// fmt.Println("执行到deamon了")
			if getClusterCapacityLeft()*10 < len(curClusterCapacity)*DefaultServerCapacity { //剩余容量小于十分之一
				RestrictContainer()
				log.Println("启动清理程序，剩余容量是", getClusterCapacityLeft())
			}
			timeSlot.Reset(time.Second * 5)
		}
	}
}
