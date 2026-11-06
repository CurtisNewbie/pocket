package main

import (
	"bytes"
	"errors"
	"image/png"

	"github.com/makiuchi-d/gozxing"
	"github.com/makiuchi-d/gozxing/qrcode"
)

var (
	ErrQrNotFound     = errors.New("qr-not-found")
	ErrQrDecodeFailed = errors.New("qr-decode-failed")
)

// DecodeQrPng returns the text embedded in a QR code PNG.
func DecodeQrPng(pngBytes []byte) (string, error) {
	img, err := png.Decode(bytes.NewReader(pngBytes))
	if err != nil {
		return "", ErrQrDecodeFailed
	}
	bitmap, err := gozxing.NewBinaryBitmapFromImage(img)
	if err != nil {
		return "", ErrQrDecodeFailed
	}
	result, err := qrcode.NewQRCodeReader().Decode(bitmap, nil)
	if err != nil {
		var notFound gozxing.NotFoundException
		if errors.As(err, &notFound) {
			return "", ErrQrNotFound
		}
		return "", ErrQrDecodeFailed
	}
	return result.GetText(), nil
}
