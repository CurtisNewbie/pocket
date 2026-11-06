package main

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	qrcode "github.com/skip2/go-qrcode"
)

const (
	exportDirName  = "pocket-export"
	exportSize     = 256
	exportFileMode = 0o644
	exportDirMode  = 0o755
)

var (
	ErrFileExists      = errors.New("file-exists")
	ErrFileWriteFailed = errors.New("file-write-failed")
	ErrQrEncodeFailed  = errors.New("encode-failed")
)

// EncodeQrPng writes text as a QR PNG to path. The file is created exclusively,
// so an existing file reports file-exists and is never overwritten.
func EncodeQrPng(text string, path string) error {
	pngBytes, err := qrcode.Encode(text, qrcode.Medium, exportSize)
	if err != nil {
		return ErrQrEncodeFailed
	}

	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, exportFileMode)
	if err != nil {
		if errors.Is(err, fs.ErrExist) {
			return ErrFileExists
		}
		return ErrFileWriteFailed
	}
	if _, err := f.Write(pngBytes); err != nil {
		f.Close()
		os.Remove(path)
		return ErrFileWriteFailed
	}
	if err := f.Close(); err != nil {
		os.Remove(path)
		return ErrFileWriteFailed
	}
	return nil
}

// defaultExportPath returns the export directory under the user's home joined
// with a filename built from the entry. The directory is created when absent.
func defaultExportPath(e TotpEntry) (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	dir := filepath.Join(home, exportDirName)
	if err := os.MkdirAll(dir, exportDirMode); err != nil {
		return "", err
	}
	return filepath.Join(dir, exportFileName(e)), nil
}

// exportFileName names an export after the entry's issuer and account, with
// "/", a space, and ":" replaced by "-" so the name is safe to join to a path.
func exportFileName(e TotpEntry) string {
	parts := make([]string, 0, 2)
	if e.Issuer != "" {
		parts = append(parts, e.Issuer)
	}
	if e.Account != "" {
		parts = append(parts, e.Account)
	}
	name := strings.Join(parts, " ")
	name = strings.NewReplacer("/", "-", " ", "-", ":", "-").Replace(name)
	name = strings.Trim(name, "-")
	if name == "" {
		name = "totp"
	}
	return name + ".png"
}
