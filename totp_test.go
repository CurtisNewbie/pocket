package main

import (
	"errors"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func openTestDB(t *testing.T) {
	t.Helper()
	InitPassword("mypassword")
	_initSchemaFlag = false
	if err := OpenDB(filepath.Join(t.TempDir(), "pocket.db"), false, nil); err != nil {
		t.Fatal(err)
	}
	sqlDB, err := GetDB().DB()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { sqlDB.Close() })
}

func hasTable(t *testing.T, name string) bool {
	t.Helper()
	var n string
	if err := GetDB().Raw(`SELECT name FROM sqlite_master WHERE type = 'table' AND name = ?`, name).Scan(&n).Error; err != nil {
		t.Fatal(err)
	}
	return n == name
}

func testTotpEntry(issuer, secret string, ctime ETime) TotpEntry {
	return TotpEntry{
		Issuer:    issuer,
		Account:   "alice@example.com",
		Secret:    secret,
		Digits:    6,
		Period:    30,
		Algorithm: "SHA1",
		Ctime:     ctime,
		Utime:     ctime,
	}
}

func TestParseTotpTextDefaults(t *testing.T) {
	got, err := ParseTotpText("otpauth://totp/GitHub:alice%40example.com?secret=JBSWY3DPEHPK3PXP")
	if err != nil {
		t.Fatal(err)
	}
	if got.Bare {
		t.Fatal("otpauth URI parsed as a bare secret")
	}
	if got.Issuer != "GitHub" {
		t.Fatalf("issuer = %q, want %q", got.Issuer, "GitHub")
	}
	if got.Account != "alice@example.com" {
		t.Fatalf("account = %q, want %q", got.Account, "alice@example.com")
	}
	if got.Secret != "JBSWY3DPEHPK3PXP" {
		t.Fatalf("secret = %q, want %q", got.Secret, "JBSWY3DPEHPK3PXP")
	}
	if got.Digits != 6 {
		t.Fatalf("digits = %d, want 6", got.Digits)
	}
	if got.Period != 30 {
		t.Fatalf("period = %d, want 30", got.Period)
	}
	if got.Algorithm != "SHA1" {
		t.Fatalf("algorithm = %q, want %q", got.Algorithm, "SHA1")
	}
}

func TestParseTotpTextNonDefaultParams(t *testing.T) {
	got, err := ParseTotpText("otpauth://totp/alice?secret=MFRGGZDFMZTWQ2LK&issuer=Acme&algorithm=SHA256&digits=8&period=60")
	if err != nil {
		t.Fatal(err)
	}
	if got.Issuer != "Acme" {
		t.Fatalf("issuer = %q, want %q", got.Issuer, "Acme")
	}
	if got.Account != "alice" {
		t.Fatalf("account = %q, want %q", got.Account, "alice")
	}
	if got.Digits != 8 {
		t.Fatalf("digits = %d, want 8", got.Digits)
	}
	if got.Period != 60 {
		t.Fatalf("period = %d, want 60", got.Period)
	}
	if got.Algorithm != "SHA256" {
		t.Fatalf("algorithm = %q, want %q", got.Algorithm, "SHA256")
	}
}

func TestParseTotpTextBareSecret(t *testing.T) {
	got, err := ParseTotpText("jbswy3dpehpk3pxp")
	if err != nil {
		t.Fatal(err)
	}
	if !got.Bare {
		t.Fatal("bare secret parsed as an otpauth URI")
	}
	if got.Issuer != "" || got.Account != "" {
		t.Fatalf("issuer/account = %q/%q, want both empty", got.Issuer, got.Account)
	}
	if got.Secret != "JBSWY3DPEHPK3PXP" {
		t.Fatalf("secret = %q, want %q", got.Secret, "JBSWY3DPEHPK3PXP")
	}
	if got.Digits != 6 || got.Period != 30 || got.Algorithm != "SHA1" {
		t.Fatalf("defaults = %d/%d/%q, want 6/30/SHA1", got.Digits, got.Period, got.Algorithm)
	}
}

func TestParseTotpTextRejectsHotp(t *testing.T) {
	_, err := ParseTotpText("otpauth://hotp/GitHub:alice?secret=JBSWY3DPEHPK3PXP&counter=0")
	if !errors.Is(err, ErrHotpUnsupported) {
		t.Fatalf("err = %v, want %v", err, ErrHotpUnsupported)
	}
}

func TestParseTotpTextRejectsInvalidSecret(t *testing.T) {
	cases := []string{
		"otpauth://totp/GitHub:alice?secret=INVALID1",
		"JBSWY3DPEHPK3P1",
	}
	for _, text := range cases {
		_, err := ParseTotpText(text)
		if !errors.Is(err, ErrSecretInvalid) {
			t.Fatalf("ParseTotpText(%q) err = %v, want %v", text, err, ErrSecretInvalid)
		}
	}
}

func TestParseTotpTextRejectsNonOtpauthText(t *testing.T) {
	cases := []string{
		"this is not a secret",
		"https://example.com",
		"otpauth://totp/GitHub:alice",
	}
	for _, text := range cases {
		_, err := ParseTotpText(text)
		if !errors.Is(err, ErrUriInvalid) {
			t.Fatalf("ParseTotpText(%q) err = %v, want %v", text, err, ErrUriInvalid)
		}
	}
}

func TestParseTotpTextRejectsInvalidParams(t *testing.T) {
	cases := []struct {
		uri  string
		want string
	}{
		{"otpauth://totp/GitHub:alice?secret=JBSWY3DPEHPK3PXP&digits=7", "digits must be 6 or 8"},
		{"otpauth://totp/GitHub:alice?secret=JBSWY3DPEHPK3PXP&period=0", "period must be 1 to 3600 seconds"},
		{"otpauth://totp/GitHub:alice?secret=JBSWY3DPEHPK3PXP&period=3601", "period must be 1 to 3600 seconds"},
		{"otpauth://totp/GitHub:alice?secret=JBSWY3DPEHPK3PXP&algorithm=MD5", "algorithm must be SHA1, SHA256, or SHA512"},
	}
	for _, c := range cases {
		_, err := ParseTotpText(c.uri)
		if !errors.Is(err, ErrParamInvalid) {
			t.Fatalf("ParseTotpText(%q) err = %v, want %v", c.uri, err, ErrParamInvalid)
		}
		if !strings.Contains(err.Error(), c.want) {
			t.Fatalf("ParseTotpText(%q) err = %v, want it to mention %q", c.uri, err, c.want)
		}
	}
}

func TestReadTotpFromClipboardText(t *testing.T) {
	orig := StCaptureClipboard
	t.Cleanup(func() { StCaptureClipboard = orig })
	StCaptureClipboard = func() (ClipboardContent, error) {
		return ClipboardContent{
			Kind: ClipboardKindText,
			Text: "otpauth://totp/GitHub:alice?secret=JBSWY3DPEHPK3PXP",
		}, nil
	}

	got, err := ReadTotpFromClipboard()
	if err != nil {
		t.Fatal(err)
	}
	if got.Issuer != "GitHub" || got.Account != "alice" {
		t.Fatalf("issuer/account = %q/%q, want GitHub/alice", got.Issuer, got.Account)
	}
	if got.Secret != "JBSWY3DPEHPK3PXP" {
		t.Fatalf("secret = %q, want %q", got.Secret, "JBSWY3DPEHPK3PXP")
	}
}

func TestReadTotpFromClipboardImage(t *testing.T) {
	orig := StCaptureClipboard
	t.Cleanup(func() { StCaptureClipboard = orig })
	StCaptureClipboard = func() (ClipboardContent, error) {
		return ClipboardContent{
			Kind: ClipboardKindImage,
			Png:  qrPng(t, "otpauth://totp/GitHub:alice?secret=JBSWY3DPEHPK3PXP"),
		}, nil
	}

	got, err := ReadTotpFromClipboard()
	if err != nil {
		t.Fatal(err)
	}
	if got.Issuer != "GitHub" || got.Account != "alice" {
		t.Fatalf("issuer/account = %q/%q, want GitHub/alice", got.Issuer, got.Account)
	}
	if got.Secret != "JBSWY3DPEHPK3PXP" {
		t.Fatalf("secret = %q, want %q", got.Secret, "JBSWY3DPEHPK3PXP")
	}
}

func TestReadTotpFromClipboardImageFailures(t *testing.T) {
	orig := StCaptureClipboard
	t.Cleanup(func() { StCaptureClipboard = orig })

	cases := []struct {
		name    string
		png     []byte
		want    error
		wantMsg string
	}{
		{"no qr code", plainPng(t), ErrQrNotFound, "no qr code in clipboard image"},
		{"not a totp uri", qrPng(t, "just some text"), ErrQrNotTotpUri, "qr code isn't a totp uri"},
		{"hotp uri", qrPng(t, "otpauth://hotp/GitHub:alice?secret=JBSWY3DPEHPK3PXP&counter=0"), ErrHotpUnsupported, "hotp isn't supported, totp only"},
		{"invalid secret", qrPng(t, "otpauth://totp/GitHub:alice?secret=INVALID1"), ErrSecretInvalid, "secret isn't valid base32"},
		{"invalid digits", qrPng(t, "otpauth://totp/GitHub:alice?secret=JBSWY3DPEHPK3PXP&digits=7"), ErrParamInvalid, "digits must be 6 or 8"},
	}
	for _, c := range cases {
		StCaptureClipboard = func() (ClipboardContent, error) {
			return ClipboardContent{Kind: ClipboardKindImage, Png: c.png}, nil
		}
		_, err := ReadTotpFromClipboard()
		if !errors.Is(err, c.want) {
			t.Fatalf("%s: err = %v, want %v", c.name, err, c.want)
		}
		if got := importErrorMessage(err); got != c.wantMsg {
			t.Fatalf("%s: message = %q, want %q", c.name, got, c.wantMsg)
		}
	}
}

func TestReadTotpFromClipboardFailures(t *testing.T) {
	orig := StCaptureClipboard
	t.Cleanup(func() { StCaptureClipboard = orig })

	cases := []struct {
		name    string
		content ClipboardContent
		err     error
		want    error
	}{
		{"empty clipboard", ClipboardContent{}, ErrClipboardEmpty, ErrClipboardEmpty},
		{"unsupported kind", ClipboardContent{Kind: "file"}, nil, ErrClipboardUnsupportedContent},
		{"blank text", ClipboardContent{Kind: ClipboardKindText, Text: "  \n"}, nil, ErrClipboardEmpty},
		{"neither", ClipboardContent{Kind: ClipboardKindText, Text: "not an otpauth uri"}, nil, ErrUriInvalid},
		{"read failed", ClipboardContent{}, ErrClipboardReadFailed, ErrClipboardReadFailed},
	}
	for _, c := range cases {
		StCaptureClipboard = func() (ClipboardContent, error) {
			return c.content, c.err
		}
		_, err := ReadTotpFromClipboard()
		if !errors.Is(err, c.want) {
			t.Fatalf("%s: err = %v, want %v", c.name, err, c.want)
		}
	}
}

func TestInitSchemaCreatesTotpTableOnExistingDB(t *testing.T) {
	openTestDB(t)

	if err := GetDB().Exec(`
		CREATE TABLE pocket_config (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			config_key TEXT NOT NULL,
			config_value TEXT NOT NULL
		)
	`).Error; err != nil {
		t.Fatal(err)
	}
	if hasTable(t, "pocket_totp") {
		t.Fatal("pocket_totp exists before InitSchema")
	}

	if err := InitSchema(); err != nil {
		t.Fatal(err)
	}
	if !hasTable(t, "pocket_totp") {
		t.Fatal("pocket_totp not created on an existing database")
	}

	var configCount int
	if err := GetDB().Raw(`SELECT count(*) FROM pocket_config`).Scan(&configCount).Error; err != nil {
		t.Fatal(err)
	}
	if configCount != 0 {
		t.Fatalf("got %d pocket_config records on an existing database, want 0", configCount)
	}
}

func TestInitSchemaCreatesTotpTableOnFreshDB(t *testing.T) {
	openTestDB(t)
	_initSchemaFlag = true

	if err := InitSchema(); err != nil {
		t.Fatal(err)
	}
	if !hasTable(t, "pocket_totp") {
		t.Fatal("pocket_totp not created on a fresh database")
	}
}

func TestCreateTotpEntryEncryptsSecret(t *testing.T) {
	openTestDB(t)
	if err := InitSchema(); err != nil {
		t.Fatal(err)
	}

	e := testTotpEntry("GitHub", "JBSWY3DPEHPK3PXP", Now())
	e.Remark = "work"
	created, err := CreateTotpEntry(e)
	if err != nil {
		t.Fatal(err)
	}
	if created.Id == 0 {
		t.Fatal("created entry has no id")
	}

	var stored string
	if err := GetDB().Raw(`SELECT secret FROM pocket_totp WHERE id = ?`, created.Id).Scan(&stored).Error; err != nil {
		t.Fatal(err)
	}
	if stored == e.Secret {
		t.Fatal("secret is stored in plaintext")
	}
	dec, err := Decrypt(stored)
	if err != nil {
		t.Fatal(err)
	}
	if dec != e.Secret {
		t.Fatalf("stored secret decrypts to %q, want %q", dec, e.Secret)
	}
}

func TestFetchTotpEntriesNewestFirst(t *testing.T) {
	openTestDB(t)
	if err := InitSchema(); err != nil {
		t.Fatal(err)
	}

	issuers := []string{"old", "middle", "new"}
	secrets := []string{"JBSWY3DPEHPK3PXP", "KRSXG5CTMVRXEZLU", "MFRGGZDFMZTWQ2LK"}
	base := time.Date(2026, 11, 6, 10, 0, 0, 0, time.UTC)
	for i := range issuers {
		ctime := ETime(base.Add(time.Duration(i) * time.Second))
		e := testTotpEntry(issuers[i], secrets[i], ctime)
		if _, err := CreateTotpEntry(e); err != nil {
			t.Fatal(err)
		}
	}

	entries, err := FetchTotpEntries()
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != len(issuers) {
		t.Fatalf("got %d entries, want %d", len(entries), len(issuers))
	}
	wantIssuers := []string{"new", "middle", "old"}
	wantSecrets := []string{secrets[2], secrets[1], secrets[0]}
	for i := range entries {
		if entries[i].Issuer != wantIssuers[i] {
			t.Fatalf("entry %d issuer = %q, want %q", i, entries[i].Issuer, wantIssuers[i])
		}
		if entries[i].Secret != wantSecrets[i] {
			t.Fatalf("entry %d secret = %q, want %q", i, entries[i].Secret, wantSecrets[i])
		}
	}
}

func TestFetchTotpEntriesToleratesUndecryptableSecret(t *testing.T) {
	openTestDB(t)
	if err := InitSchema(); err != nil {
		t.Fatal(err)
	}

	base := time.Date(2026, 11, 6, 10, 0, 0, 0, time.UTC)
	if _, err := CreateTotpEntry(testTotpEntry("good", "JBSWY3DPEHPK3PXP", ETime(base))); err != nil {
		t.Fatal(err)
	}
	bad, err := CreateTotpEntry(testTotpEntry("bad", "KRSXG5CTMVRXEZLU", ETime(base.Add(time.Second))))
	if err != nil {
		t.Fatal(err)
	}
	if err := GetDB().Exec(`UPDATE pocket_totp SET secret = 'not-ciphertext' WHERE id = ?`, bad.Id).Error; err != nil {
		t.Fatal(err)
	}

	entries, err := FetchTotpEntries()
	if err != nil {
		t.Fatalf("fetch aborted on an undecryptable secret: %v", err)
	}
	if len(entries) != 2 {
		t.Fatalf("got %d entries, want 2", len(entries))
	}
	for _, e := range entries {
		switch e.Issuer {
		case "good":
			if e.Secret != "JBSWY3DPEHPK3PXP" {
				t.Fatalf("good secret = %q, want %q", e.Secret, "JBSWY3DPEHPK3PXP")
			}
		case "bad":
			if _, err := GenerateTotpCode(e.Secret, e.Digits, e.Period, e.Algorithm, time.Unix(59, 0)); !errors.Is(err, ErrSecretUndecryptable) {
				t.Fatalf("bad entry code generation err = %v, want %v", err, ErrSecretUndecryptable)
			}
		default:
			t.Fatalf("unexpected issuer %q", e.Issuer)
		}
	}
}

func TestUpdateTotpAccountAndRemark(t *testing.T) {
	openTestDB(t)
	if err := InitSchema(); err != nil {
		t.Fatal(err)
	}

	original := testTotpEntry("GitHub", "JBSWY3DPEHPK3PXP", Now())
	original.Remark = "work"
	original.Digits = 8
	original.Period = 60
	original.Algorithm = "SHA256"
	created, err := CreateTotpEntry(original)
	if err != nil {
		t.Fatal(err)
	}

	var storedSecretBefore string
	if err := GetDB().Raw(`SELECT secret FROM pocket_totp WHERE id = ?`, created.Id).Scan(&storedSecretBefore).Error; err != nil {
		t.Fatal(err)
	}

	created.Account = "alice@work.example"
	created.Remark = "personal"
	updated := ETime(time.Date(2026, 11, 6, 10, 30, 0, 0, time.UTC))
	created.Utime = updated
	if err := UpdateTotpAccountAndRemark(created); err != nil {
		t.Fatal(err)
	}

	entries, err := FetchTotpEntries()
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 {
		t.Fatalf("got %d entries, want 1", len(entries))
	}
	e := entries[0]
	if e.Account != "alice@work.example" {
		t.Fatalf("account = %q, want %q", e.Account, "alice@work.example")
	}
	if e.Remark != "personal" {
		t.Fatalf("remark = %q, want %q", e.Remark, "personal")
	}

	uri, err := BuildTotpUri(e)
	if err != nil {
		t.Fatal(err)
	}
	parsed, err := ParseTotpText(uri)
	if err != nil {
		t.Fatal(err)
	}
	if parsed.Account != "alice@work.example" {
		t.Fatalf("exported uri account = %q, want %q", parsed.Account, "alice@work.example")
	}

	// The editable fields change; identity and generation parameters do not.
	if e.Issuer != original.Issuer {
		t.Fatalf("issuer = %q, want %q", e.Issuer, original.Issuer)
	}
	if e.Secret != original.Secret {
		t.Fatalf("secret = %q, want %q", e.Secret, original.Secret)
	}
	if e.Digits != original.Digits {
		t.Fatalf("digits = %d, want %d", e.Digits, original.Digits)
	}
	if e.Period != original.Period {
		t.Fatalf("period = %d, want %d", e.Period, original.Period)
	}
	if e.Algorithm != original.Algorithm {
		t.Fatalf("algorithm = %q, want %q", e.Algorithm, original.Algorithm)
	}

	var storedSecretAfter string
	if err := GetDB().Raw(`SELECT secret FROM pocket_totp WHERE id = ?`, created.Id).Scan(&storedSecretAfter).Error; err != nil {
		t.Fatal(err)
	}
	if storedSecretAfter != storedSecretBefore {
		t.Fatal("update rewrote the stored secret")
	}

	// utime is a DATETIME column, which go-sqlite3 only parses back into a time
	// value for dash-separated formats, so read the stored text to check the
	// update time was written.
	var storedUtime string
	if err := GetDB().Raw(`SELECT CAST(utime AS TEXT) FROM pocket_totp WHERE id = ?`, created.Id).Scan(&storedUtime).Error; err != nil {
		t.Fatal(err)
	}
	if want := updated.FormatClassic(); storedUtime != want {
		t.Fatalf("stored utime = %q, want the new update time %q", storedUtime, want)
	}
}

func TestDeleteTotpEntry(t *testing.T) {
	openTestDB(t)
	if err := InitSchema(); err != nil {
		t.Fatal(err)
	}

	created, err := CreateTotpEntry(testTotpEntry("GitHub", "JBSWY3DPEHPK3PXP", Now()))
	if err != nil {
		t.Fatal(err)
	}

	if err := DeleteTotpEntry(created); err != nil {
		t.Fatal(err)
	}

	entries, err := FetchTotpEntries()
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		t.Fatalf("got %d entries after delete, want 0", len(entries))
	}
}

func TestCreateTotpEntryFailureLeavesNoRow(t *testing.T) {
	openTestDB(t)
	if err := InitSchema(); err != nil {
		t.Fatal(err)
	}

	if err := GetDB().Exec(`
		CREATE TRIGGER fail_totp_insert BEFORE INSERT ON pocket_totp
		BEGIN
			SELECT RAISE(ABORT, 'boom');
		END
	`).Error; err != nil {
		t.Fatal(err)
	}

	_, err := CreateTotpEntry(testTotpEntry("GitHub", "JBSWY3DPEHPK3PXP", Now()))
	if err == nil {
		t.Fatal("expected create to fail")
	}

	if err := GetDB().Exec(`DROP TRIGGER fail_totp_insert`).Error; err != nil {
		t.Fatal(err)
	}
	entries, err := FetchTotpEntries()
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		t.Fatalf("got %d entries after failed create, want 0", len(entries))
	}
}

const (
	rfc6238SHA1Seed   = "GEZDGNBVGY3TQOJQGEZDGNBVGY3TQOJQ"
	rfc6238SHA256Seed = "GEZDGNBVGY3TQOJQGEZDGNBVGY3TQOJQGEZDGNBVGY3TQOJQGEZA"
	rfc6238SHA512Seed = "GEZDGNBVGY3TQOJQGEZDGNBVGY3TQOJQGEZDGNBVGY3TQOJQGEZDGNBVGY3TQOJQGEZDGNBVGY3TQOJQGEZDGNBVGY3TQOJQGEZDGNA"
)

func TestGenerateTotpCodeRfc6238Vectors(t *testing.T) {
	cases := []struct {
		name      string
		secret    string
		algorithm string
		digits    int
		period    int
		unix      int64
		want      string
	}{
		{"sha1 t=59", rfc6238SHA1Seed, "SHA1", 8, 30, 59, "94287082"},
		{"sha256 t=59", rfc6238SHA256Seed, "SHA256", 8, 30, 59, "46119246"},
		{"sha512 t=59", rfc6238SHA512Seed, "SHA512", 8, 30, 59, "90693936"},
		{"sha1 t=1111111109", rfc6238SHA1Seed, "SHA1", 8, 30, 1111111109, "07081804"},
		{"sha256 t=1111111109", rfc6238SHA256Seed, "SHA256", 8, 30, 1111111109, "68084774"},
		{"sha512 t=1111111109", rfc6238SHA512Seed, "SHA512", 8, 30, 1111111109, "25091201"},
		{"sha1 t=1234567890", rfc6238SHA1Seed, "SHA1", 8, 30, 1234567890, "89005924"},
		{"sha256 t=1234567890", rfc6238SHA256Seed, "SHA256", 8, 30, 1234567890, "91819424"},
		{"sha512 t=1234567890", rfc6238SHA512Seed, "SHA512", 8, 30, 1234567890, "93441116"},
		// A six-digit code is the same truncated value modulo 10^6.
		{"sha1 6 digits", rfc6238SHA1Seed, "SHA1", 6, 30, 59, "287082"},
		{"sha256 6 digits", rfc6238SHA256Seed, "SHA256", 6, 30, 59, "119246"},
		{"sha512 6 digits", rfc6238SHA512Seed, "SHA512", 6, 30, 59, "693936"},
		// RFC 6238 vector with a non-default 60-second period.
		{"sha1 period 60", rfc6238SHA1Seed, "SHA1", 8, 60, 59, "84755224"},
	}
	for _, c := range cases {
		got, err := GenerateTotpCode(c.secret, c.digits, c.period, c.algorithm, time.Unix(c.unix, 0))
		if err != nil {
			t.Fatalf("%s: %v", c.name, err)
		}
		if got != c.want {
			t.Fatalf("%s: code = %q, want %q", c.name, got, c.want)
		}
	}
}

func TestGenerateTotpCodeWithoutSecretFails(t *testing.T) {
	if _, err := GenerateTotpCode("", 6, 30, "SHA1", time.Unix(59, 0)); !errors.Is(err, ErrSecretUndecryptable) {
		t.Fatalf("err = %v, want %v", err, ErrSecretUndecryptable)
	}
}
