package main

import (
	"fmt"
)

const (
	SchemaVersion = "v0.0.0"
	CKeyPwTest    = "PasswordTest"
	PwTestLen     = 13
)

// Storage API Contract
var (
	StDeleteNote    func(note Note) error                                       = DeleteNote
	StEditNote      func(note Note) error                                       = UpdateNote
	StCreateNote    func(note Note) (Note, error)                               = CreateNote
	StFetchNotes    func(page int, limit int, name string) (int, []Note, error) = FetchNotes
	StCheckPassword func() (bool, error)                                        = CheckPassword
	StInitSchema    func() error                                                = InitSchema
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

func CheckPassword() (bool, error) {
	var n string
	err := GetDB().Raw(`SELECT name FROM sqlite_master WHERE type = 'table' AND name = 'pocket_config'`).Scan(&n).Error
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
		Debugf("Check password failed, %v", err)
		return false, nil
	}
	return isValidPwCheckVal(val), nil
}

func isValidPwCheckVal(s string) bool {
	if s == CKeyPwTest {
		return true
	}
	rr := []rune(s)
	if len(rr) != PwTestLen {
		return false
	}
	for _, r := range rr {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
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

	val, err := Encrypt(doRand(PwTestLen, digits))
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

func FetchNotes(page int, limit int, kw string) (int, []Note, error) {
	t := GetDB().Table("pocket_note").
		Select(`count(*)`)
	if kw != "" {
		t = t.Where("name MATCH ? OR desc MATCH ?", kw, kw)
	}
	var total int
	if err := t.Scan(&total).Error; err != nil {
		return 0, nil, fmt.Errorf("failed to query notes, %v", err)
	}
	if total < 1 {
		return total, []Note{}, nil
	}

	t = GetDB().Table("pocket_note").
		Select("rowid id, name, desc, content, ctime, utime").
		Order("id DESC").
		Limit(limit).
		Offset((page - 1) * limit)

	if kw != "" {
		t = t.Where("name MATCH ? OR desc MATCH ?", kw, kw)
	}

	var notes []Note
	if err := t.Scan(&notes).Error; err != nil {
		return 0, nil, fmt.Errorf("failed to query notes, %v", err)
	}
	if notes == nil {
		notes = make([]Note, 0)
	}
	// Debugf("fetched notes: %#v", notes)
	for i := range notes {
		notes[i] = DecryptNote(notes[i])
	}
	return total, notes, nil
}

func CreateNote(n Note) (Note, error) {
	en := EncryptNote(n)

	err := GetDB().Exec(`
	INSERT INTO pocket_note (name, desc, content, ctime, utime)
	VALUES (?,?,?,?,?)
	`, en.Name, en.Desc, en.Content, en.Ctime, en.Utime).Error

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
