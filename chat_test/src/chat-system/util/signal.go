package util

import (
	"github.com/jb00007/common/log"
	"os"
	"os/signal"
	//"syscall"
)

var (
	inited = false
)

func HandleSignal(handler func(sig os.Signal)) {
	if !inited {
		inited = true
		chSignal := make(chan os.Signal, 1)
		//signal.Notify(chSignal, syscall.SIGHUP, syscall.SIGQUIT, syscall.SIGTERM, syscall.SIGINT)
		signal.Notify(chSignal)
		for {
			if sig, ok := <-chSignal; ok {
				log.Debug("Recv Signal: %v", sig)

				if handler != nil {
					handler(sig)
				}
			} else {
				return
			}
		}
	}
}
