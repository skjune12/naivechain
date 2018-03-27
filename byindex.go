package main

type ByIndex []*Block

func (b ByIndex) Len() int {
	verboseMsg("(b ByIndex Len")
	return len(b)
}

func (b ByIndex) Swap(i, j int) {
	verboseMsg("(b ByIndex) Swap")
	b[i], b[j] = b[j], b[i]
}

func (b ByIndex) Less(i, j int) bool {
	verboseMsg("(b ByIndex) Less")
	return b[i].Index < b[j].Index
}
