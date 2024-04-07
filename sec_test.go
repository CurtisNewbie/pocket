package main

import "testing"

func TestEncrypt(t *testing.T) {
	InitPassword("mypassword")
	dat := "mydata"
	enc, err := Encrypt(dat)
	if err != nil {
		t.Fatal(err)
	}
	t.Log(enc)
	dec, err := Decrypt(enc)
	if err != nil {
		t.Fatal(err)
	}
	t.Log(dec)
	if dat != dec {
		t.Fatal("result not match")
	}
}
