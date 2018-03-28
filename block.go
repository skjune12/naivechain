package main

import "fmt"

type Block struct {
	Index        int64  `json:index`
	PreviousHash string `json:previousHash`
	Timestamp    int64  `json:"timestamp"`
	Data         []byte `json:"data"`
	Hash         string `json:"hash"`
}

func (b *Block) String() string {
	verboseMsg("(b *Block) String()")
	return fmt.Sprintf("index: %d,previousHash:%s,timestamp:%d,data:%s,hash:%s", b.Index, b.PreviousHash, b.Timestamp, b.Data, b.Hash)
}
