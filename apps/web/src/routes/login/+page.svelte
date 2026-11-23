<script lang="ts">
	import { enhance } from '$app/forms';
	import type { PageProps } from './$types';
	import * as Card from '#lib/components/ui/card/index.js';
	import * as Alert from '#lib/components/ui/alert/index.js';
	import Input from '#lib/components/ui/input/input.svelte';
	import Label from '#lib/components/ui/label/label.svelte';
	import Button from '#lib/components/ui/button/button.svelte';
	import CircleAlert from '@lucide/svelte/icons/circle-alert';
	import CircleCheck from '@lucide/svelte/icons/circle-check';

	let { data, form }: PageProps = $props();
	let isSubmitting = $state(false);
</script>

<svelte:head>
	<title>Masuk ke Akun</title>
	<meta name="description" content="Halaman login pengguna ke sistem autentikasi" />
</svelte:head>

<!-- Landmark Semantic: <main> -->
<main class="flex min-h-screen items-center justify-center p-4">
	<Card.Root class="w-full max-w-md shadow-lg">
		<Card.Header class="space-y-1 text-center">
			<!-- Semantic H1 -->
			<h1 class="text-2xl font-bold tracking-tight">Masuk ke Akun</h1>
			<Card.Description>Masukkan email dan password Anda untuk melanjutkan</Card.Description>
		</Card.Header>

		<Card.Content>
			<!-- Alert Sukses Reset Password -->
			{#if data?.resetSuccess}
				<Alert.Root class="mb-4 border-emerald-500/30 bg-emerald-500/10 text-emerald-700 dark:text-emerald-400">
					<CircleCheck class="size-4 text-emerald-600 dark:text-emerald-400" />
					<Alert.Title>Kata Sandi Diperbarui</Alert.Title>
					<Alert.Description class="text-foreground/90">
						Kata sandi Anda telah berhasil diubah. Silakan masuk menggunakan kata sandi baru Anda.
					</Alert.Description>
				</Alert.Root>
			{/if}

			<!-- Alert Error Global dari Server (401 / 403 / 500) -->
			{#if form?.message}
				<Alert.Root variant="destructive" class="mb-4">
					<CircleAlert class="size-4" />
					<Alert.Title>Gagal Masuk</Alert.Title>
					<Alert.Description class="space-y-2">
						<p>{form.message}</p>
						{#if form.message.toLowerCase().includes('not verified') || form.message.toLowerCase().includes('verifikasi')}
							<p class="pt-1">
								<a
									href="/resend-verification"
									class="text-xs font-semibold underline underline-offset-4 hover:opacity-80"
								>
									Kirim ulang email verifikasi &rarr;
								</a>
							</p>
						{/if}
					</Alert.Description>
				</Alert.Root>
			{/if}

			<form
				method="POST"
				use:enhance={() => {
					isSubmitting = true;
					return async ({ update }) => {
						await update();
						isSubmitting = false;
					};
				}}
				novalidate
			>
				<!-- Semantic <fieldset>: Menonaktifkan seluruh form secara serentak saat loading -->
				<fieldset disabled={isSubmitting} class="space-y-4">
					<legend class="sr-only">Formulir Masuk Pengguna</legend>

					<!-- Email Field -->
					<div class="space-y-2">
						<Label for="email">Email</Label>
						<Input
							id="email"
							name="email"
							type="email"
							autocomplete="email"
							placeholder="nama@domain.com"
							required
							aria-invalid={form?.errors?.email ? 'true' : undefined}
							aria-describedby={form?.errors?.email ? 'email-error' : undefined}
						/>
						{#if form?.errors?.email}
							<p id="email-error" role="alert" class="text-xs text-destructive">
								{form.errors.email[0]}
							</p>
						{/if}
					</div>

					<!-- Password Field -->
					<div class="space-y-2">
						<div class="flex items-center justify-between">
							<Label for="password">Password</Label>
							<!-- Tautan Lupa Password (Opsional) -->
							<a
								href="/forgot-password"
								class="text-xs text-muted-foreground hover:text-foreground underline-offset-4 hover:underline"
							>
								Lupa password?
							</a>
						</div>
						<Input
							id="password"
							name="password"
							type="password"
							autocomplete="current-password"
							placeholder="••••••••"
							required
							aria-invalid={form?.errors?.password ? 'true' : undefined}
							aria-describedby={form?.errors?.password ? 'password-error' : undefined}
						/>
						{#if form?.errors?.password}
							<p id="password-error" role="alert" class="text-xs text-destructive">
								{form.errors.password[0]}
							</p>
						{/if}
					</div>

					<!-- Tombol Submit -->
					<Button type="submit" class="w-full">
						{isSubmitting ? 'Memverifikasi...' : 'Masuk Sekarang'}
					</Button>
				</fieldset>
			</form>
		</Card.Content>

		<!-- Footer dengan Tautan Registrasi -->
		<Card.Footer class="flex justify-center border-t p-4 text-sm text-muted-foreground">
			<p>
				Belum memiliki akun?
				<a href="/register" class="font-medium text-foreground underline underline-offset-4">
					Daftar sekarang
				</a>
			</p>
		</Card.Footer>
	</Card.Root>
</main>
