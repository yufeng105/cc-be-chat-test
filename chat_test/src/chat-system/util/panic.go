package util

import (
	"fmt"
	"chat-system/log"
	"runtime"
)

const (
	maxStack  = 20
	separator = "---------------------------------------\n"
)

func HandlePanic() interface{} {
	if err := recover(); err != nil {
		errstr := fmt.Sprintf("%sruntime error: %v\ntraceback:\n", separator, err)

		i := 2
		for {
			pc, file, line, ok := runtime.Caller(i)
			if !ok || i > maxStack {
				break
			}
			errstr += fmt.Sprintf("    stack: %d %v [file: %s] [func: %s] [line: %d]\n", i-1, ok, file, runtime.FuncForPC(pc).Name(), line)
			i++
		}
		errstr += separator

		log.Debug(errstr)

		return err
	}
	return nil
}

func Safe(cb func()) {
	defer HandlePanic()
	cb()
}

func Go(cb func()) {
	go func() {
		defer HandlePanic()
		cb()
	}()
}