package main

import (
	"chat-system/log"
	"sync"
)

var (
	historyRecordCount = 50
	historyRecord      = NewHistory()
)

type HistoryRecord struct {
	history []string
	sync.RWMutex
	fileName string
}

func NewHistory() *HistoryRecord {
	historyRecord := &HistoryRecord{}
	return historyRecord
}

func (historyRecord *HistoryRecord) getHistory() []string {
	historyRecord.Lock()
	defer historyRecord.Unlock()
	return historyRecord.history
}

func (historyRecord *HistoryRecord) addHistory(str string) {
	historyRecord.Lock()
	defer historyRecord.Unlock()
	historyRecord.history = append(historyRecord.history, str)
	historyRecordLen := len(historyRecord.history)
	if historyRecordLen > historyRecordCount {
		historyRecord.history = historyRecord.history[historyRecordLen-historyRecordCount : historyRecordLen]
	}
	log.Debug("historyRecord.history count:%v",len(historyRecord.history))
}

