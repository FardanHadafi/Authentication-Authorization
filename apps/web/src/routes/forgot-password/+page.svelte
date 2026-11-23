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
	import ArrowLeft from '@lucide/svelte/icons/arrow-left';

	let { form }: PageProps = $props();
	let isSubmitting = $state(false);
</script>

<svelte:head>
	<title>Lupa Kata Sandi</title>
	<meta name="description" content="Halaman permintaan reset kata sandi pengguna" />
</svelte:head>

<main class="flex min-h-screen items-center justify-center p-4">
	<Card.Root class="w-full max-w-md shadow-lg">
		<Card.Header class="space-y-1 text-center">
			<h1 class="text-2xl font-bold tracking-tight">Lupa Kata Sandi?</h1>
			<Card.Description>
				Masukkan alamat email akun Anda untuk menerima instruksi pemulihan kata sandi
			</Card.Description>
		</Card.Header>

		<Card.Content>
			<!-- Status Sukses: Email Instruksi Telah Dikirim -->
			{#if form?.success}
				<div class="space-y-4">
					<Alert.Root class="border-emerald-500/30 bg-emerald-500/10 text-emerald-700 dark:text-emerald-400">
						<CircleCheck class="size-4 text-emerald-600 dark:text-emerald-400" />
						<Alert.Title>Tautan Terkirim</Alert.Title>
						<Alert.Description class="text-foreground/90">
							{form.message}
						</Alert.Description>
					</Alert.Root>

					<p class="text-xs text-muted-foreground text-center">
						Periksa folder inbox atau spam di email Anda. Tautan berlaku selama 15 menit.
					</p>

					<Button href="/login" class="w-full" variant="outline">
						<ArrowLeft class="mr-2 size-4" />
						Kembali ke Halaman Masuk
					</Button>
				</div>
			{:else}
				<!-- Alert Error Global dari Server -->
				{#if form?.message}
					<Alert.Root variant="destructive" class="mb-4">
						<CircleAlert class="size-4" />
						<Alert.Title>Permintaan Gagal</Alert.Title>
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
					<fieldset disabled={isSubmitting} class="space-y-4">
						<legend class="sr-only">Formulir Lupa Kata Sandi</legend>

						<!-- Email Field -->
						<div class="space-y-2">
							<Label for="email">Alamat Email</Label>
							<Input
								id="email"
								name="email"
								type="email"
								autocomplete="email"
								placeholder="nama@domain.com"
								value={form?.email ?? ''}
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

						<!-- Tombol Submit -->
						<Button type="submit" class="w-full">
							{isSubmitting ? 'Mengirim Instruksi...' : 'Kirim Tautan Reset'}
						</Button>
					</fieldset>
				</form>
			{/if}
		</Card.Content>

		<!-- Footer dengan Tautan Kembali ke Login dan Kirim Ulang Verifikasi -->
		{#if !form?.success}
			<Card.Footer class="flex flex-col gap-2 border-t p-4 text-center text-sm text-muted-foreground">
				<p>
					Sudah ingat kata sandi Anda?
					<a href="/login" class="font-medium text-foreground underline underline-offset-4">
						Masuk sekarang
					</a>
				</p>
				<p class="text-xs">
					Akun belum aktif?
					<a href="/resend-verification" class="font-medium text-foreground underline underline-offset-4">
						Kirim ulang email verifikasi
					</a>
				</p>
			</Card.Footer>
		{/if}
	</Card.Root>
</main>
