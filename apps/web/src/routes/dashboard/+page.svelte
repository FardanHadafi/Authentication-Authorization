<script lang="ts">
	import { enhance } from '$app/forms';
	import type { PageData, ActionData } from './$types';
	import * as Card from '#lib/components/ui/card/index.ts';
	import * as Alert from '#lib/components/ui/alert/index.ts';
	import { Button } from '#lib/components/ui/button/index.ts';
	import Input from '#lib/components/ui/input/input.svelte';
	import Label from '#lib/components/ui/label/label.svelte';
	import ShieldCheck from '@lucide/svelte/icons/shield-check';
	import ShieldAlert from '@lucide/svelte/icons/shield-alert';
	import Shield from '@lucide/svelte/icons/shield';
	import KeyRound from '@lucide/svelte/icons/key-round';
	import Copy from '@lucide/svelte/icons/copy';
	import Check from '@lucide/svelte/icons/check';
	import CircleCheck from '@lucide/svelte/icons/circle-check';
	import CircleAlert from '@lucide/svelte/icons/circle-alert';

	let { data, form }: { data: PageData; form: ActionData } = $props();

	let showSetupModal = $state(false);
	let showDisableForm = $state(false);
	let copiedSecret = $state(false);
	let isSubmitting = $state(false);

	function copyToClipboard(text: string) {
		navigator.clipboard.writeText(text);
		copiedSecret = true;
		setTimeout(() => {
			copiedSecret = false;
		}, 2000);
	}
</script>

<svelte:head>
	<title>Dashboard - Antigravity</title>
</svelte:head>

<main class="min-h-screen bg-background p-4 sm:p-6 lg:p-8">
	<div class="mx-auto max-w-3xl space-y-6">
		<!-- Notifikasi Pesan Sukses / Error -->
		{#if form?.mfaSuccessMessage}
			<Alert.Root class="border-emerald-500/30 bg-emerald-500/10 text-emerald-700 dark:text-emerald-400">
				<CircleCheck class="size-4 text-emerald-600 dark:text-emerald-400" />
				<Alert.Title>Berhasil</Alert.Title>
				<Alert.Description>{form.mfaSuccessMessage}</Alert.Description>
			</Alert.Root>
		{/if}

		{#if form?.mfaActionError}
			<Alert.Root variant="destructive">
				<CircleAlert class="size-4" />
				<Alert.Title>Terjadi Kesalahan</Alert.Title>
				<Alert.Description>{form.mfaActionError}</Alert.Description>
			</Alert.Root>
		{/if}

		<!-- Kartu Informasi Pengguna & Status Sesi -->
		<Card.Root>
			<Card.Header>
				<div class="flex items-center justify-between">
					<div class="inline-flex items-center gap-2 rounded-md bg-emerald-500/10 px-2.5 py-1 text-xs font-medium text-emerald-600 dark:text-emerald-400">
						<span class="size-2 rounded-full bg-emerald-500"></span>
						Autentikasi Aktif (JWT Stateless)
					</div>
					<form action="?/logout" method="POST">
						<Button type="submit" variant="destructive" size="sm">
							Logout
						</Button>
					</form>
				</div>
				<Card.Title class="text-2xl font-bold tracking-tight mt-2">
					Selamat Datang, {data.user?.full_name || 'Pengguna'}!
				</Card.Title>
				<Card.Description>
					Sesi Anda diamankan dengan JWT Access Token (15m), Refresh Token Rotation, dan Immediate Revocation List.
				</Card.Description>
			</Card.Header>

			<Card.Content class="space-y-4">
				{#if data.user}
					<div class="rounded-lg border bg-muted/40 p-4 space-y-2.5 text-sm">
						{#if data.user.full_name}
							<div class="flex justify-between items-center py-1 border-b border-border/50">
								<span class="text-muted-foreground">Nama Lengkap</span>
								<span class="font-medium text-foreground">{data.user.full_name}</span>
							</div>
						{/if}
						<div class="flex justify-between items-center py-1 border-b border-border/50">
							<span class="text-muted-foreground">User ID</span>
							<span class="font-mono text-xs font-medium text-foreground truncate max-w-64">{data.user.id}</span>
						</div>
						<div class="flex justify-between items-center py-1 border-b border-border/50">
							<span class="text-muted-foreground">Email</span>
							<span class="font-medium text-foreground">{data.user.email}</span>
						</div>
						<div class="flex justify-between items-center py-1 border-b border-border/50">
							<span class="text-muted-foreground">Role Akun</span>
							<span class="inline-flex items-center rounded-full bg-primary/10 px-2.5 py-0.5 text-xs font-semibold text-primary capitalize">
								{data.user.role}
							</span>
						</div>
						{#if data.user.token_version !== undefined}
							<div class="flex justify-between items-center py-1 border-b border-border/50">
								<span class="text-muted-foreground">Token Version (Revocation Counter)</span>
								<span class="font-mono text-xs font-bold text-foreground">v{data.user.token_version}</span>
							</div>
						{/if}
						{#if data.user.status}
							<div class="flex justify-between items-center py-1">
								<span class="text-muted-foreground">Status Akun</span>
								<span class="inline-flex items-center rounded-full bg-emerald-500/10 px-2.5 py-0.5 text-xs font-semibold text-emerald-600 dark:text-emerald-400 capitalize">
									{data.user.status}
								</span>
							</div>
						{/if}
					</div>
				{/if}
			</Card.Content>
		</Card.Root>

		<!-- Kartu Keamanan 2FA (Multi-Factor Authentication / Google Authenticator) -->
		<Card.Root>
			<Card.Header>
				<div class="flex items-center justify-between">
					<div class="flex items-center gap-2">
						<KeyRound class="size-5 text-primary" />
						<Card.Title class="text-lg">Keamanan Akun: Multi-Factor Authentication (2FA)</Card.Title>
					</div>
					{#if data.user?.mfa_enabled}
						<span class="inline-flex items-center gap-1 rounded-full bg-emerald-500/15 px-2.5 py-0.5 text-xs font-semibold text-emerald-600 dark:text-emerald-400 border border-emerald-500/30">
							<ShieldCheck class="size-3.5" />
							2FA Aktif (Google Authenticator)
						</span>
					{:else}
						<span class="inline-flex items-center gap-1 rounded-full bg-amber-500/15 px-2.5 py-0.5 text-xs font-semibold text-amber-600 dark:text-amber-400 border border-amber-500/30">
							<ShieldAlert class="size-3.5" />
							2FA Belum Aktif
						</span>
					{/if}
				</div>
				<Card.Description>
					Lindungi akun Anda dari peretasan kata sandi dengan autentikasi berbasis waktu RFC 6238 TOTP (Google Authenticator).
				</Card.Description>
			</Card.Header>

			<Card.Content class="space-y-4">
				{#if data.user?.mfa_enabled}
					<div class="rounded-lg border border-emerald-500/20 bg-emerald-500/5 p-4 text-sm text-foreground space-y-2">
						<p class="font-medium">Akun Anda telah diamankan dengan Google Authenticator.</p>
						<p class="text-xs text-muted-foreground">
							Setiap kali login, Anda akan diminta memasukkan 6-digit kode verifikasi atau menggunakan salah satu dari 8 emergency recovery codes.
						</p>
					</div>

					{#if !showDisableForm}
						<Button
							variant="outline"
							size="sm"
							class="text-destructive hover:bg-destructive/10"
							onclick={() => (showDisableForm = true)}
						>
							Nonaktifkan 2FA
						</Button>
					{:else}
						<form
							action="?/disableMfa"
							method="POST"
							class="rounded-lg border p-4 space-y-3 bg-muted/30"
							use:enhance={() => {
								isSubmitting = true;
								return async ({ update }) => {
									await update();
									isSubmitting = false;
									showDisableForm = false;
								};
							}}
						>
							<p class="text-sm font-semibold text-destructive">Konfirmasi Penonaktifan 2FA</p>
							<div class="grid grid-cols-1 sm:grid-cols-2 gap-3">
								<div class="space-y-1">
									<Label for="password" class="text-xs">Password Saat Ini</Label>
									<Input id="password" name="password" type="password" placeholder="••••••••" required />
								</div>
								<div class="space-y-1">
									<Label for="code" class="text-xs">Kode TOTP (6-Digit)</Label>
									<Input id="code" name="code" type="text" maxlength={6} placeholder="123456" required />
								</div>
							</div>
							<div class="flex gap-2 justify-end pt-2">
								<Button type="button" variant="ghost" size="sm" onclick={() => (showDisableForm = false)}>
									Batal
								</Button>
								<Button type="submit" variant="destructive" size="sm" disabled={isSubmitting}>
									{isSubmitting ? 'Memproses...' : 'Ya, Nonaktifkan'}
								</Button>
							</div>
						</form>
					{/if}
				{:else}
					<!-- Belum Aktif: Form Setup 2FA -->
					{#if !form?.setupData}
						<form
							action="?/setupMfa"
							method="POST"
							use:enhance={() => {
								isSubmitting = true;
								return async ({ update }) => {
									await update();
									isSubmitting = false;
								};
							}}
						>
							<Button type="submit" size="sm" disabled={isSubmitting}>
								<ShieldCheck class="mr-2 size-4" />
								{isSubmitting ? 'Membuat Secret...' : 'Aktifkan 2FA Sekarang'}
							</Button>
						</form>
					{:else}
						<!-- Langkah Setup MFA Terbuka -->
						<div class="rounded-lg border p-4 sm:p-5 space-y-5 bg-muted/30">
							<!-- 1. SCAN QR CODE -->
							<div class="space-y-3">
								<p class="text-sm font-semibold text-foreground">1. Pindai (Scan) QR Code Menggunakan Google Authenticator</p>
								<p class="text-xs text-muted-foreground">
									Buka aplikasi <strong>Google Authenticator</strong> (atau Microsoft Authenticator / Apple Keychain) di ponsel Anda, ketuk tombol <strong>+</strong> lalu pilih <strong>Pindai kode QR (Scan a QR code)</strong>:
								</p>

								<!-- Box Gambar QR Code -->
								<div class="flex flex-col items-center justify-center p-4 bg-white rounded-xl border shadow-sm w-fit mx-auto sm:mx-0">
									{#if form.setupData.qrCodeDataUrl}
										<img
											src={form.setupData.qrCodeDataUrl}
											alt="QR Code Google Authenticator"
											class="size-52 object-contain"
										/>
									{:else}
										<div class="size-52 flex items-center justify-center text-xs text-muted-foreground">
											Memuat QR Code...
										</div>
									{/if}
									<span class="text-[11px] font-medium text-neutral-500 mt-2">
										Arahkan kamera ponsel Anda ke QR code di atas
									</span>
								</div>

								<!-- Alternatif Manual Key -->
								<details class="text-xs text-muted-foreground pt-1">
									<summary class="cursor-pointer hover:text-foreground font-medium select-none">
										Tidak bisa memindai QR code? Masukkan key secara manual &rarr;
									</summary>
									<div class="mt-2 p-3 rounded-lg bg-muted/60 border space-y-2">
										<p class="text-[11px]">Salin Secret Key ini dan tempelkan di aplikasi Authenticator Anda (Type: Time-based):</p>
										<div class="flex items-center gap-2">
											<code class="rounded bg-background px-3 py-1.5 font-mono text-xs sm:text-sm font-bold tracking-wider text-foreground select-all border">
												{form.setupData.secret}
											</code>
											<Button
												type="button"
												variant="outline"
												size="sm"
												onclick={() => copyToClipboard(form?.setupData?.secret || '')}
											>
												{#if copiedSecret}
													<Check class="size-4 text-emerald-500 mr-1" />
													Tersalin!
												{:else}
													<Copy class="size-4 mr-1" />
													Salin
												{/if}
											</Button>
										</div>
									</div>
								</details>
							</div>

							<!-- 8 Recovery Codes -->
							<div class="space-y-1.5 pt-2 border-t border-border/60">
								<p class="text-sm font-semibold text-foreground">2. Simpan 8 Kode Pemulihan Darurat (One-Time Backup Codes)</p>
								<p class="text-xs text-muted-foreground">
									Simpan kode cadangan berikut di tempat aman. Masing-masing kode hanya bisa digunakan satu kali jika ponsel hilang:
								</p>
								<div class="grid grid-cols-2 sm:grid-cols-4 gap-2 pt-2">
									{#each form.setupData.recoveryCodes as code}
										<div class="rounded border bg-background px-2 py-1 text-center font-mono text-xs font-semibold tracking-wider">
											{code}
										</div>
									{/each}
								</div>
							</div>

							<!-- Konfirmasi Kode 6-Digit Pertama -->
							<form
								action="?/enableMfa"
								method="POST"
								class="space-y-3 pt-3 border-t border-border/60"
								use:enhance={() => {
									isSubmitting = true;
									return async ({ update }) => {
										await update();
										isSubmitting = false;
									};
								}}
							>
								<div class="space-y-1.5">
									<Label for="enable_code" class="text-sm font-semibold">
										3. Masukkan 6-Digit Kode Pertama dari Google Authenticator:
									</Label>
									<Input
										id="enable_code"
										name="code"
										type="text"
										maxlength={6}
										placeholder="123456"
										class="font-mono text-center text-lg tracking-widest max-w-48"
										required
										autofocus
									/>
								</div>
								<Button type="submit" size="sm" disabled={isSubmitting}>
									{isSubmitting ? 'Memverifikasi...' : 'Verifikasi & Aktifkan Permanen'}
								</Button>
							</form>
						</div>
					{/if}
				{/if}
			</Card.Content>
		</Card.Root>

		<!-- Kartu Granular PBAC (Permission-Based Access Control) -->
		<Card.Root>
			<Card.Header>
				<div class="flex items-center gap-2">
					<Shield class="size-5 text-primary" />
					<Card.Title class="text-lg">Hak Akses Granular (PBAC Engine)</Card.Title>
				</div>
				<Card.Description>
					Hak istimewa akun Anda dievaluasi secara dinamis oleh middleware PBAC dengan caching berkinerja tinggi di Redis.
				</Card.Description>
			</Card.Header>

			<Card.Content>
				{#if data.permissions && data.permissions.length > 0}
					<div class="flex flex-wrap gap-2">
						{#each data.permissions as perm}
							<span class="inline-flex items-center rounded-md bg-muted px-2.5 py-1 text-xs font-mono font-medium text-foreground border border-border/80">
								✓ {perm}
							</span>
						{/each}
					</div>
				{:else}
					<p class="text-xs text-muted-foreground">Belum ada izin khusus yang terdaftar untuk role ini.</p>
				{/if}
			</Card.Content>
		</Card.Root>

		<!-- Akses Administrator -->
		{#if data.user?.role === 'admin' || data.permissions?.includes('users:read') || data.permissions?.includes('audit_logs:read')}
			<div class="rounded-lg border border-destructive/20 bg-destructive/5 p-4 flex items-center justify-between">
				<div>
					<p class="text-sm font-semibold text-foreground">Akses Panel Administrator</p>
					<p class="text-xs text-muted-foreground">Kelola pengguna, tinjau audit logs kepatuhan SOC 2, dan kelola izin.</p>
				</div>
				<Button href="/admin" variant="default" size="sm">
					Panel Admin &rarr;
				</Button>
			</div>
		{/if}
	</div>
</main>
