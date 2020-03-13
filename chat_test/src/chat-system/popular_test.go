package main

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"strconv"
	"testing"
	"time"
)

func TestPopular(t *testing.T) {
	mgr := &PopularWordMgr{
		popularWords: map[int64](map[string]int){},
	}
	isRun := 0
	for {
		time.Sleep(time.Second)
		for i := 0; i < 100; i++ {
			mgr.add(strconv.Itoa(rand.Intn(15)))
		}
		isRun++
		if isRun > 3 {
			break
		}
	}
	poplist := mgr.get()
	popbytes, _ := json.Marshal(poplist)
	fmt.Println(string(popbytes))
}
