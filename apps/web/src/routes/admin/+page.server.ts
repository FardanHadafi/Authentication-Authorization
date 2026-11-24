import { error, isHttpError, redirect } from "@sveltejs/kit";
import type { PageServerLoad } from "./$types";

const API_URL = import.meta.env.VITE_API_URL!;

export const load = (async ({ locals, fetch, cookies }) => {
  // Guard autentikasi, harus sudah login
  if (!locals.user) {
    redirect(303, "/login");
  }

  // Guard Authorization / RBAC (Admin Only Role)
  if (locals.user.role !== "admin") {
    error(403, "Akses Ditolak: Halaman ini hanya dapat diakses oleh Administrator.");
  }

  const accessToken = cookies.get("access_token");
  if (!accessToken) {
    redirect(303, "/login");
  }

  // Panggil endpoint GET /api/admin/users
  try {
    const response = await fetch(`${API_URL}/admin/users`, {
      method: "GET",
      headers: { "Authorization": `Bearer ${accessToken}` }
    });

    // Jika token invalid / expired
    if (response.status === 401) {
      cookies.delete("access_token", { path: "/" });
      cookies.delete("refresh_token", { path: "/" });
      redirect(303, "/login");
    }

    // Jika role bukan "admin"
    if (response.status === 403) {
      error(403, "Akses Ditolak: Izin tidak mencukupi untuk mengakses data ini.");
    }

    if (!response.ok) {
      error(response.status, "Gagal memuat daftar pengguna dari server backend.");
    }

    const users = await response.json();
    return {
      user: locals.user,
      users
    };
  } catch (err) {
    if (isHttpError(err)) {
      throw err;
    }

    error(500, "Gagal terhubung dengan server backend");
  }
}) satisfies PageServerLoad;
