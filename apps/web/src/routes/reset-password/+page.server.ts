import { fail, redirect, type Actions } from "@sveltejs/kit";
import type { PageServerLoad } from "./$types";
import * as v from "valibot";
import { ResetPasswordSchema } from "#lib/schemas/auth.ts";

const API_URL = import.meta.env.VITE_API_URL!;

export const load = (async ({ locals, url }) => {
  // Jika sudah memiliki sesi login aktif, arahkan ke dashboard
  if (locals.user) {
    redirect(303, "/dashboard");
  }

  // Ambil token dari query parameter (?token=)
  const token = url.searchParams.get("token");

  // Jika tidak ada token di URL, kembalikan ke halaman login
  if (!token) {
    redirect(303, "/login");
  }

  return { token };
}) satisfies PageServerLoad;

export const actions = {
  default: async ({ request, fetch }) => {
    const formData = await request.formData();
    const tokenRaw = formData.get("token")?.toString() || "";

    // Validasi input menggunakan valibot
    const result = v.safeParse(ResetPasswordSchema, Object.fromEntries(formData));
    if (!result.success) {
      return fail(400, {
        errors: v.flatten<typeof ResetPasswordSchema>(result.issues).nested,
        token: tokenRaw
      });
    }

    // Kirim request ke backend
    try {
      const response = await fetch(`${API_URL}/auth/reset-password`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          token: result.output.token,
          new_password: result.output.new_password,
          confirm_new_password: result.output.confirm_new_password,
        })
      });

      if (!response.ok) {
        const { error } = await response.json();
        return fail(response.status, {
          message: error || "Gagal reset password",
          token: tokenRaw,
        });
      }
    } catch {
      return fail(500, {
        message: "Gagal terhubung dengan server",
        token: tokenRaw
      });
    }

    // Arahkan ke halaman login dengan query reset=true setelah berhasil
    redirect(303, "/login?reset=true");
  }
} satisfies Actions;
