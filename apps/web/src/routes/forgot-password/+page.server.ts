import * as v from "valibot";
import type { PageServerLoad } from "./$types";
import { fail, redirect, type Actions } from "@sveltejs/kit";
import { ForgotPasswordSchema } from "#lib/schemas/auth.ts";

const API_URL = import.meta.env.VITE_API_URL!;

export const load = (async ({ locals }) => {
  // Jika pengguna sudah memiliki sesi login aktif, arahkan langsung ke dashboard
  if (locals.user) {
    redirect(303, "/dashboard");
  }
}) satisfies PageServerLoad;

export const actions = {
  default: async ({ request, fetch }) => {
    const formData = await request.formData();
    const emailRaw = formData.get("email")?.toString() || "";

    // Validasi input menggunakan valibot
    const result = v.safeParse(ForgotPasswordSchema, Object.fromEntries(formData));
    if (!result.success) {
      return fail(400, {
        errors: v.flatten<typeof ForgotPasswordSchema>(result.issues).nested,
        email: emailRaw,
      });
    }

    // Kirim request ke backend
    try {
      const response = await fetch(`${API_URL}/auth/forgot-password`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          email: result.output.email
        })
      });

      if (!response.ok) {
        const { error } = await response.json();
        return fail(response.status, {
          message: error || "Gagal memproses permintaan reset password",
          email: emailRaw
        });
      }

      const data = await response.json();

      return {
        success: true,
        message: data.message || "Jika email terdaftar di sistem kami, instruksi reset kata sandi telah dikirimkan ke kotak masuk Anda."
      };
    } catch {
      return fail(500, {
        message: "Gagal terhubung dengan server",
        email: emailRaw
      });
    }
  }
} satisfies Actions;
