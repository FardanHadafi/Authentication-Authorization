<script lang="ts">
	import { enhance } from '$app/forms';
	import { page } from '$app/state';
	import type { PageProps } from './$types';
	import * as Card from '#lib/components/ui/card/index.js';
	import Input from '#lib/components/ui/input/input.svelte';
	import Label from '#lib/components/ui/label/label.svelte';
	import Button from '#lib/components/ui/button/button.svelte';

	let { form }: PageProps = $props();
	let isSubmitting = $state(false);
	const isVerified = $derived(page.url.searchParams.get('verified') === 'true');
</script>

<svelte:head>
	<title>Daftar Akun Baru</title>
	<meta name="description" content="Halaman pendaftaran pengguna baru" />
</svelte:head>

<!-- Landmark Semantic: <main> -->
<main class="flex min-h-screen items-center justify-center p-4">
	<Card.Root class="w-full max-w-md shadow-lg">
		<Card.Header class="space-y-1 text-center">
			<!-- Semantic H1 -->
			<h1 class="text-2xl font-bold tracking-tight">Buat Akun Baru</h1>
			<Card.Description>Daftar untuk mengakses sistem autentikasi</Card.Description>
		</Card.Header>

		<Card.Content>
			{#if isVerified}
				<section
					aria-live="polite"
					class="rounded-lg bg-emerald-500/10 p-5 text-center text-emerald-600 dark:text-emerald-400"
				>
					<h2 class="text-lg font-semibold">Email Berhasil Diverifikasi!</h2>
					<p class="mt-1 text-sm text-muted-foreground">
						Akun Anda telah aktif dan siap digunakan. Silakan login untuk melanjutkan.
					</p>
					<div class="mt-5">
						<Button href="/login" class="w-full">
							Masuk ke Akun
						</Button>
					</div>
				</section>
			{:else if form?.success}
				<section
					aria-live="polite"
					class="rounded-lg bg-emerald-500/10 p-4 text-center text-emerald-600 dark:text-emerald-400"
				>
					<h2 class="text-base font-semibold">Registrasi Berhasil!</h2>
					<p class="mt-1 text-sm text-muted-foreground">
						Silakan periksa kotak masuk email Anda (buka Mailpit di
						<a
							href="http://localhost:8025"
							target="_blank"
							class="font-medium text-foreground underline"
						>
							localhost:8025
						</a>) untuk memverifikasi akun Anda.
					</p>
				</section>
			{:else}
				{#if form?.message}
					<div
						role="alert"
						class="mb-4 rounded-md bg-destructive/15 p-3 text-sm font-medium text-destructive"
					>
						{form.message}
					</div>
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
						<legend class="sr-only">Formulir Pendaftaran Pengguna</legend>

						<!-- Nama Lengkap -->
						<div class="space-y-2">
							<Label for="full_name">Nama Lengkap</Label>
							<Input
								id="full_name"
								name="full_name"
								type="text"
								autocomplete="name"
								placeholder="Nilou Mirei"
								required
								aria-invalid={form?.errors?.full_name ? 'true' : undefined}
								aria-describedby={form?.errors?.full_name ? 'name-error' : undefined}
							/>
							{#if form?.errors?.full_name}
								<p id="name-error" role="alert" class="text-xs text-destructive">
									{form.errors.full_name[0]}
								</p>
							{/if}
						</div>

						<!-- Email -->
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

						<!-- Password -->
						<div class="space-y-2">
							<Label for="password">Password</Label>
							<Input
								id="password"
								name="password"
								type="password"
								autocomplete="new-password"
								placeholder="Minimal 8 karakter"
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

						<!-- Konfirmasi Password -->
						<div class="space-y-2">
							<Label for="confirm_password">Konfirmasi Password</Label>
							<Input
								id="confirm_password"
								name="confirm_password"
								type="password"
								autocomplete="new-password"
								placeholder="Ketik ulang password"
								required
								aria-invalid={form?.errors?.confirm_password ? 'true' : undefined}
								aria-describedby={form?.errors?.confirm_password ? 'confirm-error' : undefined}
							/>
							{#if form?.errors?.confirm_password}
								<p id="confirm-error" role="alert" class="text-xs text-destructive">
									{form.errors.confirm_password[0]}
								</p>
							{/if}
						</div>

						<Button type="submit" class="w-full">
							{isSubmitting ? 'Memproses...' : 'Daftar Sekarang'}
						</Button>
					</fieldset>
				</form>
			{/if}
		</Card.Content>
	</Card.Root>
</main>
