package main

import (
	"fmt"
)

const (
	SchemaVersion = "v0.0.0"
	CKeyPwTest    = "PasswordTest"
)

// Storage API Contract
var (
	StDeleteNote    func(note Note) error                                  = DeleteNote
	StEditNote      func(note Note) error                                  = UpdateNote
	StCreateNote    func(note Note) (Note, error)                          = CreateNote
	StFetchNotes    func(page int, limit int, name string) ([]Note, error) = FetchNotes
	StCheckPassword func(pw string) (bool, error)                          = CheckPassword
	StInitSchema    func() error                                           = InitSchema
)

var (
	_initSchemaFlag = false
)

type Note struct {
	Id      int
	Name    string
	Desc    string
	Content string
	Ctime   ETime
	Utime   ETime
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
			config_key TEXT NOT NULL,
			config_value TEXT NOT NULL
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
	if err != nil {
		return fmt.Errorf("failed to init pocket_config record, %v", err)
	}

	err = GetDB().Exec(`
		CREATE VIRTUAL TABLE IF NOT EXISTS pocket_note USING fts4 (
			name TEXT NOT NULL,
			desc TEXT NOT NULL,
			content TEXT NOT NULL,
			ctime DATETIME NOT NULL,
			utime DATETIME NOT NULL
		)
	`).Error
	if err != nil {
		return fmt.Errorf("failed to initialize schema, %v", err)
	}

	return nil
}

func FetchNotes(page int, limit int, name string) ([]Note, error) {
	t := GetDB().Table("pocket_note").
		Select("rowid id, name, desc, content, ctime, utime").
		Order("id DESC").
		Limit(limit).
		Offset((page - 1) * limit)

	if name != "" {
		t = t.Where("name MATCH ?", name)
	}

	var notes []Note
	if err := t.Scan(&notes).Error; err != nil {
		return nil, fmt.Errorf("failed to query notes, %v", err)
	}
	if notes == nil {
		notes = make([]Note, 0)
	}
	Debugf("fetched notes: %#v", notes)
	for i := range notes {
		notes[i] = DecryptNote(notes[i])
	}
	return notes, nil
}

func CreateNote(n Note) (Note, error) {
	n = EncryptNote(n)

	err := GetDB().Exec(`
	INSERT INTO pocket_note (name, desc, content, ctime, utime)
	VALUES (?,?,?,?,?)
	`, n.Name, n.Desc, n.Content, n.Ctime, n.Utime).Error

	if err != nil {
		return Note{}, fmt.Errorf("failed to save note, %v", err)
	}

	var id int
	err = GetDB().Raw(`SELECT last_insert_rowid()`).Scan(&id).Error
	if err != nil {
		return Note{}, fmt.Errorf("failed to find id of newly saved note, %v", err)
	}

	n.Id = id
	return n, nil
}

func EncryptNote(n Note) Note {
	n.Content = Encrypt0(n.Content)
	return n
}

func DecryptNote(n Note) Note {
	n.Content = Decrypt0(n.Content)
	return n
}

func UpdateNote(n Note) error {
	n = EncryptNote(n)
	err := GetDB().Exec(`
	UPDATE pocket_note
	SET name = ?, desc = ?, content = ?, utime = ?
	WHERE rowid = ?
	`, n.Name, n.Desc, n.Content, n.Utime, n.Id).Error
	if err != nil {
		return fmt.Errorf("failed to update pocket_note, %v", err)
	}
	return nil
}

func DeleteNote(note Note) error {
	err := GetDB().Exec(`DELETE FROM pocket_note WHERE rowid = ?`, note.Id).Error
	if err != nil {
		return fmt.Errorf("failed to delete pocket_note, %v", err)
	}
	return nil
}
