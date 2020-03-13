package log

import (
	"fmt"
	"testing"
)

// 按天切割日志文件，不限制单个日志文件大小
func TestLog1(t *testing.T) {
	rootdir := "./logs1/" //日志根目录
	newdir := false       //如果为true，每次启动在"./logs1/"目录下新建一个目录，本次启动后的所有日志在该目录下
	subdir := false       //如果为true，每次启动在"./logs1/newdir/"目录下按照SetSubDirFormat的值进行日志目录切割
	SetLogDir(rootdir, newdir, subdir)
	SetLogFilenameTimeFormat("20060102")
	SetLogTimeFormat("[APP] 2006/01/02 - 15:04:05")
	// SetSubDirFormat("20060102/")
	SetLogLevel(LOG_LEVEL_INFO)
	SetLogType(LogCmd | LogFile)

	for i := 0; i < 100; i++ {
		Debug(fmt.Sprintf("log %d", i))
		Info(fmt.Sprintf("log %d", i))
		Warn(fmt.Sprintf("log %d", i))
		Error(fmt.Sprintf("log %d", i))
	}
}

// 按天切割日志文件，并限制单个日志文件大小
func TestLog2(t *testing.T) {
	rootdir := "./logs2/" //日志根目录
	newdir := true        //如果为true，每次启动在"./logs2/"目录下新建一个目录，本次启动后的所有日志在该目录下，如果SetMaxLogFileSize大于0，newdir应设置为true
	subdir := false       //如果为true，每次启动在"./logs2/newdir/"目录下按照SetSubDirFormat的值进行日志目录切割，通常设置为false
	SetLogDir(rootdir, newdir, subdir)
	SetLogFilenameTimeFormat("20060102")
	SetLogTimeFormat("[APP] 2006/01/02 - 15:04:05")
	SetMaxLogFileSize(1024)
	SetLogType(LogCmd | LogFile)

	for i := 0; i < 100; i++ {
		Debug(fmt.Sprintf("log %d", i))
		Info(fmt.Sprintf("log %d", i))
		Warn(fmt.Sprintf("log %d", i))
		Error(fmt.Sprintf("log %d", i))
	}
}
