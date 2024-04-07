package main

import (
	"fmt"
	"io"
	"log"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

const (
	EnvSqliteFile = "POCKET_DB"
)

var (
	sqliteDb *gorm.DB = nil
)

func GetDB() *gorm.DB {
	return sqliteDb
}

func NewGormDebugLogger(debugOut io.Writer) logger.Interface {
	return logger.New(log.New(debugOut, "\r\n", log.LstdFlags), logger.Config{
		SlowThreshold:             200 * time.Millisecond,
		LogLevel:                  logger.Info,
		IgnoreRecordNotFoundError: false,
		Colorful:                  false,
	})
}

func OpenDB(file string, debug bool, debugOut io.Writer) error {
	sq, err := newSqlite(file)
	if err != nil {
		return fmt.Errorf("failed to open SQLite file, file: %v, %v", file, err)
	}
	if debug {
		sq.Logger = NewGormDebugLogger(debugOut)
	}
	sqliteDb = sq

	// https://www.sqlite.org/pragma.html#pragma_journal_mode
	Debugf("Enabling SQLite WAL mode")
	var mode string
	t := sq.Raw("PRAGMA journal_mode=WAL").Scan(&mode)
	if err := t.Error; err != nil {
		panic(fmt.Errorf("failed to enable WAL mode, %v", err))
	} else {
		Debugf("Enabled SQLite WAL mode, result: %v", mode)
	}
	return nil
}

func newSqlite(path string) (*gorm.DB, error) {
	Debugf("Connecting to SQLite database '%s'", path)

	db, err := gorm.Open(sqlite.Open(path), &gorm.Config{})
	if err != nil {
		return nil, fmt.Errorf("failed to open SQLite, %v", err)
	}

	tx, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to connect SQLite, %v", err)
	}

	// make sure the handle is actually connected
	err = tx.Ping()
	if err != nil {
		return nil, fmt.Errorf("failed to ping SQLite, %v", err)
	}

	Debugf("SQLite connected")
	return db, nil
}
