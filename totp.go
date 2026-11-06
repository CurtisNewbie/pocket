package main

import (
	"encoding/base32"
	"errors"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/pquerna/otp"
	"github.com/pquerna/otp/totp"
)

// ParsedTotp is the result of parsing import text. Bare is true when the text
// carried a bare Base32 secret instead of an otpauth URI; such an entry has no
// issuer or account until the user supplies them.
type ParsedTotp struct {
	Bare      bool
	Issuer    string
	Account   string
	Secret    string
	Digits    int
	Period    int
	Algorithm string
}

var (
	ErrUriInvalid          = errors.New("uri-invalid")
	ErrHotpUnsupported     = errors.New("hotp-unsupported")
	ErrSecretInvalid       = errors.New("secret-invalid")
	ErrParamInvalid        = errors.New("param-invalid")
	ErrSecretUndecryptable = errors.New("secret-undecryptable")
)

// GenerateTotpCode returns the TOTP code for the secret at t. An empty secret
// means the stored secret could not be decrypted, so generation reports
// secret-undecryptable instead of a code.
func GenerateTotpCode(secret string, digits int, period int, algorithm string, t time.Time) (string, error) {
	if secret == "" {
		return "", ErrSecretUndecryptable
	}
	alg, ok := totpAlgorithm(algorithm)
	if !ok {
		return "", ErrParamInvalid
	}
	return totp.GenerateCodeCustom(secret, t, totp.ValidateOpts{
		Period:    uint(period),
		Digits:    otp.Digits(digits),
		Algorithm: alg,
	})
}

func totpAlgorithm(name string) (otp.Algorithm, bool) {
	switch name {
	case "SHA1":
		return otp.AlgorithmSHA1, true
	case "SHA256":
		return otp.AlgorithmSHA256, true
	case "SHA512":
		return otp.AlgorithmSHA512, true
	default:
		return otp.AlgorithmSHA1, false
	}
}

// ParseTotpText parses clipboard text as an otpauth URI or a bare Base32
// secret. URI values that are absent take the TOTP defaults.
func ParseTotpText(text string) (ParsedTotp, error) {
	text = strings.TrimSpace(text)
	if !strings.HasPrefix(strings.ToLower(text), "otpauth://") {
		return parseBareSecret(text)
	}

	u, err := url.Parse(text)
	if err != nil {
		return ParsedTotp{}, ErrUriInvalid
	}
	if !strings.EqualFold(u.Scheme, "otpauth") {
		return ParsedTotp{}, ErrUriInvalid
	}
	switch strings.ToLower(u.Host) {
	case "totp":
	case "hotp":
		return ParsedTotp{}, ErrHotpUnsupported
	default:
		return ParsedTotp{}, ErrUriInvalid
	}

	q := u.Query()
	issuer, account, err := splitTotpLabel(u.EscapedPath())
	if err != nil {
		return ParsedTotp{}, ErrUriInvalid
	}
	if v := q.Get("issuer"); v != "" {
		issuer = v
	}

	secret := q.Get("secret")
	if secret == "" {
		return ParsedTotp{}, ErrUriInvalid
	}

	digits := 6
	if v := q.Get("digits"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || (n != 6 && n != 8) {
			return ParsedTotp{}, paramInvalid("digits must be 6 or 8")
		}
		digits = n
	}

	period := 30
	if v := q.Get("period"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || n < 1 || n > 3600 {
			return ParsedTotp{}, paramInvalid("period must be 1 to 3600 seconds")
		}
		period = n
	}

	algorithm := "SHA1"
	if v := q.Get("algorithm"); v != "" {
		algorithm = strings.ToUpper(v)
	}
	if _, ok := totpAlgorithm(algorithm); !ok {
		return ParsedTotp{}, paramInvalid("algorithm must be SHA1, SHA256, or SHA512")
	}

	secret, err = normalizeBase32(secret)
	if err != nil {
		return ParsedTotp{}, err
	}

	parsed := ParsedTotp{
		Issuer:    issuer,
		Account:   account,
		Secret:    secret,
		Digits:    digits,
		Period:    period,
		Algorithm: algorithm,
	}
	return parsed, nil
}

// BuildTotpUri returns the otpauth URI for a stored entry. The entry's Secret
// must already be decrypted; an empty secret reports secret-undecryptable.
// Absent issuer or account are left out of the label, and every parameter is
// written explicitly so the URI parses back to the same values.
func BuildTotpUri(e TotpEntry) (string, error) {
	if e.Secret == "" {
		return "", ErrSecretUndecryptable
	}

	label := escapeTotpPart(e.Account)
	if e.Issuer != "" {
		label = escapeTotpPart(e.Issuer) + ":" + label
	}

	q := url.Values{}
	q.Set("secret", e.Secret)
	if e.Issuer != "" {
		q.Set("issuer", e.Issuer)
	}
	q.Set("algorithm", e.Algorithm)
	q.Set("digits", strconv.Itoa(e.Digits))
	q.Set("period", strconv.Itoa(e.Period))

	return "otpauth://totp/" + label + "?" + q.Encode(), nil
}

// escapeTotpPart percent-encodes one label segment. Query escaping is used
// because it encodes the label delimiters (":", "/", "@") that would otherwise
// change how the parser splits the label; a space becomes %20 rather than "+"
// so the path decodes back to the original text.
func escapeTotpPart(s string) string {
	return strings.ReplaceAll(url.QueryEscape(s), "+", "%20")
}

func parseBareSecret(text string) (ParsedTotp, error) {
	if !looksLikeBareSecret(text) {
		return ParsedTotp{}, ErrUriInvalid
	}
	secret, err := normalizeBase32(text)
	if err != nil {
		return ParsedTotp{}, err
	}
	return ParsedTotp{
		Bare:      true,
		Secret:    secret,
		Digits:    6,
		Period:    30,
		Algorithm: "SHA1",
	}, nil
}

// looksLikeBareSecret reports whether the text is a single token that could
// have been meant as a Base32 secret, rather than arbitrary clipboard text.
func looksLikeBareSecret(s string) bool {
	s = strings.TrimSpace(s)
	if s == "" {
		return false
	}
	for _, r := range s {
		if (r >= 'A' && r <= 'Z') || (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '=' {
			continue
		}
		return false
	}
	return true
}

func normalizeBase32(s string) (string, error) {
	s = strings.ToUpper(strings.TrimSpace(s))
	s = strings.TrimRight(s, "=")
	if s == "" {
		return "", ErrSecretInvalid
	}
	if _, err := base32.StdEncoding.WithPadding(base32.NoPadding).DecodeString(s); err != nil {
		return "", ErrSecretInvalid
	}
	return s, nil
}

// splitTotpLabel splits the URI path into the issuer prefix and the account.
// The path arrives percent-encoded, so a colon inside either part is still %3A
// and only a literal colon separates the issuer from the account.
func splitTotpLabel(escapedPath string) (issuer string, account string, err error) {
	label := strings.TrimPrefix(escapedPath, "/")
	rawIssuer, rawAccount, ok := strings.Cut(label, ":")
	if !ok {
		rawIssuer, rawAccount = "", label
	}
	issuer, err = url.PathUnescape(rawIssuer)
	if err != nil {
		return "", "", err
	}
	account, err = url.PathUnescape(rawAccount)
	if err != nil {
		return "", "", err
	}
	return issuer, account, nil
}

func paramInvalid(msg string) error {
	return fmt.Errorf("%w: %s", ErrParamInvalid, msg)
}
