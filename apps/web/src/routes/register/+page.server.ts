import { RegisterSchema } from '#lib/schemas/auth.ts';
import { fail, type Actions } from '@sveltejs/kit';
import * as v from 'valibot';

const API_URL = import.meta.env.VITE_API_URL!;

export const actions: Actions = {
	default: async ({ request, fetch }) => {
		const formData = await request.formData();
		const result = v.safeParse(RegisterSchema, Object.fromEntries(formData));
		if (!result.success) {
			return fail(400, {
				errors: v.flatten<typeof RegisterSchema>(result.issues).nested
			});
		}

		// Teruskan ke Go backend (/api/auth/register)
		try {
			const response = await fetch(`${API_URL}/auth/register`, {
				method: 'POST',
				headers: { 'Content-Type': 'application/json' },
				body: JSON.stringify({
					full_name: result.output.full_name,
					email: result.output.email,
					password: result.output.password
				})
			});

			if (!response.ok) {
				const { error } = await response.json();
				return fail(response.status, { message: error || 'Pendaftaran Gagal' });
			}

			return { success: true };
		} catch {
			return fail(500, { message: 'Gagal terhubung dengan server backend' });
		}
	}
};
