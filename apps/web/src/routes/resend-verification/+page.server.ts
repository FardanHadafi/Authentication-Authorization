import { ResendVerificationSchema } from "#lib/schemas/auth.ts";
import { fail, redirect, type Actions } from "@sveltejs/kit";
import type { PageServerLoad } from "./$types";
import * as v from "valibot";

const API_URL = import.meta.env.VITE_API_URL!;

export const load = (async ({ locals, url }) => {
  // Jika pengguna sudah memiliki sesi login aktif, arahkan ke dashboard
  if (locals.user) {
    redirect(303, "/dashboard");
  }

  return {
    email: url.searchParams.get("email") || ""
  };
}) satisfies PageServerLoad;

export const actions: Actions = {
  default: async ({ request, fetch }) => {
    const formData = await request.formData();
    const emailRaw = formData.get("email")?.toString() || "";

    // 1. Validasi input menggunakan Valibot
    const result = v.safeParse(ResendVerificationSchema, Object.fromEntries(formData));
    if (!result.success) {
      return fail(400, {
        errors: v.flatten<typeof ResendVerificationSchema>(result.issues).nested,
        email: emailRaw
      });
    }

    try {
      // 2. Kirim request ke backend Go
      const response = await fetch(`${API_URL}/auth/resend-verification`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          email: result.output.email
        })
      });

      if (!response.ok) {
        const { error } = await response.json();
        return fail(response.status, {
          message: error || "Gagal mengirim ulang email verifikasi",
          email: emailRaw
        });
      }

      const data = await response.json();

      return {
        success: true,
        message:
          data.message ||
          "Jika email terdaftar dan belum diverifikasi, tautan verifikasi baru telah dikirimkan ke kotak masuk Anda."
      };
    } catch {
      return fail(500, {
        message: "Gagal terhubung dengan server backend",
        email: emailRaw
      });
    }
  }
} satisfies Actions;
