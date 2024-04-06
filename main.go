package main

import (
	"flag"
	"log"
	"os"
	"strings"
)

var (
	debug    = flag.Bool("debug", false, "enable debug log")
	database = flag.String("db", "", "sqlite database file, default to $HOME/pocket")
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

	if *database == "" {
		if v, ok := os.LookupEnv(EnvSqliteFile); ok {
			v = strings.TrimSpace(v)
			if v != "" {
				*database = v
			}
		}
	}

	if *database == "" {
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
		*database = pdir + sp + "pocket.db"
	}

	if err := OpenDB(*database); err != nil {
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
