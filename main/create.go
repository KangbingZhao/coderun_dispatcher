package main

import (
	"errors"
	"log"
	"time"
)

type createContainerData struct {
	serverIP  string
	imageName string
	addr      chan *containerCreated
	err       chan *reError
}
type createSnap struct {
	addr   containerCreated
	err    reError
	t      time.Time
	status int
	count  int
}

var (
	//buf is 100
	createBuf     = make(chan createContainerData, 100)
	createConsMap = make(map[string]*createSnap)
)

func RemoteCreateContainer(one createContainerData) {
	temp, err := createNewContainer(one.serverIP, one.imageName)
	var re containerCreated
	log.Println("创建创建创建")
	if err.err != nil { //出错
		re.Status = 6
		log.Println("错误信息", err)
		re.Instance = temp
		//not update
		now := time.Now()
		createConsMap[one.imageName] = &createSnap{
			addr:   re,
			err:    err,
			t:      now,
			status: 6,
		}
	} else { //正确
		//update
		time.Sleep(400 * time.Millisecond)
		now := time.Now()
		re.Status = 2
		re.Instance = temp
		createConsMap[one.imageName] = &createSnap{
			addr:   re,
			err:    err,
			t:      now,
			status: 2,
		}
		var newContainer ContainerCapacity //添加新容器
		// newContainer.capacityLeft = DefaultContainerCapacify - 1
		newContainer.containerID = re.Instance.containerID
		newContainer.host = re.Instance.ServerIP
		newContainer.imageName = one.imageName
		newContainer.port = re.Instance.ServerPort
		newContainer.updateTime = time.Now()

		hostIndex, errHI := FindServerByHostInUpdate(newContainer.host)
		if errHI != nil { //查找出错
			log.Println("查找出错")
		}

		hasContainer := FindContainerInServer(curClusterCapacity[hostIndex], newContainer.containerID)
		if !hasContainer {
			curClusterCapacity[hostIndex].containers = append(curClusterCapacity[hostIndex].containers, newContainer)
		} else {
			log.Println("此容器已存在", newContainer)
		}
	}
}
func CreateContainerMain() {

	for {
		one := <-createBuf
		log.Println("随便写个啥")
		last := createConsMap[one.imageName]
		if last != nil {
			if last.status == 6 {
				log.Println("数个数字6")
				one.err <- &last.err
				one.addr <- &last.addr
				continue
			} else if last.status == 7 {
				log.Println("数个数字7")
				log.Println("统计数字是", createConsMap[one.imageName].count)
				createConsMap[one.imageName].count++
				if createConsMap[one.imageName].count > DefaultContainerCapacify {
					createConsMap[one.imageName].count -= 50

					go RemoteCreateContainer(one)
				}
				/*if time.Now().Sub(last.t) > 100*1000*1000 {
					log.Println("统计数量是", createConsMap[one.imageName].count)
					for i := 0; i < (createConsMap[one.imageName].count / 50); i++ {
						go RemoteCreateContainer(one)
					}
				}*/

				one.err <- &last.err
				one.addr <- &last.addr
				continue
			} else {
				now := time.Now()
				dur := now.Sub(last.t)
				if dur < 1000*1000*100 {
					//if <10 ms return redirct
					log.Println("不知道写啥")
					AvailableContainerPointer := SearchAvailableContainerInCluster(one.imageName) //75%
					log.Println("不知道写啥2")
					if AvailableContainerPointer != nil {
						// one.addr <- &last.addr
						log.Println("找到合适的容器", AvailableContainerPointer.Instance)
						log.Println("第一个长度", len(one.err), "第二个长度", len(one.addr), "内容是", last.err)
						one.err <- &(last.err)

						log.Println("第一次放数据成功")
						one.addr <- (AvailableContainerPointer)
						// one.addr <- nil
						log.Println("第二次放数据成功")

						continue
					}
					log.Println("找不到合适的容器")
					// AvailableContainerPointer2 := SearchAvailableContainerInCluster(one.imageName, 10)
					/*				one.addr <- &last.addr
									one.err <- &last.err*/

				}

			}

		}
		//need create
		var re containerCreated
		re.Status = 7
		re.Instance = containerAddr{}
		//not update
		initCount := int(0)
		now := time.Now()
		createConsMap[one.imageName] = &createSnap{
			addr:   re,
			err:    reError{err: errors.New("正在创建"), msg: "正在创建"},
			t:      now,
			status: 7,
			count:  initCount,
		}
		// tt := time.Now()
		go RemoteCreateContainer(one)

		one.err <- &reError{err: errors.New("正在创建"), msg: "正在创建"}
		one.addr <- &re
	}
}
