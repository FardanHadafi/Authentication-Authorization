package shared

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

var (
	ErrInvalidToken = errors.New("invalid token")
	ErrExpiredToken = errors.New("token has expired")
)

type JWTClaims struct {
	UserID uuid.UUID `json:"user_id"`
	Email  string    `json:"email"`
	Role   string    `json:"role"`
	jwt.RegisteredClaims
}

type TokenManager interface {
	GenerateAccessToken(userID uuid.UUID, email, role string) (string, time.Time, error)
	GenerateRefreshToken() (string, error)
	ValidateAccessToken(tokenString string) (*JWTClaims, error)
}

type JWTManager struct {
	secretKey    []byte
	accessExpiry time.Duration
	issuer       string
}

func NewTokenManager(secretKey []byte, accessExpiry time.Duration, issuer string) TokenManager {
	return &JWTManager{
		secretKey:    []byte(secretKey),
		accessExpiry: accessExpiry,
		issuer:       issuer,
	}
}

func (t *JWTManager) GenerateAccessToken(userID uuid.UUID, email, role string) (string, time.Time, error) {
	// 1. Hitung timestampt saat ini dan batas waktu kadaluarsa token
	now := time.Now()
	expiresAt := now.Add(t.accessExpiry)

	// 2. Susun payload claim custom user beserta claim standard jwt
	claims := JWTClaims{
		UserID: userID,
		Email:  email,
		Role:   role,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    t.issuer,
			Subject:   userID.String(),
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(expiresAt),
			ID:        uuid.NewString(),
		},
	}

	// 3. Buat objek token baru menggunakan signing simetris HS256
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	// 4. Tanda tangan header dan payload memakai secretKey
	signedToken, err := token.SignedString(t.secretKey)
	if err != nil {
		return "", time.Time{}, fmt.Errorf("failed to signed token: %w", err)
	}

	return signedToken, expiresAt, nil
}

func (t *JWTManager) GenerateRefreshToken() (string, error) {
	// 1. Siapkan slice kosong sepanjang 32 byte
	randomBytes := make([]byte, 32)

	// 2. Isi slice kosong tersebut dengan byte acak dari kriptografi
	if _, err := rand.Read(randomBytes); err != nil {
		return "", fmt.Errorf("failed to make a random bytes: %w", err)
	}

	// 3. Ubah raw bytes menjadi string heksadesimal sepanjang 64 karakter
	refreshToken := hex.EncodeToString(randomBytes)

	// 4. Return
	return refreshToken, nil
}

func (t *JWTManager) ValidateAccessToken(tokenString string) (*JWTClaims, error) {
	// 1. Siapkan struct kosong sebaagai wadah hasil decoding claims
	claims := &JWTClaims{}

	// 2. Parse string token sekaligus jalankan fungsi verifikasi kunci
	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (any, error) {
		// Pastikan metode signing di header berasal dari keluarga HMAC
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("invalid signing method: %v", token.Header["alg"])
		}
		// Kembalikan secretKey agar library bisa mencocokan signature token
		return t.secretKey, nil
	},
		jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}),
	)

	// 3. Tangani error saat parsing, signature palsu, kadaluarsa
	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			return nil, ErrExpiredToken
		}
		return nil, fmt.Errorf("%w: %w", ErrInvalidToken, err)
	}

	// 4. Pastikan tipe klaim sesuai dan flag validasi internal library bernilai true
	parsedClaims, ok := token.Claims.(*JWTClaims)
	if !ok || !token.Valid {
		return nil, ErrInvalidToken
	}

	// 5. Return
	return parsedClaims, nil
}
