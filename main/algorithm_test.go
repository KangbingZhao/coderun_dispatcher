package main

import (
	// "fmt"
	"testing"
	"time"
)

func TestFindImagesInServer(t *testing.T) {
	var tempS ServerCapacity
	var tempC ContainerCapacity

	tempC.capacityLeft = 1
	tempC.containerID = "ididid"
	tempC.host = "host"
	tempC.imageName = "imagename"
	tempC.port = 0
	tempC.updateTime = time.Now()

	tempS.CapacityLeft = 100
	tempS.host = "hosthost"
	tempS.updateTime = time.Now()
	tempS.containers = append(tempS.containers, tempC)
	// curClusterCapacity = curClusterCapacity[0:0]
	ftest := "test"
	ttest := "imagename"
	fre := FindImagesInServer(tempS, ftest)
	tre := FindImagesInServer(tempS, ttest)
	if len(fre) == 0 && len(tre) != 0 {
		t.Errorf("查找镜像成功")
	} else {
		t.Errorf("查找镜像出错")
	}

}
