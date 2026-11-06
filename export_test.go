package main

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestBuildTotpUriRoundTrip(t *testing.T) {
	cases := []struct {
		name  string
		entry TotpEntry
	}{
		{
			name: "custom parameters",
			entry: TotpEntry{
				Issuer:    "Acme",
				Account:   "alice@example.com",
				Secret:    "MFRGGZDFMZTWQ2LK",
				Digits:    8,
				Period:    60,
				Algorithm: "SHA256",
			},
		},
		{
			name: "sha512 with default digits and period",
			entry: TotpEntry{
				Issuer:    "GitHub",
				Account:   "alice",
				Secret:    "JBSWY3DPEHPK3PXP",
				Digits:    6,
				Period:    30,
				Algorithm: "SHA512",
			},
		},
		{
			name: "empty issuer",
			entry: TotpEntry{
				Account:   "alice@example.com",
				Secret:    "JBSWY3DPEHPK3PXP",
				Digits:    6,
				Period:    30,
				Algorithm: "SHA1",
			},
		},
		{
			name: "empty account",
			entry: TotpEntry{
				Issuer:    "Acme",
				Secret:    "JBSWY3DPEHPK3PXP",
				Digits:    6,
				Period:    30,
				Algorithm: "SHA1",
			},
		},
		{
			name: "issuer containing a colon",
			entry: TotpEntry{
				Issuer:    "Acme:Ops",
				Account:   "alice@example.com",
				Secret:    "JBSWY3DPEHPK3PXP",
				Digits:    6,
				Period:    30,
				Algorithm: "SHA1",
			},
		},
		{
			name: "account containing a colon",
			entry: TotpEntry{
				Issuer:    "Acme",
				Account:   "alice:work",
				Secret:    "JBSWY3DPEHPK3PXP",
				Digits:    6,
				Period:    30,
				Algorithm: "SHA1",
			},
		},
		{
			name: "characters that need percent-encoding",
			entry: TotpEntry{
				Issuer:    "Acme Inc./Ops",
				Account:   "alice smith&bob@example.com",
				Secret:    "JBSWY3DPEHPK3PXP",
				Digits:    6,
				Period:    30,
				Algorithm: "SHA1",
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			uri, err := BuildTotpUri(tc.entry)
			if err != nil {
				t.Fatal(err)
			}
			parsed, err := ParseTotpText(uri)
			if err != nil {
				t.Fatalf("built uri %q does not parse: %v", uri, err)
			}
			if parsed.Bare {
				t.Fatal("built uri parsed as a bare secret")
			}
			if parsed.Issuer != tc.entry.Issuer {
				t.Fatalf("issuer = %q, want %q (uri %q)", parsed.Issuer, tc.entry.Issuer, uri)
			}
			if parsed.Account != tc.entry.Account {
				t.Fatalf("account = %q, want %q (uri %q)", parsed.Account, tc.entry.Account, uri)
			}
			if parsed.Secret != tc.entry.Secret {
				t.Fatalf("secret = %q, want %q (uri %q)", parsed.Secret, tc.entry.Secret, uri)
			}
			if parsed.Digits != tc.entry.Digits {
				t.Fatalf("digits = %d, want %d (uri %q)", parsed.Digits, tc.entry.Digits, uri)
			}
			if parsed.Period != tc.entry.Period {
				t.Fatalf("period = %d, want %d (uri %q)", parsed.Period, tc.entry.Period, uri)
			}
			if parsed.Algorithm != tc.entry.Algorithm {
				t.Fatalf("algorithm = %q, want %q (uri %q)", parsed.Algorithm, tc.entry.Algorithm, uri)
			}
		})
	}
}

func TestBuildTotpUriUndecryptableSecret(t *testing.T) {
	_, err := BuildTotpUri(TotpEntry{Issuer: "Acme", Account: "alice", Digits: 6, Period: 30, Algorithm: "SHA1"})
	if !errors.Is(err, ErrSecretUndecryptable) {
		t.Fatalf("err = %v, want %v", err, ErrSecretUndecryptable)
	}
}

func TestExportFileName(t *testing.T) {
	cases := []struct {
		name  string
		entry TotpEntry
		want  string
	}{
		{
			name:  "issuer and account",
			entry: TotpEntry{Issuer: "GitHub", Account: "alice@example.com"},
			want:  "GitHub-alice@example.com.png",
		},
		{
			name:  "slash space and colon become a dash",
			entry: TotpEntry{Issuer: "Acme Inc./Ops", Account: "alice:work smith"},
			want:  "Acme-Inc.-Ops-alice-work-smith.png",
		},
		{
			name:  "empty issuer",
			entry: TotpEntry{Account: "alice@example.com"},
			want:  "alice@example.com.png",
		},
		{
			name:  "empty account",
			entry: TotpEntry{Issuer: "Acme"},
			want:  "Acme.png",
		},
		{
			name:  "both empty",
			entry: TotpEntry{},
			want:  "totp.png",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := exportFileName(tc.entry); got != tc.want {
				t.Fatalf("exportFileName() = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestEncodeQrPngRoundTrip(t *testing.T) {
	entry := TotpEntry{
		Issuer:    "Acme Inc./Ops",
		Account:   "alice@example.com",
		Secret:    "MFRGGZDFMZTWQ2LK",
		Digits:    8,
		Period:    60,
		Algorithm: "SHA256",
	}
	uri, err := BuildTotpUri(entry)
	if err != nil {
		t.Fatal(err)
	}

	path := filepath.Join(t.TempDir(), "export.png")
	if err := EncodeQrPng(uri, path); err != nil {
		t.Fatal(err)
	}

	pngBytes, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	decoded, err := DecodeQrPng(pngBytes)
	if err != nil {
		t.Fatal(err)
	}
	if decoded != uri {
		t.Fatalf("decoded text = %q, want %q", decoded, uri)
	}
}

func TestEncodeQrPngRefusesExistingFile(t *testing.T) {
	const existing = "do not touch"
	path := filepath.Join(t.TempDir(), "export.png")
	if err := os.WriteFile(path, []byte(existing), 0o600); err != nil {
		t.Fatal(err)
	}

	err := EncodeQrPng("otpauth://totp/Acme:alice?secret=JBSWY3DPEHPK3PXP", path)
	if !errors.Is(err, ErrFileExists) {
		t.Fatalf("err = %v, want %v", err, ErrFileExists)
	}

	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != existing {
		t.Fatalf("existing file was modified: %q", got)
	}
}

func TestEncodeQrPngUnwritableFolder(t *testing.T) {
	path := filepath.Join(t.TempDir(), "missing", "export.png")
	err := EncodeQrPng("otpauth://totp/Acme:alice?secret=JBSWY3DPEHPK3PXP", path)
	if !errors.Is(err, ErrFileWriteFailed) {
		t.Fatalf("err = %v, want %v", err, ErrFileWriteFailed)
	}
}
