import { redirect, type Actions } from "@sveltejs/kit";
import type { PageServerLoad } from "./$types";

const API_URL = import.meta.env.VITE_API_URL || "http://localhost:8080/api";

export const load: PageServerLoad = async ({ locals }) => {
  if (!locals.user) {
    redirect(303, "/login");
  }

  return {
    user: locals.user
  };
};

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
