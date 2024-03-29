package main

import (
	"crypto/sha256"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/gorilla/mux"
	"golang.org/x/net/websocket"
)

const (
	queryLatest = iota
	queryAll
	responseBlockchain
)

var genesisBlock = initGenesisBlock()

func initGenesisBlock() *Block {
	b := &Block{}
	b.Index = 0
	b.PreviousHash = "0"
	b.Timestamp = 1465154705
	b.Data = []byte("My genesis block!")
	b.Hash = fmt.Sprintf("%x", sha256.Sum256([]byte(fmt.Sprintf("%d%s%d%x", b.Index, b.PreviousHash, b.Timestamp, b.Data))))
	return b
}

var (
	sockets      []*websocket.Conn
	blockchain   = []*Block{genesisBlock}
	httpAddr     = flag.String("api", ":3001", "api server address")
	p2pAddr      = flag.String("p2p", ":6001", "p2p server address")
	initialPeers = flag.String("peers", "ws://localhost:6001", "initial peers")
	verbose      = flag.Bool("verbose", false, "verbose")
)

type ResponseBlockchain struct {
	Type int    `json:"type"`
	Data []byte `json:"data"`
}

func errFatal(msg string, err error) {
	if err != nil {
		log.Fatal(msg, err)
	}
}

func verboseMsg(msg string) {
	if *verbose == true {
		fmt.Fprintf(os.Stderr, "\x1b[31m-> %s\x1b[0m\n", msg)
	}
}

func connectToPeers(peersAddr []string) {
	verboseMsg("connectToPeers()")
	for _, peer := range peersAddr {
		if peer == "" {
			continue
		}

		ws, err := websocket.Dial(peer, "", peer)

		if err != nil {
			log.Println("Dial to peer", err)
			continue
		}

		initConnection(ws)
	}
}

func initConnection(ws *websocket.Conn) {
	verboseMsg("initConnection()")
	go wsHandleP2P(ws)

	log.Println("query latest block.")
	ws.Write(queryLatestMsg())
}

func wsHandleP2P(ws *websocket.Conn) {
	verboseMsg("wsHandleP2P()")
	var (
		v    = &ResponseBlockchain{}
		peer = ws.LocalAddr().String()
	)
	sockets = append(sockets, ws)

	for {
		var msg []byte
		err := websocket.Message.Receive(ws, &msg)

		if err == io.EOF {
			log.Printf("P2P peer[%s] shutdown, remove it from peers pool.\n", peer)
			break
		}

		if err != nil {
			log.Println("Can't receive P2P message from ", peer, err.Error())
			break
		}

		log.Printf("Received[from %s]: %s\n", peer, msg)
		err = json.Unmarshal(msg, v)
		errFatal("invalid P2P message", err)

		switch v.Type {
		case queryLatest:
			v.Type = responseBlockchain
			bs := responseLatestMsg()
			log.Printf("responseLatestMsg: %s\n", bs)
			ws.Write(bs)

		case queryAll:
			d, _ := json.Marshal(blockchain)
			v.Type = responseBlockchain
			v.Data = []byte(d)
			bs, _ := json.Marshal(v)
			log.Printf("responseChainMsg: %s\n", bs)
			ws.Write(bs)

		case responseBlockchain:
			handleBlockchainResponse([]byte(v.Data))
		}
	}
}

func getLatestBlock() (block *Block) {
	verboseMsg("getLatestBlock()")
	return blockchain[len(blockchain)-1]
}

func responseLatestMsg() (bs []byte) {
	verboseMsg("responseLatestMsg()")
	var v = &ResponseBlockchain{Type: responseBlockchain}
	d, _ := json.Marshal(blockchain[len(blockchain)-1:])
	v.Data = []byte(d)
	bs, _ = json.Marshal(v)
	return
}

func queryLatestMsg() []byte {
	verboseMsg("queryLatestMsg()")
	return []byte(fmt.Sprintf("{\"type\": %d}", queryLatest))
}

func queryAllMsg() []byte {
	verboseMsg("queryAllMsg()")
	return []byte(fmt.Sprintf("{\"type\": %d}", queryAll))
}

func calculateHashForBlock(b *Block) string {
	verboseMsg("calculateHashForBlock()")
	return fmt.Sprintf("%x", sha256.Sum256([]byte(fmt.Sprintf("%d%s%d%x", b.Index, b.PreviousHash, b.Timestamp, b.Data))))
}

func generateNextBlock(data []byte) (nb *Block) {
	verboseMsg("generateNextBlock()")
	var previousBlock = getLatestBlock()
	nb = &Block{
		Data:         data,
		PreviousHash: previousBlock.Hash,
		Index:        previousBlock.Index + 1,
		Timestamp:    time.Now().Unix(),
	}
	nb.Hash = calculateHashForBlock(nb)
	return
}

func addBlock(b *Block) {
	verboseMsg("addBlock")
	if isValidNewBlock(b, getLatestBlock()) {
		blockchain = append(blockchain, b)
	}
}

func isValidNewBlock(nb, pb *Block) (ok bool) {
	verboseMsg("isValidNewBlock")
	if nb.Hash == calculateHashForBlock(nb) &&
		pb.Index+1 == nb.Index &&
		pb.Hash == nb.PreviousHash {
		ok = true
	}
	return
}

func isValidChain(bc []*Block) bool {
	verboseMsg("isValidChain")

	if bc[0].String() != genesisBlock.String() {
		log.Println("No same GenesisBlock.", bc[0].String())
		return false
	}

	var temp = []*Block{bc[0]}

	for i := 1; i < len(bc); i++ {
		if isValidNewBlock(bc[i], temp[i-1]) {
			temp = append(temp, bc[i])
		} else {
			return false
		}
	}
	return true
}

func replaceChain(bc []*Block) {
	verboseMsg("replaceChain")

	if isValidChain(bc) && len(bc) > len(blockchain) {
		log.Println("Received blockchain is valid. Replacing current blockchain with received blockchain.")
		blockchain = bc
		broadcast(responseLatestMsg())
	} else {
		log.Println("Received blockchain invalid.")
	}
}

func broadcast(msg []byte) {
	verboseMsg("broadcast")

	for n, socket := range sockets {
		_, err := socket.Write(msg)
		if err != nil {
			log.Printf("peer [%s] disconnected.", socket.RemoteAddr().String())
			sockets = append(sockets[0:n], sockets[n+1:]...)
		}
	}
}

func handleBlockchainResponse(msg []byte) {
	verboseMsg("handleBlockchainResponse")

	var receivedBlocks = []*Block{}

	err := json.Unmarshal(msg, &receivedBlocks)
	errFatal("invalid blockchain", err)

	sort.Sort(ByIndex(receivedBlocks))

	latestBlockReceived := receivedBlocks[len(receivedBlocks)-1]
	latestBlockHeld := getLatestBlock()

	if latestBlockReceived.Index > latestBlockHeld.Index {
		log.Printf("blockchain possibly behind. We got: %d Peer got: %d", latestBlockHeld.Index, latestBlockReceived.Index)

		if latestBlockHeld.Hash == latestBlockReceived.PreviousHash {
			log.Println("We can append the received block to our chain.")
			blockchain = append(blockchain, latestBlockReceived)
		} else if len(receivedBlocks) == 1 {
			log.Println("We have to query the chain from our peer.")
			broadcast(queryAllMsg())
		} else {
			log.Println("Received blockchain is longer than current blockchain.")
			replaceChain(receivedBlocks)
		}
	} else {
		log.Println("received blockchain is not longer than current blockchain. Do nothing.")
	}
}

func main() {
	flag.Parse()
	router := mux.NewRouter()

	connectToPeers(strings.Split(*initialPeers, ","))

	router.HandleFunc("/blocks", handleBlocks).Methods("GET")
	router.HandleFunc("/block/{index}", handleBlock).Methods("GET")
	router.HandleFunc("/mine_block", handleMineBlock).Methods("POST")
	router.HandleFunc("/peers", handlePeers).Methods("GET")
	router.HandleFunc("/add_peer", handleAddPeer).Methods("POST")

	go func() {
		log.Println("Listen HTTP on", *httpAddr)
		errFatal("start api server", http.ListenAndServe(*httpAddr, router))
	}()

	http.Handle("/", websocket.Handler(wsHandleP2P))
	log.Println("Listen P2P on ", *p2pAddr)
	errFatal("start p2p server", http.ListenAndServe(*p2pAddr, nil))
}
