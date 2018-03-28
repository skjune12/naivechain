package main

import (
	"encoding/json"
	"log"
	"net/http"
	"strings"
)

func handleBlocks(w http.ResponseWriter, r *http.Request) {
	verboseMsg("handleBlocks")

	bs, _ := json.Marshal(blockchain)
	w.Write(bs)
}

func handleMineBlock(w http.ResponseWriter, r *http.Request) {
	verboseMsg("handleMineBlock")

	var v struct {
		Data []byte `json:"data"`
	}
	decoder := json.NewDecoder(r.Body)
	defer r.Body.Close()

	err := decoder.Decode(&v)
	if err != nil {
		w.WriteHeader(http.StatusGone)
		log.Println("[API] Invalid block data : ", err.Error())
		w.Write([]byte("Invalid block data. " + err.Error() + "/n"))
		return
	}

	block := generateNextBlock(v.Data)

	addBlock(block)
	broadcast(responseLatestMsg())
}

func handlePeers(w http.ResponseWriter, r *http.Request) {
	verboseMsg("handlePeers")

	var slice []string

	for _, socket := range sockets {
		if socket.IsClientConn() {
			slice = append(slice, strings.Replace(socket.LocalAddr().String(), "ws://", "", 1))
		} else {
			slice = append(slice, socket.Request().RemoteAddr)
		}
	}

	bs, _ := json.Marshal(slice)
	w.Write(bs)
}

func handleAddPeer(w http.ResponseWriter, r *http.Request) {
	verboseMsg("handleAddPeer")

	var v struct {
		Peer string `json:"peer"`
	}

	decoder := json.NewDecoder(r.Body)
	defer r.Body.Close()

	err := decoder.Decode(&v)

	if err != nil {
		w.WriteHeader(http.StatusGone)

		log.Println("[API] invalid peer data : ", err.Error())
		w.Write([]byte("invalid peer data. " + err.Error()))
		return
	}
	connectToPeers([]string{v.Peer})
}
