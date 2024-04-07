package main

import (
	"fmt"
	"math/rand"
)

const (
	SchemaVersion = "v0.0.0"
	CKeyPwTest    = "PasswordTest"
)

// Storage API Contract
var (
	StDeleteNote    func(it NoteItem) error
	StEditNote      func(note NoteItem) error
	StCreateNote    func(note NoteItem) error
	StFetchNotes    func(page int, name string) ([]NoteItem, error)
	StCheckPassword func(pw string) (bool, error) = CheckPassword
	StInitSchema    func() error                  = InitSchema
)

var (
	_initSchemaFlag = false
)

func init() {
	// TODO: for demo only
	StDeleteNote = func(it NoteItem) error {
		Debugf("Delete note %#v", it)
		return nil
	}
	StEditNote = func(note NoteItem) error {
		Debugf("Edit note %#v", note)
		return nil
	}
	StCreateNote = func(note NoteItem) error {
		Debugf("Create note %#v", note)
		return nil
	}
	StFetchNotes = func(page int, name string) ([]NoteItem, error) {
		Debugf("Fetch page, %v, name: %v", page, name)
		return []NoteItem{
			{
				id:   rand.Intn(10),
				name: "yo",
				desc: "yo it's me",
			}, {
				id:   rand.Intn(10),
				name: "yo",
				desc: "yo it's me",
			},
			{
				id:   rand.Intn(10),
				name: "yo",
				desc: "yo it's me",
			},
			{
				id:   rand.Intn(10),
				name: "yo",
				desc: "yo it's me",
			},
			{
				id:   rand.Intn(10),
				name: "yo",
				desc: "yo it's me",
			},
		}, nil
	}
	// StCheckPassword = func(pw string) bool {
	// 	Debugf("Checking passward")
	// 	return true
	// }
}

func CheckPassword(pw string) (bool, error) {
	var n string
	err := GetDB().Raw(`SELECT name FROM sqlite_master WHERE type ='table' AND name = 'pocket_config'`).Scan(&n).Error
	if err != nil {
		return false, fmt.Errorf("failed to query database, %v", err)
	}
	firstTime := n == ""
	if firstTime {
		_initSchemaFlag = true
		return true, nil
	}

	var val string
	err = GetDB().Raw(`SELECT config_value FROM pocket_config WHERE config_key = ?`, CKeyPwTest).
		Scan(&val).Error
	if err != nil {
		return false, fmt.Errorf("failed to query pocket_config, database may be corrupted, %v", err)
	}
	val, err = Decrypt(val)
	if err != nil {
		return false, err
	}
	return val == CKeyPwTest, nil
}

func InitSchema() error {
	if !_initSchemaFlag {
		return nil
	}

	err := GetDB().Exec(`
		CREATE TABLE IF NOT EXISTS pocket_config (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			config_key VARCHAR(30) NOT NULL,
			config_value VARCHAR(255) NOT NULL
		)
	`).Error
	if err != nil {
		return fmt.Errorf("failed to initialize schema, %v", err)
	}

	err = GetDB().Exec(`
		CREATE INDEX IF NOT EXISTS key_idx ON pocket_config (config_key)
	`).Error
	if err != nil {
		return fmt.Errorf("failed to initialize schema, %v", err)
	}

	val, err := Encrypt(CKeyPwTest)
	if err != nil {
		return err
	}

	err = GetDB().Exec(`INSERT INTO pocket_config (config_key, config_value) VALUES (?,?)`, CKeyPwTest, val).Error
	return err
}
