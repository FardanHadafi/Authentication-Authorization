import type { Handle } from '@sveltejs/kit/hooks';

const API_URL = import.meta.env.VITE_API_URL!;

/**
 * Decode payload JWT tanpa memverifikasi signature (signature diverifikasi oleh backend Go)
 */
function parseJwtPayload(token: string): { user_id?: string; sub?: string; email?: string; role?: string; exp?: number } | null {
	try {
		const parts = token.split('.');
		if (parts.length !== 3) return null;
		const base64Url = parts[1];
		const base64 = base64Url.replace(/-/g, '+').replace(/_/g, '/');
		const jsonPayload = decodeURIComponent(
			atob(base64)
				.split('')
				.map((c) => '%' + ('00' + c.charCodeAt(0).toString(16)).slice(-2))
				.join('')
		);
		return JSON.parse(jsonPayload);
	} catch {
		return null;
	}
}

export const handle: Handle = async ({ event, resolve }) => {
	const accessToken = event.cookies.get('access_token');
	const refreshToken = event.cookies.get('refresh_token');

	let user = null;

	if (accessToken) {
		const payload = parseJwtPayload(accessToken);
		const isExpired = payload?.exp ? payload.exp * 1000 <= Date.now() : true;

		if (payload && !isExpired) {
			user = {
				id: (payload.user_id || payload.sub) as string,
				email: payload.email as string,
				role: (payload.role || 'member') as string
			};
		}
	}

	// Silent Refresh: Jika access token expired/hilang tapi refresh token masih ada
	if (!user && refreshToken) {
		try {
			const res = await event.fetch(`${API_URL}/auth/refresh`, {
				method: 'POST',
				headers: { 'Content-Type': 'application/json' },
				body: JSON.stringify({ refresh_token: refreshToken })
			});

			if (res.ok) {
				const data = await res.json();
				const isProduction = import.meta.env.PROD;

				if (data.access_token) {
					event.cookies.set('access_token', data.access_token, {
						path: '/',
						httpOnly: true,
						secure: isProduction,
						sameSite: 'lax',
						maxAge: 15 * 60
					});

					const newPayload = parseJwtPayload(data.access_token);
					if (newPayload) {
						user = {
							id: (newPayload.user_id || newPayload.sub) as string,
							email: newPayload.email as string,
							role: (newPayload.role || 'member') as string
						};
					}
				}

				if (data.refresh_token) {
					event.cookies.set('refresh_token', data.refresh_token, {
						path: '/',
						httpOnly: true,
						secure: isProduction,
						sameSite: 'lax',
						maxAge: 7 * 24 * 60 * 60
					});
				}
			} else {
				// Refresh token sudah tidak valid / expired: hapus cookie
				event.cookies.delete('access_token', { path: '/' });
				event.cookies.delete('refresh_token', { path: '/' });
			}
		} catch {
			// Backend down atau network error: biarkan user null
			user = null;
		}
	}

	// Pasang user ke event.locals
	event.locals.user = user;

	return resolve(event);
};
