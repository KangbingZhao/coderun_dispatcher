package main

import (
	"encoding/json"
	"fmt"
	"github.com/Sirupsen/logrus"
	"github.com/go-martini/martini"
	"io/ioutil"
	"net/http"
	// "net/url"
	"os"
	// "reflect"
	"strconv"
	"strings"
	// "strconv"
)

var logger = logrus.New()

type containerAddr struct { // function dispatcherContainer will return this
	ServerIP   string
	ServerPost int
}
type imageName struct { // function dispatchContainer receive this parameter
	iName string
}

type serverConfig struct { // store data of ./metadata/config.json
	Server []struct {
		Host         string
		DockerPort   int
		CAdvisorPort int
	}
}

type machineUsage struct {
	CPUUsage string
	MemUsage string
}

func getInitialServerAddr() serverConfig { // get default server info from ./metadata/config.json
	r, err := os.Open("./metadata/config.json")
	if err != nil {
		logger.Error(err)
	}
	decoder := json.NewDecoder(r)
	var c serverConfig
	err = decoder.Decode(&c)
	if err != nil {
		logger.Error(err)
	}
	for k, v := range c.Server {
		fmt.Println(k, v.Host, v.DockerPort, v.CAdvisorPort)
	}
	return c

}

type serverStat struct {
	cpuUsage     float32 //百分比
	cpuFrequency int     //Hz
	cpuCore      int     //核心数

	memUsageTotal float32 //内存容量，单位为Byte
	memUsageHot   float32 //当前活跃内存量
	memCapacity   float32 //内存总量

}

func getServerStats(serverList serverConfig) []machineUsage { // get current server stat

	su := make([]machineUsage, 3, 10)
	for index := 0; index < len(serverList.Server); index++ {

		if serverList.Server[index].Host == "" {
			continue
		}
		su[index].CPUUsage = "CPU" + strconv.Itoa(index)
		su[index].MemUsage = "Mem" + strconv.Itoa(index)

		cadvisorUrl := "http://" + serverList.Server[index].Host + ":" + strconv.Itoa(serverList.Server[index].CAdvisorPort)
		posturl := cadvisorUrl + "/api/v1.0/containers"

		reqContent := "{\"num_stats\":1,\"num_samples\":0}"
		body := ioutil.NopCloser(strings.NewReader(reqContent))
		client := &http.Client{}
		req, _ := http.NewRequest("POST", posturl, body)
		resq, err := client.Do(req)
		defer resq.Body.Close()
		data, _ := ioutil.ReadAll(resq.Body)
		// fmt.Println(string(data), err)

		//保存获取的服务器状态信息
		var jsonEncode interface{}
		err = json.Unmarshal(data, &jsonEncode)
		if err != nil {
			logger.Error(err)
		}
		tempStat, ok := jsonEncode.(map[string]interface{})
		if ok {
			fmt.Println("结果")
			// fmt.Println(tempStat["stats"])
			// md, _ := tempStat["stats"].(map[string]interface{})
			// fmt.Println(md)
			// fmt.Println(reflect.TypeOf(tempStat["stats"]["0"]))
			md, ok := tempStat["stats"].([]interface{})
			// fmt.Println(md)
			for _, iiv := range md {
				// fmt.Println(ii, "ii is ", iiv)
				mm, _ := iiv.(map[string]interface{})
				fmt.Println(mm["memory"])
			}
			fmt.Println("ok is ", ok)

			/*			var t interface{}

																																	   			t, ok = md.(map[string]interface{})*/

			fmt.Println("ok is ", ok)
			/*			for k, v := range t {
						switch v2 := v.(type) {
						case string:
							fmt.Println(k, " is string", v2)
						case int:
							fmt.Println(k, " is string", v2)
						default:
							fmt.Println(k, " is other ", v2)
						}
					}*/

			// var ha interface{}
			// fmt.Println("ha is ", ha.(interface{}))

			fmt.Println("结果")
			//应当从中获取cpu_usage和memUsage、mee_working_set
		}

		posturl2 := cadvisorUrl + "/api/v1.0/machine"
		client2 := &http.Client{}
		req2, _ := http.NewRequest("POST", posturl2, nil)
		resq2, _ := client2.Do(req2)
		defer resq2.Body.Close()
		data2, _ := ioutil.ReadAll(resq2.Body)

		var jsonEncode2 interface{}
		err = json.Unmarshal(data2, &jsonEncode2)
		if err != nil {
			logger.Error(err)
		}
		tempStat2, _ := jsonEncode2.(map[string]interface{})
		//从此处获取没内存总量
		fmt.Println("Machine信息")
		fmt.Println(tempStat2["memory_capacity"])

		// fmt.Println("body is ", reqContent)

	} // end of loop
	// fmt.Println("daowol")
	return su

}

func getValidContainer() {

}

func dispatchContainer(w http.ResponseWriter, r *http.Request) {
	// 接受image-name，返回json（server-ip，server-port）
	// return "hello world " + param["word"]
	// var ca containerAddr
	var in imageName //the image name received
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		logger.Warnf("error decoding image: %s", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	/*	out := containerAddr{
			ServerIP:   "456789",
			ServerPost: 32,
		}
		w.Header().Set("content-type", "application/json")
		w.WriteHeader(http.StatusOK)
		if err := json.NewEncoder(w).Encode(out); err != nil {
			logger.Error(err)
		}*/
	server := getInitialServerAddr()
	str := getServerStats(server)
	out := containerAddr{
		ServerIP:   "str",
		ServerPost: 32,
	}
	/*	out := containerAddr{
		ServerIP:   "456789",
		ServerPost: 32,
	}*/
	// logger.Debug("halog")
	// logger.Debug(str)
	// fmt.Println(str)

	w.Header().Set("content-type", "text/html")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(out); err != nil {
		logger.Error(err)
	}
	fmt.Println(str)

}

func main() {
	// getInitialServerInfo()
	m := martini.Classic()
	m.Post("/api/dispatcher/v1.0/container/create", dispatchContainer)

	m.Run()
}
