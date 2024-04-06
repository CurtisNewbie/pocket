package main

import (
	"flag"
	"log"
	"os"
)

var (
	debug = flag.Bool("debug", false, "enable debug log")
)

var (
	_debugLogger  *log.Logger
	_debugLogPipe chan func(logger *log.Logger)
)

func main() {
	flag.Parse()
	if *debug {
		f, _ := os.Create("debug.log")
		defer f.Close()
		log.SetOutput(f)
		_debugLogger = log.Default()
		_debugLogPipe = make(chan func(*log.Logger), 50)
		go func() {
			for f := range _debugLogPipe {
				if *debug {
					f(_debugLogger)
				}
			}
		}()
	}

	if err := NewApp().Run(); err != nil {
		panic(err)
	}
}

func DebugLog(fmt string, args ...any) {
	if _debugLogPipe == nil {
		return
	}
	_debugLogPipe <- func(logger *log.Logger) {
		logger.Printf(fmt, args...)
	}
}

func DebugLogFunc(f func(logger *log.Logger)) {
	if _debugLogPipe == nil {
		return
	}
	_debugLogPipe <- f
}
