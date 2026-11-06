package main

import (
	"bytes"
	"errors"
	"image"
	"image/color"
	"image/png"
	"testing"

	qrcode "github.com/skip2/go-qrcode"
)

func qrPng(t *testing.T, text string) []byte {
	t.Helper()
	pngBytes, err := qrcode.Encode(text, qrcode.Medium, 256)
	if err != nil {
		t.Fatal(err)
	}
	return pngBytes
}

func plainPng(t *testing.T) []byte {
	t.Helper()
	img := image.NewGray(image.Rect(0, 0, 64, 64))
	for y := 0; y < 64; y++ {
		for x := 0; x < 64; x++ {
			img.SetGray(x, y, color.Gray{Y: 255})
		}
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func TestDecodeQrPngRoundTrip(t *testing.T) {
	const uri = "otpauth://totp/GitHub:alice%40example.com?secret=JBSWY3DPEHPK3PXP"

	got, err := DecodeQrPng(qrPng(t, uri))
	if err != nil {
		t.Fatal(err)
	}
	if got != uri {
		t.Fatalf("decoded text = %q, want %q", got, uri)
	}
}

func TestDecodeQrPngWithoutQrCode(t *testing.T) {
	_, err := DecodeQrPng(plainPng(t))
	if !errors.Is(err, ErrQrNotFound) {
		t.Fatalf("err = %v, want %v", err, ErrQrNotFound)
	}
}
