package shared

import (
	"net/http"
	"time"
)

// Tentukan nama cookie dan durasi dalam detik (standar http.Cookie.MaxAge)
const (
	AccessTokenCookieName  = "access_token"
	RefreshTokenCookieName = "refresh_token"

	AccessTokenMaxAge  = 15 * 60          // 15 menit (900 detik)
	RefreshTokenMaxAge = 7 * 24 * 60 * 60 // 7 hari (604.800 detik)
)

func SetAuthCookie(w http.ResponseWriter, accessToken, refreshToken string, isProd bool) {
	now := time.Now()

	// 1. SameSiteLaxMode aman dari CSRF dan ramah navigasi top-level
	sameSiteMode := http.SameSiteLaxMode

	// 2. Konfigurasi cookie untuk access token
	accessCookie := &http.Cookie{
		Name:     AccessTokenCookieName,
		Value:    accessToken,
		Path:     "/",
		MaxAge:   AccessTokenMaxAge,
		Expires:  now.Add(15 * time.Minute),
		HttpOnly: true,
		Secure:   isProd,
		SameSite: sameSiteMode,
	}

	// 3. Konfigurasi cookie untuk refresh token
	refreshCookie := &http.Cookie{
		Name:     RefreshTokenCookieName,
		Value:    refreshToken,
		Path:     "/",
		MaxAge:   RefreshTokenMaxAge,
		Expires:  now.Add(7 * 24 * time.Hour),
		HttpOnly: true,
		Secure:   isProd,
		SameSite: sameSiteMode,
	}

	// 4. Tulis Set-Cookie ke response header
	http.SetCookie(w, accessCookie)
	http.SetCookie(w, refreshCookie)
}

func ClearAuthCookie(w http.ResponseWriter, isProd bool) {
	sameSiteMode := http.SameSiteLaxMode

	clearAccess := &http.Cookie{
		Name:     AccessTokenCookieName,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		Expires:  time.Unix(0, 0),
		HttpOnly: true,
		Secure:   isProd,
		SameSite: sameSiteMode,
	}

	clearRefresh := &http.Cookie{
		Name:     RefreshTokenCookieName,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		Expires:  time.Unix(0, 0),
		HttpOnly: true,
		Secure:   isProd,
		SameSite: sameSiteMode,
	}

	http.SetCookie(w, clearAccess)
	http.SetCookie(w, clearRefresh)
}
