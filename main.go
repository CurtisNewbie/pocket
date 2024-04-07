package main

import (
	"flag"
	"log"
	"os"
	"strings"
)

var (
	_debug    = flag.Bool("debug", false, "enable debug log")
	_database = flag.String("db", "", "sqlite database file, default to $HOME/pocket")
)

var (
	_debugLogFile *os.File
	_debugLogger  *log.Logger
	_debugLogPipe chan func(logger *log.Logger)
)

func main() {
	flag.Parse()

	if *_debug {
		_debugLogFile, _ = os.Create("debug.log")
		defer _debugLogFile.Close()
		log.SetOutput(_debugLogFile)
		_debugLogger = log.Default()
		_debugLogPipe = make(chan func(*log.Logger), 50)
		go func() {
			for f := range _debugLogPipe {
				if *_debug {
					f(_debugLogger)
				}
			}
		}()
	}

	if *_database == "" {
		if v, ok := os.LookupEnv(EnvSqliteFile); ok {
			v = strings.TrimSpace(v)
			if v != "" {
				*_database = v
			}
		}
	}

	if *_database == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			panic(err)
		}
		sp := string(os.PathSeparator)
		pdir := home + sp + "pocket"
		err = os.MkdirAll(pdir, os.ModePerm)
		if err != nil {
			panic(err)
		}
		*_database = pdir + sp + "pocket.db"
	}

	if err := OpenDB(*_database, *_debug, _debugLogFile); err != nil {
		panic(err)
	}

	if err := NewApp().Run(); err != nil {
		panic(err)
	}
}

func Debugf(fmt string, args ...any) {
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
