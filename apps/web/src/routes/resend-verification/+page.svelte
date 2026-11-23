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
	import Mail from '@lucide/svelte/icons/mail';
	import ArrowLeft from '@lucide/svelte/icons/arrow-left';

	let { data, form }: PageProps = $props();
	let isSubmitting = $state(false);
</script>

<svelte:head>
	<title>Kirim Ulang Verifikasi Email</title>
	<meta name="description" content="Halaman pengiriman ulang tautan verifikasi akun email" />
</svelte:head>

<main class="flex min-h-screen items-center justify-center p-4">
	<Card.Root class="w-full max-w-md shadow-lg">
		<Card.Header class="space-y-1 text-center">
			<div class="mx-auto mb-2 flex size-12 items-center justify-center rounded-full bg-primary/10 text-primary">
				<Mail class="size-6" />
			</div>
			<h1 class="text-2xl font-bold tracking-tight">Kirim Ulang Verifikasi</h1>
			<Card.Description>
				Masukkan email terdaftar Anda untuk menerima tautan aktivasi akun yang baru
			</Card.Description>
		</Card.Header>

		<Card.Content>
			<!-- Kondisi Sukses -->
			{#if form?.success}
				<div class="space-y-4">
					<Alert.Root class="border-emerald-500/30 bg-emerald-500/10 text-emerald-700 dark:text-emerald-400">
						<CircleCheck class="size-4 text-emerald-600 dark:text-emerald-400" />
						<Alert.Title>Email Verifikasi Terkirim</Alert.Title>
						<Alert.Description class="text-foreground/90">
							{form.message}
						</Alert.Description>
					</Alert.Root>

					<p class="text-xs text-muted-foreground text-center">
						Tautan verifikasi berlaku selama 24 jam. Jangan lupa memeriksa folder spam jika email belum masuk.
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
						<Alert.Title>Pengiriman Gagal</Alert.Title>
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
						<legend class="sr-only">Formulir Kirim Ulang Verifikasi Email</legend>

						<!-- Email Field -->
						<div class="space-y-2">
							<Label for="email">Alamat Email</Label>
							<Input
								id="email"
								name="email"
								type="email"
								autocomplete="email"
								placeholder="nama@domain.com"
								value={form?.email ?? data.email ?? ''}
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
							{isSubmitting ? 'Mengirim Tautan...' : 'Kirim Tautan Verifikasi Baru'}
						</Button>
					</fieldset>
				</form>
			{/if}
		</Card.Content>

		<!-- Footer Link -->
		{#if !form?.success}
			<Card.Footer class="flex justify-center border-t p-4 text-sm text-muted-foreground">
				<p>
					Sudah berhasil verifikasi?
					<a href="/login" class="font-medium text-foreground underline underline-offset-4">
						Masuk ke akun
					</a>
				</p>
			</Card.Footer>
		{/if}
	</Card.Root>
</main>
