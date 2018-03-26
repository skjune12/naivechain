package main

import "fmt"

type Block struct {
	Index        int64  `json:index`
	PreviousHash string `json:previousHash`
	Timestamp    int64  `json:"timestamp"`
	Data         string `json:"data"` // TODO: Dataをbyte[]に変えて、データを保持したい。
	Hash         string `json:"hash"`
}

func (b *Block) String() string {
	return fmt.Sprintf("index: %d,previousHash:%s,timestamp:%d,data:%s,hash:%s", b.Index, b.PreviousHash, b.Timestamp, b.Data, b.Hash)
}