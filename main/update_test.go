package main

import (
	"fmt"
	"testing"
)

func TestgetInitialServerAddrInUpdate(t *testing.T) {
	r := getInitialServerAddrInUpdate()
	t.Errorf("getInitialServerAddrInUpdate()出错，%s", r)
	fmt.Println("ha")
}

//Find
