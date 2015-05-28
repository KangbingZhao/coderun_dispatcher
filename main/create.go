package main

import (
	"time"
)

type createContainerData struct {
	serverIP  string
	imageName string
	addr      chan *containerAddr
	err       chan *reError
}
type createSnap struct {
	addr   containerAddr
	err    reError
	t      time.Time
	status int
}

var (
	//buf is 100
	createBuf = make(chan createContainerData, 100)
)

func CreateContainerMain() {
	createConsMap := make(map[string]*createSnap)
	for {
		one := <-createBuf
		last := createConsMap[one.imageName]
		if last != nil {
			if last.status == 6 {
				one.addr <- &last.addr
				one.err <- &last.err
				continue
			} else {
				now := time.Now()
				dur := now.Sub(last.t)
				if dur < 1000*1000*50 {
					//if <2 ms return redirct
					one.addr <- &last.addr
					one.err <- &last.err
					continue
				}
			}

		}
		//need create
		temp, err := createNewContainer(one.serverIP, one.imageName)
		if err.err != nil { //出错
			//not update
			now := time.Now()
			createConsMap[one.imageName] = &createSnap{
				addr:   temp,
				err:    err,
				t:      now,
				status: 6,
			}
		} else { //正确
			//update
			now := time.Now()
			createConsMap[one.imageName] = &createSnap{
				addr:   temp,
				err:    err,
				t:      now,
				status: 2,
			}
		}
		one.addr <- &temp
		one.err <- &err
	}
}
