import { LoginSchema } from "#lib/schemas/auth.ts";
import { fail, redirect, type Actions } from "@sveltejs/kit";
import type { PageServerLoad } from "./$types";
import * as v from "valibot";

const API_URL = import.meta.env.VITE_API_URL!;

export const load: PageServerLoad = async ({ locals, url }) => {
  if (locals.user) {
    redirect(303, "/dashboard");
  }

  return {
    resetSuccess: url.searchParams.get("reset") === "true"
  };
};

export const actions: Actions = {
  default: async ({ request, fetch, cookies }) => {
    const formData = await request.formData();
    const result = v.safeParse(LoginSchema, Object.fromEntries(formData));
    if (!result.success) {
      return fail(400, {
        errors: v.flatten<typeof LoginSchema>(result.issues).nested
      });
    }

    try {
      // 1. Teruskan ke Go backend API
      const response = await fetch(`${API_URL}/auth/login`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          email: result.output.email,
          password: result.output.password,
        })
      });

      if (!response.ok) {
        const { error } = await response.json();
        return fail(response.status, { message: error || "Login Gagal" });
      }

      const data = await response.json();

      // 2. Pasang httpOnly cookie ke browser client melalui SvelteKit BFF
      const isProduction = import.meta.env.PROD;

      if (data.access_token) {
        cookies.set("access_token", data.access_token, {
          path: "/",
          httpOnly: true,
          secure: isProduction,
          sameSite: "lax",
          maxAge: 15 * 60
        });
      }

      if (data.refresh_token) {
        cookies.set("refresh_token", data.refresh_token, {
          path: "/",
          httpOnly: true,
          secure: isProduction,
          sameSite: "lax",
          maxAge: 7 * 24 * 60 * 60
        });
      }
    } catch {
      // Return fail saat jaringan/server terputus agar tidak meluncur ke redirect di bawah
      return fail(500, { message: "Gagal terhubung dengan server backend" });
    }

    // 3. Arahkan pengguna ke halaman dashboard setelah login berhasil
    redirect(303, "/dashboard");
  }
};
