package main

import (
	"encoding/hex"
	"errors"
	"os/exec"
	"strings"
)

const (
	ClipboardKindText  = "text"
	ClipboardKindImage = "image"
)

// ClipboardContent is what the pasteboard currently holds. Png is empty for
// text content, and Text is empty for image content.
type ClipboardContent struct {
	Kind string
	Text string
	Png  []byte
}

var (
	ErrClipboardEmpty              = errors.New("clipboard-empty")
	ErrClipboardUnsupportedContent = errors.New("clipboard-unsupported-content")
	ErrClipboardReadFailed         = errors.New("clipboard-read-failed")
	ErrClipboardWriteFailed        = errors.New("clipboard-write-failed")
	ErrQrNotTotpUri                = errors.New("qr-not-totp-uri")
)

// StWriteClipboard is the substitutable clipboard writer.
var StWriteClipboard func(text string) error = WriteClipboard

// WriteClipboard copies text through the macOS pasteboard.
func WriteClipboard(text string) error {
	cmd := exec.Command("pbcopy")
	cmd.Stdin = strings.NewReader(text)
	if err := cmd.Run(); err != nil {
		return ErrClipboardWriteFailed
	}
	return nil
}

// StCaptureClipboard is the substitutable clipboard reader. The UI reads the
// pasteboard only through this function, so the import flow can be tested
// without touching the OS pasteboard.
var StCaptureClipboard func() (ClipboardContent, error) = CaptureClipboard

// CaptureClipboard reads the pasteboard. An image wins when the clipboard
// carries both an image and text.
func CaptureClipboard() (ClipboardContent, error) {
	if pngBytes, ok := readClipboardPng(); ok {
		return ClipboardContent{Kind: ClipboardKindImage, Png: pngBytes}, nil
	}
	out, err := exec.Command("pbpaste").Output()
	if err != nil {
		return ClipboardContent{}, ErrClipboardReadFailed
	}
	text := string(out)
	if strings.TrimSpace(text) == "" {
		return ClipboardContent{}, ErrClipboardEmpty
	}
	return ClipboardContent{Kind: ClipboardKindText, Text: text}, nil
}

// readClipboardPng asks the pasteboard for its image rendered as PNG. The
// osascript result carries the bytes as hex, so they are decoded in memory and
// the image is never written to disk.
func readClipboardPng() ([]byte, bool) {
	out, err := exec.Command("osascript", "-e", "the clipboard as «class PNGf»").Output()
	if err != nil {
		return nil, false
	}
	return decodeAppleScriptData(out)
}

func decodeAppleScriptData(out []byte) ([]byte, bool) {
	const prefix = "«data PNGf"
	s := strings.TrimSpace(string(out))
	if !strings.HasPrefix(s, prefix) || !strings.HasSuffix(s, "»") {
		return nil, false
	}
	pngBytes, err := hex.DecodeString(s[len(prefix) : len(s)-len("»")])
	if err != nil || len(pngBytes) == 0 {
		return nil, false
	}
	return pngBytes, true
}

// ReadTotpFromClipboard reads the clipboard and parses it as a TOTP entry. An
// image is decoded as a QR code; text is parsed as an otpauth URI or a bare
// Base32 secret.
func ReadTotpFromClipboard() (ParsedTotp, error) {
	content, err := StCaptureClipboard()
	if err != nil {
		return ParsedTotp{}, err
	}
	switch content.Kind {
	case ClipboardKindImage:
		text, err := DecodeQrPng(content.Png)
		if err != nil {
			return ParsedTotp{}, err
		}
		parsed, err := ParseTotpText(text)
		if err != nil {
			if errors.Is(err, ErrHotpUnsupported) || errors.Is(err, ErrSecretInvalid) || errors.Is(err, ErrParamInvalid) {
				return ParsedTotp{}, err
			}
			return ParsedTotp{}, ErrQrNotTotpUri
		}
		return parsed, nil
	case ClipboardKindText:
		if strings.TrimSpace(content.Text) == "" {
			return ParsedTotp{}, ErrClipboardEmpty
		}
		return ParseTotpText(content.Text)
	default:
		return ParsedTotp{}, ErrClipboardUnsupportedContent
	}
}
