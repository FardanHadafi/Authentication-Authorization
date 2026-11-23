<script lang="ts">
	import { enhance } from '$app/forms';
	import type { PageProps } from './$types';
	import * as Card from '#lib/components/ui/card/index.js';
	import * as Alert from '#lib/components/ui/alert/index.js';
	import Input from '#lib/components/ui/input/input.svelte';
	import Label from '#lib/components/ui/label/label.svelte';
	import Button from '#lib/components/ui/button/button.svelte';
	import CircleAlert from '@lucide/svelte/icons/circle-alert';
	import KeyRound from '@lucide/svelte/icons/key-round';

	let { data, form }: PageProps = $props();
	let isSubmitting = $state(false);
</script>

<svelte:head>
	<title>Reset Kata Sandi</title>
	<meta name="description" content="Halaman pembuatan kata sandi baru akun" />
</svelte:head>

<main class="flex min-h-screen items-center justify-center p-4">
	<Card.Root class="w-full max-w-md shadow-lg">
		<Card.Header class="space-y-1 text-center">
			<div class="mx-auto mb-2 flex size-12 items-center justify-center rounded-full bg-primary/10 text-primary">
				<KeyRound class="size-6" />
			</div>
			<h1 class="text-2xl font-bold tracking-tight">Buat Kata Sandi Baru</h1>
			<Card.Description>
				Silakan masukkan kata sandi baru yang aman untuk akun Anda
			</Card.Description>
		</Card.Header>

		<Card.Content>
			<!-- Alert Error Global dari Server (400 Token Expired / 500) -->
			{#if form?.message}
				<Alert.Root variant="destructive" class="mb-4">
					<CircleAlert class="size-4" />
					<Alert.Title>Gagal Reset Kata Sandi</Alert.Title>
					<Alert.Description>{form.message}</Alert.Description>
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
				<!-- Token tersembunyi yang diteruskan dari URL -->
				<input type="hidden" name="token" value={form?.token ?? data.token} />

				<fieldset disabled={isSubmitting} class="space-y-4">
					<legend class="sr-only">Formulir Reset Kata Sandi</legend>

					<!-- Password Baru Field -->
					<div class="space-y-2">
						<Label for="new_password">Kata Sandi Baru</Label>
						<Input
							id="new_password"
							name="new_password"
							type="password"
							autocomplete="new-password"
							placeholder="Minimal 8 karakter"
							required
							aria-invalid={form?.errors?.new_password ? 'true' : undefined}
							aria-describedby={form?.errors?.new_password ? 'new_password-error' : undefined}
						/>
						{#if form?.errors?.new_password}
							<p id="new_password-error" role="alert" class="text-xs text-destructive">
								{form.errors.new_password[0]}
							</p>
						{/if}
					</div>

					<!-- Konfirmasi Password Baru Field -->
					<div class="space-y-2">
						<Label for="confirm_new_password">Konfirmasi Kata Sandi Baru</Label>
						<Input
							id="confirm_new_password"
							name="confirm_new_password"
							type="password"
							autocomplete="new-password"
							placeholder="Ulangi kata sandi baru"
							required
							aria-invalid={form?.errors?.confirm_new_password ? 'true' : undefined}
							aria-describedby={form?.errors?.confirm_new_password ? 'confirm_new_password-error' : undefined}
						/>
						{#if form?.errors?.confirm_new_password}
							<p id="confirm_new_password-error" role="alert" class="text-xs text-destructive">
								{form.errors.confirm_new_password[0]}
							</p>
						{/if}
					</div>

					<!-- Tombol Simpan -->
					<Button type="submit" class="w-full">
						{isSubmitting ? 'Menyimpan Kata Sandi...' : 'Perbarui Kata Sandi'}
					</Button>
				</fieldset>
			</form>
		</Card.Content>

		<!-- Footer Link -->
		<Card.Footer class="flex justify-center border-t p-4 text-sm text-muted-foreground">
			<a href="/login" class="font-medium text-foreground underline underline-offset-4">
				Batal & Kembali ke Halaman Masuk
			</a>
		</Card.Footer>
	</Card.Root>
</main>
