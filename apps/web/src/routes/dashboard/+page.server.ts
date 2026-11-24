import { redirect, type Actions } from "@sveltejs/kit";
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

  // Ambil data user profile dari backend
  try {
    const response = await fetch(`${API_URL}/auth/me`, {
      method: "GET",
      headers: {
        "Authorization": `Bearer ${accessToken}`
      }
    });

    // Jika token invalid / expired
    if (response.status === 401) {
      cookies.delete("access_token", { path: "/" });
      cookies.delete("refresh_token", { path: "/" });
      redirect(303, "/login");
    }

    if (!response.ok) {
      // Fallback aman jika terjadi error selain 401
      return { user: locals.user };
    }

    const profile = await response.json();
    return { user: profile };
  } catch {
    // Fallback aman jika backend tidak merespons
    return { user: locals.user };
  }
}) satisfies PageServerLoad;

export const actions: Actions = {
  logout: async ({ fetch, cookies }) => {
    const refreshToken = cookies.get("refresh_token");

    // Panggil backend Go untuk mencabut (revoke) refresh token dari Redis
    try {
      if (refreshToken) {
        await fetch(`${API_URL}/auth/logout`, {
          method: "POST",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify({ refresh_token: refreshToken })
        });
      }
    } catch {
      // Abaikan error jaringan backend saat logout agar cookie tetap bersih
    }

    // Hapus cookies dari SvelteKit BFF
    cookies.delete("access_token", { path: "/" });
    cookies.delete("refresh_token", { path: "/" });

    redirect(303, "/login");
  }
};
