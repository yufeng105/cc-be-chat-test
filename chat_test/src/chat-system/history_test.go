package main

import (
	"fmt"
	"strconv"
	"testing"
)

func TestHistory(t *testing.T) {
	history := NewHistory()
	for i := 0; i < 100; i++ {
		history.addHistory(strconv.Itoa(i))
	}
	historyList := history.getHistory()
	for k, v := range historyList {
		fmt.Println(k, v)
	}
}
