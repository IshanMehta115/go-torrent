package main

import (
	"fmt"
	"net/http"
	"encoding/json"
	"strconv"
	"net"
)

func announceHandler(w http.ResponseWriter, r *http.Request){
	if r.Method != "GET"{
		w.WriteHeader(400)
		fmt.Fprintf(w,"Method not supported\n")
		return
	}

	q := r.URL.Query()

	infohash := q.Get("info_hash")
	peerId := q.Get("peer_id")
	port := q.Get("port")
	uploaded := q.Get("uploaded")
	downloaded := q.Get("downloaded")
	left,_ := strconv.Atoi(q.Get("left"))
	ip,_,_ := net.SplitHostPort(r.RemoteAddr)

	if ip == "::1" {
		ip = "127.0.0.1"
	}	

	if left==0{
		peerMap,ok := trackerMap[infohash]
		if !ok{
			peerMap = make(map[string]interface{})
		}else{
			peerMap = trackerMap[infohash]
		}
	
		pm := make(map[string]interface{})
		pm["peerId"] = peerId
		pm["ip"] = ip
		pm["port"] = port
		pm["uploaded"] = uploaded
		pm["downloaded"] = downloaded
		pm["left"] = left
	
		peerMap[peerId] = pm
	
		trackerMap[infohash] = peerMap
	}
	
	respBytes, _ := json.Marshal(trackerMap[infohash])

	w.WriteHeader(200)
	w.Header().Set("content-type", "application/json")
	w.Write(respBytes)
}

var trackerMap = make(map[string]map[string]interface{})

func main(){

	http.HandleFunc("/announce", announceHandler)
	http.ListenAndServe(":8080", nil)
}
