package log

import (
	"io"
	"log"
	"os"
	"sync"
)

// [info ] 颜色为蓝色，[error] 为红色
// log.Lshortfile 支持显示文件名和代码行号
var (
	errorLog = log.New(os.Stdout, "\033[31m[error]\033[0m ", log.LstdFlags|log.Lshortfile)
	infoLog  = log.New(os.Stdout, "\033[34m[info ]\033[0m ", log.LstdFlags|log.Lshortfile)
	loggers  = []*log.Logger{errorLog, infoLog}
	mutex    sync.Mutex
)

// log method
var (
	Error  = errorLog.Println
	Errorf = errorLog.Printf
	Info   = infoLog.Println
	Infof  = infoLog.Printf
)

// log level
const (
	InfoLevel = iota
	ErrorLevel
	Disabled
)

// 设置日志等级
func SetLevel(level int) {
	mutex.Lock()
	defer mutex.Unlock()

	// 重置输出源
	for _, logger := range loggers {
		logger.SetOutput(os.Stdout)
	}

	if ErrorLevel < level {
		errorLog.SetOutput(io.Discard)
	}

	if InfoLevel < level {
		infoLog.SetOutput(io.Discard)
	}
}
