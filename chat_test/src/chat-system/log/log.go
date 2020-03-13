package log

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"runtime"
	"strings"
	"sync"
	"time"
	"unsafe"
)

const (
	//LOG_NONE = iota
	LogNone = 0
	LogCmd  = 0x1 << 0
	LogFile = 0x1 << 1
	LogUser = 0x1 << 2
	//LOG_MAX
)

const (
	LOG_LEVEL_DEBUG = iota
	LOG_LEVEL_INFO
	LOG_LEVEL_WARN
	LOG_LEVEL_ERROR
	LOG_LEVEL_NONE
)

var (
	TAG_NULL               = "--"
	LOG_DIR_NAME_FORMAT    = "20060102-150405/"
	LOG_SUBDIR_NAME_FORMAT = "20060102/"
	LOG_FILE_NAME_FORMAT   = "20060102"
	LOG_SUFIX              = ".log"
	/*LOG_DIR_NAME_FORMAT    = "20060102-150405/"
	LOG_SUBDIR_NAME_FORMAT = "20060102-150405/"
	LOG_FILE_NAME_FORMAT   = "20060102-150405"*/
	LOG_STR_FORMAT = "2006-01-02 15:04:05.000"
	logsep         = ""
	callerdepth    = 1
)

var (
	logmtx       = sync.Mutex{}
	logdir       = "./logs/"
	logdirinited = false

	currlogdir                  = ""
	logfile       *os.File      = nil
	filewriter    *bufio.Writer = nil
	logfilename                 = ""
	logfilesubnum               = 0
	logfilesize                 = 0

	maxfilesize   = 0 // (1024 * 1024 * 64)
	logdebugtype  = LogCmd
	loginfotype   = LogCmd
	logwarntype   = LogCmd
	logerrortype  = LogCmd
	logactiontype = LogCmd

	//logdebug     = true
	loglevel     = LOG_LEVEL_DEBUG
	syncinterval = time.Second * 30
	saveEach     = false

	Printf  = fmt.Printf
	Sprintf = fmt.Sprintf
	Println = fmt.Println

	// LOG_IDX = 0
	// LOG_TAG = "log"

	// logtags = map[int]string{
	// 	LOG_IDX: LOG_TAG,
	// }

	/*logconf = map[string]int{
		"Debug":  LogCmd,
		"Info":   LogCmd,
		"Warn":   LogCmd,
		"Error":  LogCmd,
		"Action": LogCmd,
	}*/

	logticker    *time.Ticker = nil
	enablebufio               = false
	enablenewdir              = false
	enableSubdir              = false
	//chsynclogfile chan struct{} = nil
	output     io.Writer = nil
	userlogger           = func(str string) {
		if output != nil {
			x := (*[2]uintptr)(unsafe.Pointer(&str))
			h := [3]uintptr{x[0], x[1], x[1]}
			output.Write(*(*[]byte)(unsafe.Pointer(&h)))
			//output.WriteString(str)
		}
	}

	// now      = time.Now()
	inittime = time.Now()

	initsync = false

	formater func(format string, v ...interface{}) string = nil
)

// type Writer interface {
// 	Write(p []byte) (n int, err error)
// 	WriteString(s string) (n int, err error)
// }

type FileWriter struct {
}

func (w *FileWriter) Write(p []byte) (n int, err error) {
	n, err = writebuftofile(p)
	return n, err
}

func newFile(path string) (*os.File, error) {
	file, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, os.ModePerm)
	if err != nil {
		Printf("log newFile Error: %s, %s\n", path, err.Error())
		return nil, err
	}
	size, err := file.Seek(0, os.SEEK_END)
	logfilesize = int(size)
	if enablebufio {
		if filewriter == nil {
			filewriter = bufio.NewWriter(file)
		} else {
			filewriter.Reset(file)
		}
	}

	return file, err
}

func checkFile(now time.Time, newsize int) bool {
	var (
		err          error = nil
		nowlogdir          = ""
		currfilename       = ""
	)

	if enablenewdir {
		nowlogdir = logdir + inittime.Format(LOG_DIR_NAME_FORMAT)
	} else {
		nowlogdir = logdir
	}
	if enableSubdir {
		nowlogdir += now.Format(LOG_SUBDIR_NAME_FORMAT)
	}
	//if nowlogdir != currlogdir {
	currlogdir = nowlogdir
	if err := makeDir(currlogdir); err != nil {
		Println("log checkFile Failed")
		return false
	}
	//}
	if logfilesubnum == 0 {
		currfilename = Sprintf("%s%s%s", currlogdir, now.Format(LOG_FILE_NAME_FORMAT), LOG_SUFIX)
	} else {
		currfilename = Sprintf("%s%s%s.%04d", currlogdir, now.Format(LOG_FILE_NAME_FORMAT), LOG_SUFIX, logfilesubnum)
	}

	if logfilename != currfilename {
		logfilesubnum = 0
		logfilename = Sprintf("%s%s%s", currlogdir, now.Format(LOG_FILE_NAME_FORMAT), LOG_SUFIX)
		if enablebufio {
			if filewriter != nil {
				filewriter.Flush()
				if logfile != nil {
					logfile.Close()
				}
			}
		} else {
			if logfile != nil {
				logfile.Close()
			}
		}

		// nowlogdir := nowlogdir + inittime.Format(LOG_DIR_NAME_FORMAT)
		// if enableSubdir {
		// 	logdir += now.Format(LOG_SUBDIR_NAME_FORMAT)
		// }
		// if nowlogdir != currlogdir {
		// 	currlogdir = nowlogdir
		// 	if err := makeDir(currlogdir); err != nil {
		// 		Println("log checkFile Failed")
		// 		return false
		// 	}
		// }
		logfile, err = newFile(logfilename)
		if err != nil {
			Println("log checkFile Failed")
			return false
		} else {
			logfilesize = 0
			/*if logticker != nil {
				logticker.Reset(syncinterval)
			}*/
			return true
		}
	} else {
		if maxfilesize > 0 && logfilesize+newsize > maxfilesize {
			// if logfilename == currfilename {
			// logfilesubnum++
			// logfilename = Sprintf("%s.%s.%d", now.Format(LOG_FILE_NAME_FORMAT), LOG_SUFIX, logfilesubnum)
			// } else {
			// 	logfilesubnum = 0
			// 	logfilename = Sprintf("%s", now.Format(LOG_FILE_NAME_FORMAT))
			// }
			logfilesubnum++
			logfilename = Sprintf("%s%s%s.%04d", currlogdir, now.Format(LOG_FILE_NAME_FORMAT), LOG_SUFIX, logfilesubnum)
			if enablebufio {
				if filewriter != nil {
					filewriter.Flush()
					if logfile != nil {
						logfile.Close()
					}
				}
			} else {
				if logfile != nil {
					logfile.Close()
				}
			}
			// nowlogdir := nowlogdir + inittime.Format(LOG_DIR_NAME_FORMAT)
			// if enableSubdir {
			// 	nowlogdir += now.Format(LOG_SUBDIR_NAME_FORMAT)
			// }
			// if nowlogdir != currlogdir {
			// 	currlogdir = nowlogdir
			// 	if err := makeDir(currlogdir); err != nil {
			// 		Println("log checkFile Failed")
			// 		return false
			// 	}
			// }
			logfile, err = newFile(logfilename)
			if err != nil {
				Println("log checkFile Failed")
				return false
			}
			logfilesize = 0
			/*if logticker != nil {
				logticker.Reset(syncinterval)
			}*/
		}
	}

	if !initsync {
		startSync()
	}

	return true
}

func makeDir(path string) error {
	err := os.MkdirAll(path, os.ModePerm)
	return err
}

func setSyncLogFileInterval(interval time.Duration) {
	syncinterval = interval
}

func initLogDirAndFile() bool {
	if !logdirinited {
		logdirinited = true
		//currlogdir = logdir + time.Now().Format("20060102-150405/")
		//inittime := time.Now()
		if enablenewdir {
			currlogdir = logdir + inittime.Format(LOG_DIR_NAME_FORMAT)
		} else {
			currlogdir = logdir
		}
		if enableSubdir {
			currlogdir += inittime.Format(LOG_SUBDIR_NAME_FORMAT)
		}
		
		err := makeDir(currlogdir)
		if err != nil {
			Printf("log initLogDirAndFile Error: %s-%v\n",currlogdir, err)
			return false
		}
	}
	return true
}

func writebuftofile(p []byte) (int, error) {
	logmtx.Lock()
	defer logmtx.Unlock()

	checkFile(time.Now(), len(p))
	var (
		n   int
		err error
	)
	if enablebufio {
		n, err = filewriter.Write(p)
	} else {
		n, err = logfile.Write(p)
	}
	if err != nil || n != len(p) {
		Printf("log writetofile Failed: %d/%d wrote, Error: %v\n", n, len(p), err)
	} else {
		logfilesize += len(p)
	}
	return n, err
}

func writetofile(now time.Time, str string) {
	logmtx.Lock()
	defer logmtx.Unlock()

	checkFile(now, len(str))
	var (
		n   int
		err error
	)
	if enablebufio {
		if filewriter != nil {
			n, err = filewriter.WriteString(str)
		}
	} else {
		if logfile != nil {
			n, err = logfile.WriteString(str)
		}
	}
	if saveEach {
		if enablebufio {
			if filewriter != nil {
				filewriter.Flush()
			}
		} else {
			if logfile != nil {
				logfile.Sync()
			}
		}
	}
	if err != nil || n != len(str) {
		Printf("log writetofile Failed: %d/%d wrote, Error: %v\n", n, len(str), err)
	} else {
		logfilesize += len(str)
	}
}

func syncLogFile() {
	logmtx.Lock()
	defer logmtx.Unlock()

	if enablebufio {
		if filewriter != nil {
			filewriter.Flush()
		}
	} else {
		if logfile != nil {
			logfile.Sync()
		}
	}
}

func startSync() {
	if !initsync && !saveEach {
		initsync = true
		go func() {
			defer func() {
				recover()
			}()

			logticker = time.NewTicker(syncinterval)

			for {
				//select {
				//case _, ok := <-logticker.C:
				_, ok := <-logticker.C
				syncLogFile()
				if !ok {
					return
				}
				/*case _, ok := <-chsynclogfile:
					syncLogFile()
					if !ok {
						return
					}
				}*/
				//logticker.Reset(syncinterval)
			}
		}()
	}
}

func Debug(format string, v ...interface{}) {
	if LOG_LEVEL_DEBUG >= loglevel {
		now := time.Now()
		_, file, line, ok := runtime.Caller(callerdepth)
		if !ok {
			file = "???"
			line = -1
		} else {
			pos := strings.LastIndex(file, "/")
			if pos >= 0 {
				file = file[pos+1:]
			}
		}
		s := ""
		if formater != nil {
			s = formater(format, v...)
		} else {
			s = strings.Join([]string{now.Format(LOG_STR_FORMAT), Sprintf(" [Debug] [%s:%d] ", file, line), Sprintf(format, v...), "\n"}, logsep)
		}
		if logdebugtype&LogFile != 0 {
			writetofile(now, s)
		}
		if logdebugtype&LogCmd != 0 {
			Printf(s)
		}
		if logdebugtype&LogUser != 0 {
			userlogger(s)
		}
	}
}

func Info(format string, v ...interface{}) {
	if LOG_LEVEL_INFO >= loglevel {
		now := time.Now()
		_, file, line, ok := runtime.Caller(callerdepth)
		if !ok {
			file = "???"
		} else {
			pos := strings.LastIndex(file, "/")
			if pos >= 0 {
				file = file[pos+1:]
			}
		}
		s := ""
		if formater != nil {
			s = formater(format, v...)
		} else {
			s = strings.Join([]string{now.Format(LOG_STR_FORMAT), Sprintf(" [ Info] [%s:%d] ", file, line), Sprintf(format, v...), "\n"}, logsep)
		}
		if loginfotype&LogFile != 0 {
			writetofile(now, s)
		}
		if loginfotype&LogCmd != 0 {
			Printf(s)
		}
		if loginfotype&LogUser != 0 {
			userlogger(s)
		}
	}
}

func Warn(format string, v ...interface{}) {
	if LOG_LEVEL_WARN >= loglevel {
		now := time.Now()
		_, file, line, ok := runtime.Caller(callerdepth)
		if !ok {
			file = "???"
		} else {
			pos := strings.LastIndex(file, "/")
			if pos >= 0 {
				file = file[pos+1:]
			}
		}
		s := ""
		if formater != nil {
			s = formater(format, v...)
		} else {
			s = strings.Join([]string{now.Format(LOG_STR_FORMAT), Sprintf(" [ Warn] [%s:%d] ", file, line), Sprintf(format, v...), "\n"}, logsep)
		}
		if logwarntype&LogFile != 0 {
			writetofile(now, s)
		}
		if logwarntype&LogCmd != 0 {
			Printf(s)
		}
		if logwarntype&LogUser != 0 {
			userlogger(s)
		}
	}
}

func Error(format string, v ...interface{}) {
	if LOG_LEVEL_ERROR >= loglevel {
		now := time.Now()
		_, file, line, ok := runtime.Caller(callerdepth)
		if !ok {
			file = "???"
		} else {
			pos := strings.LastIndex(file, "/")
			if pos >= 0 {
				file = file[pos+1:]
			}
		}
		s := ""
		if formater != nil {
			s = formater(format, v...)
		} else {
			s = strings.Join([]string{now.Format(LOG_STR_FORMAT), Sprintf(" [Error] [%s:%d] ", file, line), Sprintf(format, v...), "\n"}, logsep)
		}
		if logerrortype&LogFile != 0 {
			writetofile(now, s)
		}
		if logerrortype&LogCmd != 0 {
			Printf(s)
		}
		if logerrortype&LogUser != 0 {
			userlogger(s)
		}
	}
}

func DebugX(format string, v ...interface{}) {
	if LOG_LEVEL_DEBUG >= loglevel {
		now := time.Now()
		s := strings.Join([]string{now.Format(LOG_STR_FORMAT), " [Debug] ", Sprintf(format, v...), "\n"}, logsep)
		if logdebugtype&LogFile != 0 {
			writetofile(now, s)
		}
		if logdebugtype&LogCmd != 0 {
			Printf(s)
		}
		if logdebugtype&LogUser != 0 {
			userlogger(s)
		}
	}
}

func InfoX(format string, v ...interface{}) {
	if LOG_LEVEL_INFO >= loglevel {
		now := time.Now()
		s := strings.Join([]string{now.Format(LOG_STR_FORMAT), " [ Info] ", Sprintf(format, v...), "\n"}, logsep)
		if loginfotype&LogFile != 0 {
			writetofile(now, s)
		}
		if loginfotype&LogCmd != 0 {
			Printf(s)
		}
		if loginfotype&LogUser != 0 {
			userlogger(s)
		}
	}
}

func WarnX(format string, v ...interface{}) {
	if LOG_LEVEL_WARN >= loglevel {
		now := time.Now()
		s := strings.Join([]string{now.Format(LOG_STR_FORMAT), " [ Warn] ", Sprintf(format, v...), "\n"}, logsep)
		if logwarntype&LogFile != 0 {
			writetofile(now, s)
		}
		if logwarntype&LogCmd != 0 {
			Printf(s)
		}
		if logwarntype&LogUser != 0 {
			userlogger(s)
		}
	}
}

func ErrorX(format string, v ...interface{}) {
	if LOG_LEVEL_ERROR >= loglevel {
		now := time.Now()
		s := strings.Join([]string{now.Format(LOG_STR_FORMAT), " [Error] ", Sprintf(format, v...), "\n"}, logsep)
		if logerrortype&LogFile != 0 {
			writetofile(now, s)
		}
		if logerrortype&LogCmd != 0 {
			Printf(s)
		}
		if logerrortype&LogUser != 0 {
			userlogger(s)
		}
	}
}

func SetLogDir(dir string, newdir bool, subdir bool) {
	if strings.HasSuffix(dir, "/") || strings.HasSuffix(dir, "\\") {
		logdir = dir
	} else {
		logdir = dir + "/"
	}
	enablenewdir = newdir
	enableSubdir = subdir
}

func SetFormater(f func(format string, v ...interface{}) string) {
	formater = f
}

func SetOutput(o io.Writer) {
	output = o
}

func SetLogLevel(level int) {
	if level >= 0 && level <= LOG_LEVEL_NONE {
		loglevel = level
	} else {
		panic(errors.New("invalid log level"))
		Printf("log SetLogLevel Error: Invalid Level - %d\n", level)
	}
}

func SetMaxLogFileSize(size int) {
	// if size > 0 {
	// 	maxfilesize = size
	// } else {
	// 	Printf("log SetMaxLogFileSize Error: Invalid size - %d\n", size)
	// }
	maxfilesize = size
}

func SetDebugLogType(t int) {
	logdebugtype = t
	if t&LogFile == LogFile {
		initLogDirAndFile()
	}
}

func SetInfoLogType(t int) {
	loginfotype = t
	if t&LogFile == LogFile {
		initLogDirAndFile()
	}
}

func SetWarnLogType(t int) {
	logwarntype = t
	if t&LogFile == LogFile {
		initLogDirAndFile()
	}
}

func SetErrorLogType(t int) {
	logerrortype = t
	if t&LogFile == LogFile {
		initLogDirAndFile()
	}
}

func SetLogType(t int) {
	logdebugtype = t
	loginfotype = t
	logwarntype = t
	logerrortype = t
	if t&LogFile == LogFile {
		initLogDirAndFile()
	}
}

func SetLogTimeFormat(format string) {
	LOG_STR_FORMAT = format
}

func SetSubDirFormat(format string) {
	if strings.HasSuffix(format, "/") || strings.HasSuffix(format, "\\") {
		LOG_SUBDIR_NAME_FORMAT = format
	} else {
		LOG_SUBDIR_NAME_FORMAT = format + "/"
	}
}

func SetLogFilenameTimeFormat(format string) {
	LOG_FILE_NAME_FORMAT = format
}

func SetLogFileSuffix(sufix string) {
	LOG_SUFIX = sufix
}

func EnableBufio() {
	enablebufio = true
}

func DisableBufio() {
	enablebufio = true
}

func SaveEach() {
	saveEach = true
}

func Save() {
	syncLogFile()
}
