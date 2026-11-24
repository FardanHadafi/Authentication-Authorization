package shared

import (
	"bytes"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"image/png"
	"slices"
	"strings"

	"github.com/pquerna/otp"
	"github.com/pquerna/otp/totp"
)

const (
	TOTPIssuer = "AuthAPI"
)

// GenerateTOTPSecret membuat secret key TOTP baru yang kompatibel dengan Google Authenticator
func GenerateTOTPSecret(accountName string) (*otp.Key, error) {
	return totp.Generate(totp.GenerateOpts{
		Issuer:      TOTPIssuer,
		AccountName: accountName,
		Period:      30,
		SecretSize:  20,
		Digits:      otp.DigitsSix,
		Algorithm:   otp.AlgorithmSHA1,
	})
}

// GenerateQRCodePNG menghasilkan base64 PNG data URL dari otp.Key untuk discan via Google Authenticator
func GenerateQRCodePNG(key *otp.Key, width, height int) (string, error) {
	if width <= 0 || height <= 0 {
		width = 250
		height = 250
	}
	img, err := key.Image(width, height)
	if err != nil {
		return "", fmt.Errorf("failed to generate QR code image: %w", err)
	}

	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return "", fmt.Errorf("failed to encode QR code as PNG: %w", err)
	}

	return "data:image/png;base64," + base64.StdEncoding.EncodeToString(buf.Bytes()), nil
}

// ValidateTOTPCode memverifikasi passcode 6-digit TOTP terhadap secret
func ValidateTOTPCode(passcode, secret string) bool {
	passcode = strings.TrimSpace(passcode)
	secret = strings.TrimSpace(secret)
	if passcode == "" || secret == "" {
		return false
	}
	return totp.Validate(passcode, secret)
}

// HashRecoveryCode menghasilkan hash SHA-256 dari kode pemulihan
func HashRecoveryCode(code string) string {
	code = strings.ToUpper(strings.TrimSpace(code))
	code = strings.ReplaceAll(code, "-", "")
	hash := sha256.Sum256([]byte(code))
	return hex.EncodeToString(hash[:])
}

// GenerateRecoveryCodes menghasilkan kumpulan kode pemulihan acak dan hash-nya
func GenerateRecoveryCodes(count int) (plainCodes []string, hashedCodes []string, err error) {
	if count <= 0 {
		count = 8
	}

	const charset = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789" // Tanpa karakter ambigu (0/O, 1/I)
	plainCodes = make([]string, count)
	hashedCodes = make([]string, count)

	bytes := make([]byte, count*8)
	if _, err := rand.Read(bytes); err != nil {
		return nil, nil, fmt.Errorf("failed to generate random bytes for recovery codes: %w", err)
	}

	for i := range count {
		part1 := make([]byte, 4)
		part2 := make([]byte, 4)
		for j := 0; j < 4; j++ {
			part1[j] = charset[bytes[i*8+j]%byte(len(charset))]
			part2[j] = charset[bytes[i*8+4+j]%byte(len(charset))]
		}
		plain := fmt.Sprintf("%s-%s", string(part1), string(part2))
		plainCodes[i] = plain
		hashedCodes[i] = HashRecoveryCode(plain)
	}

	return plainCodes, hashedCodes, nil
}

// ValidateAndConsumeRecoveryCode memeriksa apakah kode valid dan mengembalikannya dengan kode terpakai yang dihapus
func ValidateAndConsumeRecoveryCode(code string, storedHashedCodes []string) (bool, []string) {
	targetHash := HashRecoveryCode(code)
	idx := slices.Index(storedHashedCodes, targetHash)
	if idx == -1 {
		return false, storedHashedCodes
	}

	// Hapus kode yang sudah terpakai
	remaining := slices.Delete(slices.Clone(storedHashedCodes), idx, idx+1)
	return true, remaining
}
