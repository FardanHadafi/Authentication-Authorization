import { fail, redirect, type Actions } from "@sveltejs/kit";
import type { PageServerLoad } from "./$types";

const API_URL = import.meta.env.VITE_API_URL!;

export const load = (async ({ locals, fetch, cookies }) => {
  // Guard autentikasi dasar
  if (!locals.user) {
    redirect(303, "/login");
  }

  const accessToken = cookies.get("access_token");
  if (!accessToken) {
    redirect(303, "/login");
  }

  // 1. Ambil data user profile dari backend
  let profile = locals.user;
  let permissions: string[] = [];

  try {
    const profileRes = await fetch(`${API_URL}/auth/me`, {
      method: "GET",
      headers: { "Authorization": `Bearer ${accessToken}` }
    });

    if (profileRes.status === 401) {
      cookies.delete("access_token", { path: "/" });
      cookies.delete("refresh_token", { path: "/" });
      redirect(303, "/login");
    }

    if (profileRes.ok) {
      profile = await profileRes.json();
    }

    // 2. Ambil izin granular PBAC user
    const permsRes = await fetch(`${API_URL}/auth/me/permissions`, {
      method: "GET",
      headers: { "Authorization": `Bearer ${accessToken}` }
    });

    if (permsRes.ok) {
      const permsData = await permsRes.json();
      permissions = permsData.permissions || [];
    }
  } catch {
    // Gunakan fallback locals jika backend timeout
  }

  return {
    user: profile,
    permissions
  };
}) satisfies PageServerLoad;

export const actions: Actions = {
  logout: async ({ fetch, cookies }) => {
    const refreshToken = cookies.get("refresh_token");

    try {
      if (refreshToken) {
        await fetch(`${API_URL}/auth/logout`, {
          method: "POST",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify({ refresh_token: refreshToken })
        });
      }
    } catch {
      // Abaikan error jaringan saat logout
    }

    cookies.delete("access_token", { path: "/" });
    cookies.delete("refresh_token", { path: "/" });

    redirect(303, "/login");
  },

  setupMfa: async ({ fetch, cookies }) => {
    const accessToken = cookies.get("access_token");
    if (!accessToken) {
      redirect(303, "/login");
    }

    try {
      const res = await fetch(`${API_URL}/auth/mfa/setup`, {
        method: "POST",
        headers: {
          "Authorization": `Bearer ${accessToken}`,
          "Content-Type": "application/json"
        }
      });

      if (!res.ok) {
        const { error } = await res.json();
        return fail(res.status, { mfaActionError: error || "Gagal memulai setup MFA" });
      }

      const setupData = await res.json();
      return {
        setupData: {
          secret: setupData.secret,
          qrCodeUrl: setupData.qr_code_url,
          recoveryCodes: setupData.recovery_codes || []
        }
      };
    } catch {
      return fail(500, { mfaActionError: "Gagal terhubung dengan server backend" });
    }
  },

  enableMfa: async ({ request, fetch, cookies }) => {
    const accessToken = cookies.get("access_token");
    if (!accessToken) {
      redirect(303, "/login");
    }

    const formData = await request.formData();
    const code = (formData.get("code") as string)?.trim();

    if (!code) {
      return fail(400, { mfaActionError: "Kode verifikasi 6-digit wajib diisi" });
    }

    try {
      const res = await fetch(`${API_URL}/auth/mfa/enable`, {
        method: "POST",
        headers: {
          "Authorization": `Bearer ${accessToken}`,
          "Content-Type": "application/json"
        },
        body: JSON.stringify({ code })
      });

      if (!res.ok) {
        const { error } = await res.json();
        return fail(res.status, { mfaActionError: error || "Kode verifikasi salah atau kedaluwarsa" });
      }

      return {
        mfaSuccessMessage: "Multi-Factor Authentication (2FA) berhasil diaktifkan!"
      };
    } catch {
      return fail(500, { mfaActionError: "Gagal terhubung dengan server backend" });
    }
  },

  disableMfa: async ({ request, fetch, cookies }) => {
    const accessToken = cookies.get("access_token");
    if (!accessToken) {
      redirect(303, "/login");
    }

    const formData = await request.formData();
    const code = (formData.get("code") as string)?.trim();
    const password = (formData.get("password") as string)?.trim();

    if (!code || !password) {
      return fail(400, { mfaActionError: "Password dan kode 6-digit TOTP wajib diisi" });
    }

    try {
      const res = await fetch(`${API_URL}/auth/mfa/disable`, {
        method: "POST",
        headers: {
          "Authorization": `Bearer ${accessToken}`,
          "Content-Type": "application/json"
        },
        body: JSON.stringify({ code, password })
      });

      if (!res.ok) {
        const { error } = await res.json();
        return fail(res.status, { mfaActionError: error || "Gagal menonaktifkan 2FA. Periksa password dan kode Anda." });
      }

      return {
        mfaSuccessMessage: "2FA berhasil dinonaktifkan."
      };
    } catch {
      return fail(500, { mfaActionError: "Gagal terhubung dengan server backend" });
    }
  }
};
