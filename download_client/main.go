package main

import (
	"fmt"	
	"os"
	"github.com/anacrolix/torrent/bencode"
	"crypto/sha1"
	"net/http"
	"net/url"
	"strings"
	"github.com/google/uuid"
	"io"
	"strconv"
	"encoding/json"
	"net"
	"math"
)

func writeTcpPacket(data []byte, conn net.Conn){
	chunkLength := len(data)
	chunkLengthStr := strconv.Itoa(chunkLength)

	for len(chunkLengthStr) < 10{
		chunkLengthStr = "0" + chunkLengthStr
	}

	conn.Write([]byte(chunkLengthStr))
	conn.Write(data)
}

func readTcpPacket(conn net.Conn) []byte{
	packetLengthBytes := readBytes(10, conn)
	packetLength,_ := strconv.Atoi(string(packetLengthBytes))
	return readBytes(packetLength, conn)
}

func readBytes(packageLength int, conn net.Conn) []byte{

	buf := make([]byte, packageLength)
	total := 0
	
	for total < packageLength{
		n,err := conn.Read(buf[total:])
		if err!=nil{
			return nil
		}
		total+=n
	}
	return buf
}

func readChunk(chunkIndex int, conn net.Conn) []byte{
	
	writeTcpPacket([]byte(strconv.Itoa(chunkIndex)), conn)
	chunkBytes := readTcpPacket(conn)

	return chunkBytes
}

func main(){

	currentWorkingDirectory, err := os.Getwd()

	if err!=nil{
		fmt.Printf("error while getting current working directory, error: %s", err)
		return
	}

	commandLineArgs := os.Args[1:]

	if len(commandLineArgs) != 1{
		fmt.Printf("Need 1 command line argument found %d\n", len(commandLineArgs))
		return
	}

	torrentFilePath := commandLineArgs[0]

	_, err = os.Stat(torrentFilePath)

	if err != nil{
		fmt.Printf("absolute path: %s does not exists or is not readable\n", torrentFilePath)
		return
	}

	torrentDictionaryByte, err := os.ReadFile(torrentFilePath)
	if err!=nil{
		fmt.Printf("error while reading file at path: %s\n%s\n",torrentFilePath,err)
		return
	}

	torrentDictionaryMap := make(map[string]interface{})
	bencode.Unmarshal(torrentDictionaryByte, &torrentDictionaryMap)

	fmt.Println(torrentDictionaryMap)


	infoDictionaryMap := torrentDictionaryMap["info"].(map[string]interface{})
	infoMapByte,err := bencode.Marshal(infoDictionaryMap)
	infohash := sha1.Sum(infoMapByte)

	tracker_hostname_bytes := torrentDictionaryMap["announce"].(string)
	tracker_hostname := string(tracker_hostname_bytes)

	data_length := infoDictionaryMap["length"].(int64)
	piece_length := infoDictionaryMap["piece length"].(int64)
	file_name := infoDictionaryMap["name"].(string)

	client := &http.Client{}

	tracker_url := "{tracker_url}?info_hash={info_hash}&peer_id={peer_id}&port={port}&uploaded={uploaded}&downloaded={downloaded}&left={left}&event=started";

	tracker_url = strings.Replace(tracker_url, "{tracker_url}", tracker_hostname, 1);
	tracker_url = strings.Replace(tracker_url, "{info_hash}", url.QueryEscape(string(infohash[:])), 1);
	tracker_url = strings.Replace(tracker_url, "{peer_id}", uuid.New().String(), 1);
	tracker_url = strings.Replace(tracker_url, "{port}", "8081", 1);
	tracker_url = strings.Replace(tracker_url, "{uploaded}", "0", 1);
	tracker_url = strings.Replace(tracker_url, "{downloaded}", "0", 1);
	tracker_url = strings.Replace(tracker_url, "{left}", strconv.FormatInt(data_length, 10), 1);

	fmt.Printf("tracker url = %s\n",tracker_url)

	req,err  := http.NewRequest("GET", tracker_url,nil)

	if err!=nil{
		fmt.Printf("error while creating http request object: %s\n",err)
		return
	}

	resp,err := client.Do(req)

	if err!=nil{
		fmt.Printf("error while making http call: %s\n",err)
		return
	}

	defer resp.Body.Close()

	fmt.Printf("Got response from tracker server: %d\n",resp.StatusCode)

	body, _ := io.ReadAll(resp.Body)
	fmt.Println("Body:", string(body))

	var peerMap map[string]map[string]interface{}

	_ = json.Unmarshal(body, &peerMap)

	fmt.Println(peerMap)

	peerIp := ""
	peerPort := ""	

	for _,value := range peerMap{
		peerIp = value["ip"].(string)
		peerPort = value["port"].(string)
		break
	}

	fmt.Printf("ip = %s, port = %s\n",peerIp,peerPort)

	conn,err := net.Dial("tcp", peerIp+":"+peerPort)

	if err!=nil{
		fmt.Printf("Error while creating TCP connection %s\n", err)
	}

	totalChunks := int(math.Ceil(float64(data_length) / float64(piece_length)))


	var fileData []byte

	for i:=0;i<totalChunks;i++{
		chunkByteArray := readChunk(i, conn)
		fileData = append(fileData, chunkByteArray...)
	}

	writeTcpPacket([]byte(strconv.Itoa(-1)), conn)
	conn.Close()

	os.WriteFile(fmt.Sprintf("%s/temp/%s",currentWorkingDirectory, file_name), fileData, 0700)
	
	fmt.Printf("done\n")
}