package main

import (
	"fmt"
	"os"
	"crypto/sha1"
	"github.com/anacrolix/torrent/bencode"
	"net/http"
	"strings"
	"net/url"
	"github.com/google/uuid"
	"io"
	"net"
	"strconv"
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
	fmt.Println("len, buf", packageLength, buf)
	return buf
}

func main(){

	commandLineArgs := os.Args[1:]

	if len(commandLineArgs) != 1{
		fmt.Printf("Need 1 command line argument found %d\n", len(commandLineArgs))
		return
	}

	filePath := commandLineArgs[0]

	fileInfo, err := os.Stat(filePath)

	if err != nil{
		fmt.Printf("absolute path: %s does not exists or is not readable\n", filePath)
		return
	}

	err = os.Mkdir("temp", 0700)

	if err!=nil{
		fmt.Printf("error while creating /temp folder in local directory \n%s\n",err)
		return
	}

	data,err := os.ReadFile(filePath)

	if err!=nil{
		fmt.Printf("error while reading file at path: %s\n%s\n",filePath,err)
		return
	}

	currentWorkingDirectory, err := os.Getwd()

	if err!=nil{
		fmt.Printf("error while getting current working directory, error: %s", err)
		return
	}

	dataLength := int64(len(data))

	chunkArray := make([][]byte, 0)
	chunkSize := int64(16 * 1024)


	for i := int64(0); i < dataLength; i+=chunkSize{
		last := min(dataLength, i + chunkSize)
		chunkArray = append(chunkArray, data[i:last])
	}

	chunkHashes := make([]byte, 0)

	for i:=0;i<len(chunkArray);i++{
		os.WriteFile(fmt.Sprintf("%s/temp/file_chunk_%d",currentWorkingDirectory,i), chunkArray[i], 0700)
		chunkHash := sha1.Sum(chunkArray[i])
		chunkHashes = append(chunkHashes, chunkHash[:]...)
	}

	torrentDictionaryMap := make(map[string]interface{})
	infoDictionaryMap := make(map[string]interface{})

	torrentDictionaryMap["announce"] = "http://localhost:8080/announce"

	infoDictionaryMap["name"] = fileInfo.Name()
	infoDictionaryMap["piece length"] = chunkSize
	infoDictionaryMap["length"] = dataLength
	infoDictionaryMap["pieces"] = chunkHashes

	torrentDictionaryMap["info"] = infoDictionaryMap


	infoMapByte,err := bencode.Marshal(infoDictionaryMap)
	infohash := sha1.Sum(infoMapByte)

	fmt.Printf("info hash len = %d\n",len(infohash))

	torrentDictionaryByte, err := bencode.Marshal(torrentDictionaryMap)

	os.WriteFile(fmt.Sprintf("%s/temp/torrent.torrent",currentWorkingDirectory), torrentDictionaryByte, 0700)

	client := &http.Client{}

	tracker_url := "http://localhost:8080/announce?info_hash={info_hash}&peer_id={peer_id}&port={port}&uploaded={uploaded}&downloaded={downloaded}&left=0&event=started";

	tracker_url = strings.Replace(tracker_url, "{info_hash}", url.QueryEscape(string(infohash[:])), 1);
	tracker_url = strings.Replace(tracker_url, "{peer_id}", uuid.New().String(), 1);
	tracker_url = strings.Replace(tracker_url, "{port}", "8081", 1);
	tracker_url = strings.Replace(tracker_url, "{uploaded}", "0", 1);
	tracker_url = strings.Replace(tracker_url, "{downloaded}", "0", 1);

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

	ln,err := net.Listen("tcp", ":8081")

	if err!=nil{
		fmt.Printf("error in tcp connection %s", err)
	}

	conn,err := ln.Accept()

	if err!=nil{
		fmt.Printf("error in tcp connection %s", err)
	}
	
	for{
		chunkIndexBytes := readTcpPacket(conn)
		chunkIndex,_ := strconv.Atoi(string(chunkIndexBytes))

		fmt.Println("server recieved index = %d", chunkIndex)

		if chunkIndex==-1{
			conn.Close()
			break
		}

		writeTcpPacket(chunkArray[chunkIndex], conn)
		
	}

	// fmt.Printf("server recieved %s\n", string(buf)


}



