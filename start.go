package main

import (
	"encoding/json"
	"github.com/Sirupsen/logrus"
	"github.com/go-martini/martini"
	"net/http"
	// "strconv"
)

var logger = logrus.New()

type containerAddr struct {
	ServerIP   string
	ServerPost int
}
type imageName struct {
	iName string
}

func dispatchContainer(w http.ResponseWriter, r *http.Request) {
	// 接受image-name，返回json（server-ip，server-port）
	// return "hello world " + param["word"]
	// var ca containerAddr
	var in imageName //the image name received
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		logger.Warnf("error decoding image: %s", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		// return "error"
	}
	// ha := "hah"
	// return "dfsdf}"

	out := containerAddr{
		ServerIP:   "456789",
		ServerPost: 32,
	}
	w.Header().Set("content-type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(out); err != nil {
		logger.Error(err)
	}

}

func main() {
	m := martini.Classic()
	m.Post("/api/dispatcher/v1.0/container/create", dispatchContainer)

	m.Run()
}
