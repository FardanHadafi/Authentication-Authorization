import { error, isHttpError, redirect } from "@sveltejs/kit";
import type { PageServerLoad } from "./$types";

const API_URL = import.meta.env.VITE_API_URL!;

export const load = (async ({ locals, fetch, cookies }) => {
  // Guard autentikasi, harus sudah login
  if (!locals.user) {
    redirect(303, "/login");
  }

  // Guard Authorization (Admin Only Role)
  if (locals.user.role !== "admin") {
    error(403, "Akses Ditolak: Halaman ini hanya dapat diakses oleh Administrator.");
  }

  const accessToken = cookies.get("access_token");
  if (!accessToken) {
    redirect(303, "/login");
  }

  let users = [];
  let auditLogs = [];
  let totalAuditLogs = 0;

  try {
    // 1. Ambil daftar pengguna
    const usersRes = await fetch(`${API_URL}/admin/users`, {
      method: "GET",
      headers: { "Authorization": `Bearer ${accessToken}` }
    });

    if (usersRes.status === 401) {
      cookies.delete("access_token", { path: "/" });
      cookies.delete("refresh_token", { path: "/" });
      redirect(303, "/login");
    }

    if (usersRes.status === 403) {
      error(403, "Akses Ditolak: Izin tidak mencukupi untuk mengakses data ini.");
    }

    if (usersRes.ok) {
      users = await usersRes.json();
    }

    // 2. Ambil catatan audit log kepatuhan (SOC 2 Audit Trail)
    const logsRes = await fetch(`${API_URL}/admin/audit-logs?limit=50`, {
      method: "GET",
      headers: { "Authorization": `Bearer ${accessToken}` }
    });

    if (logsRes.ok) {
      const logsData = await logsRes.json();
      auditLogs = logsData.data || [];
      totalAuditLogs = logsData.total_count || 0;
    }

    return {
      user: locals.user,
      users,
      auditLogs,
      totalAuditLogs
    };
  } catch (err) {
    if (isHttpError(err)) {
      throw err;
    }

    error(500, "Gagal terhubung dengan server backend");
  }
}) satisfies PageServerLoad;
