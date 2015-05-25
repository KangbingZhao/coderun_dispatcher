package main

import (
	"fmt"
	"testing"
)

func Test_create(t *testing.T) {
	go CreateContainerMain()

	data := createContainerData{
		serverIP:  "127.0.0.1",
		imageName: "ubuntu:latest22",
		addr:      make(chan *containerAddr),
		err:       make(chan *reError),
	}
	createBuf <- data
	addr := <-data.addr
	err := <-data.err
	if addr != nil {
		fmt.Println(addr)
	}
	if err != nil {
		fmt.Println(err)
	}
}

func BenchmarkAdd(b *testing.B) {
	b.StopTimer()
	go CreateContainerMain()
	b.StartTimer()
	for i := 0; i < b.N; i++ {
		data := createContainerData{
			serverIP:  "127.0.0.1",
			imageName: "ubuntu:latest000",
			addr:      make(chan *containerAddr),
			err:       make(chan *reError),
		}
		createBuf <- data
		addr := <-data.addr
		err := <-data.err
		if addr != nil {
			fmt.Println(addr)
		}
		if err != nil {
			fmt.Println(err)
		}
	}
}
