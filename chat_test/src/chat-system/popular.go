package main

import (
	"sort"
	"sync"
	"time"
)

var (
	popularHoldTime = int64(5)

	popularWordMgr = &PopularWordMgr{
		popularWords: map[int64](map[string]int){},
	}
)

type WordUnit struct {
	Word  string
	Count int
}

type PopularWordMgr struct {
	sync.RWMutex
	popularWords map[int64](map[string]int)
}

func (mgr *PopularWordMgr) clear() {
	curTime := time.Now().Unix()
	if len(mgr.popularWords) > int(popularHoldTime) {
		oldTime := curTime - 1
		mgr.Lock()
		_, have := mgr.popularWords[oldTime]
		if have {
			delete(mgr.popularWords, oldTime)
		}
		mgr.Unlock()
	}
}

func (mgr *PopularWordMgr) add(msg string) {
	curTime := time.Now().Unix()
	mgr.Lock()
	words, have := mgr.popularWords[curTime]
	if have {
		words[msg] += 1
	} else {
		mgr.popularWords[curTime] = map[string]int{msg: 1}
	}
	mgr.Unlock()
	mgr.clear()
}

func (mgr *PopularWordMgr) get() []string {
	curTime := time.Now().Unix()
	start := curTime - popularHoldTime
	tempWord := map[string]int{}
	mgr.Lock()
	for i := start + 1; i <= curTime; i++ {
		words, have := mgr.popularWords[i]
		if have {
			for word, sum := range words {
				_, have := tempWord[word]
				if have {
					tempWord[word] += sum
				} else {
					tempWord[word] = sum
				}
			}
		}
	}
	mgr.Unlock()
	if len(tempWord) == 0 {
		return nil
	}

	verb := []WordUnit{}
	for word, sum := range tempWord {
		verb = append(verb, WordUnit{Word: word, Count: sum})
	}

	sort.Slice(verb, func(i, j int) bool {
		return verb[i].Count > verb[j].Count
	})

	top10 := []string{}
	for i := 0; i < len(verb) && i < 10; i++ {
		top10 = append(top10, verb[i].Word)
	}
	return top10
}
