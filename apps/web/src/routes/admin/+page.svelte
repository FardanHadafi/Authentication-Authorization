<script lang="ts">
	import type { PageData } from './$types';
	import * as Card from '#lib/components/ui/card/index.ts';
	import { Button } from '#lib/components/ui/button/index.ts';
	import Shield from '@lucide/svelte/icons/shield';
	import ArrowLeft from '@lucide/svelte/icons/arrow-left';
	import Users from '@lucide/svelte/icons/users';
	import FileText from '@lucide/svelte/icons/file-text';
	import CircleCheck from '@lucide/svelte/icons/circle-check';
	import CircleAlert from '@lucide/svelte/icons/circle-alert';

	let { data }: { data: PageData } = $props();

	let activeTab = $state<'users' | 'audit'>('users');
</script>

<svelte:head>
	<title>Panel Administrator - Kelola Pengguna & Audit Logs</title>
	<meta name="description" content="Halaman manajemen pengguna dan audit trail kepatuhan SOC 2" />
</svelte:head>

<main class="min-h-screen bg-background p-4 sm:p-6 lg:p-8">
	<div class="mx-auto max-w-6xl space-y-6">
		<!-- Header Panel -->
		<div class="flex flex-col gap-4 sm:flex-row sm:items-center sm:justify-between">
			<div class="space-y-1">
				<div class="inline-flex items-center gap-1.5 rounded-md bg-destructive/10 px-2.5 py-1 text-xs font-semibold text-destructive">
					<Shield class="size-3.5" />
					Area Terbatas (Granular PBAC: Admin Role)
				</div>
				<h1 class="text-2xl font-bold tracking-tight">Panel Administrasi Sistem</h1>
				<p class="text-sm text-muted-foreground">
					Manajemen pengguna terdaftar dan inspeksi catatan audit kepatuhan SOC 2 / GDPR secara real-time.
				</p>
			</div>

			<Button href="/dashboard" variant="outline" size="sm" class="w-fit">
				<ArrowLeft class="mr-2 size-4" />
				Kembali ke Dashboard
			</Button>
		</div>

		<!-- Tab Navigation Buttons -->
		<div class="flex border-b border-border gap-2">
			<button
				type="button"
				class="inline-flex items-center gap-2 border-b-2 px-4 py-2.5 text-sm font-semibold transition-colors {activeTab === 'users'
					? 'border-primary text-primary'
					: 'border-transparent text-muted-foreground hover:text-foreground'}"
				onclick={() => (activeTab = 'users')}
			>
				<Users class="size-4" />
				Daftar Pengguna ({data.users?.length ?? 0})
			</button>

			<button
				type="button"
				class="inline-flex items-center gap-2 border-b-2 px-4 py-2.5 text-sm font-semibold transition-colors {activeTab === 'audit'
					? 'border-primary text-primary'
					: 'border-transparent text-muted-foreground hover:text-foreground'}"
				onclick={() => (activeTab = 'audit')}
			>
				<FileText class="size-4" />
				Security Audit Trail (SOC 2 / GDPR) ({data.totalAuditLogs ?? 0})
			</button>
		</div>

		{#if activeTab === 'users'}
			<!-- TAB 1: DAFTAR PENGGUNA -->
			<Card.Root>
				<Card.Header class="pb-3">
					<div class="flex items-center justify-between">
						<div class="flex items-center gap-2">
							<Users class="size-5 text-muted-foreground" />
							<Card.Title class="text-lg">Pengguna Terdaftar</Card.Title>
						</div>
						<span class="rounded-full bg-muted px-2.5 py-0.5 text-xs font-medium text-muted-foreground">
							Total: {data.users?.length ?? 0}
						</span>
					</div>
				</Card.Header>

				<Card.Content>
					<div class="relative overflow-x-auto rounded-lg border">
						<table class="w-full text-left text-sm">
							<thead class="bg-muted/50 text-xs uppercase text-muted-foreground">
								<tr>
									<th scope="col" class="px-4 py-3">Pengguna</th>
									<th scope="col" class="px-4 py-3">Role</th>
									<th scope="col" class="px-4 py-3">2FA (MFA)</th>
									<th scope="col" class="px-4 py-3">Token Version</th>
									<th scope="col" class="px-4 py-3">Status</th>
									<th scope="col" class="px-4 py-3">Terdaftar Sejak</th>
								</tr>
							</thead>
							<tbody class="divide-y divide-border">
								{#if data.users && data.users.length > 0}
									{#each data.users as user (user.id)}
										<tr class="hover:bg-muted/30 transition-colors">
											<td class="px-4 py-3">
												<div class="font-medium text-foreground">{user.full_name || 'Tanpa Nama'}</div>
												<div class="text-xs text-muted-foreground font-mono">{user.email}</div>
											</td>
											<td class="px-4 py-3">
												<span
													class="inline-flex items-center rounded-full px-2 py-0.5 text-xs font-semibold capitalize {user.role === 'admin'
														? 'bg-destructive/15 text-destructive border border-destructive/20'
														: 'bg-primary/10 text-primary border border-primary/20'}"
												>
													{user.role}
												</span>
											</td>
											<td class="px-4 py-3">
												{#if user.mfa_enabled}
													<span class="inline-flex items-center rounded-full bg-emerald-500/10 px-2 py-0.5 text-xs font-medium text-emerald-600 dark:text-emerald-400 border border-emerald-500/20">
														Aktif
													</span>
												{:else}
													<span class="inline-flex items-center rounded-full bg-muted px-2 py-0.5 text-xs font-medium text-muted-foreground">
														Tidak
													</span>
												{/if}
											</td>
											<td class="px-4 py-3 font-mono text-xs font-bold text-muted-foreground">
												v{user.token_version ?? 1}
											</td>
											<td class="px-4 py-3">
												<span
													class="inline-flex items-center rounded-full px-2 py-0.5 text-xs font-medium capitalize {user.status === 'active'
														? 'bg-emerald-500/10 text-emerald-600 dark:text-emerald-400 border border-emerald-500/20'
														: user.status === 'pending'
															? 'bg-amber-500/10 text-amber-600 dark:text-amber-400 border border-amber-500/20'
															: 'bg-muted text-muted-foreground border border-border'}"
												>
													{user.status}
												</span>
											</td>
											<td class="px-4 py-3 text-xs text-muted-foreground">
												{new Date(user.created_at).toLocaleDateString('id-ID', {
													day: 'numeric',
													month: 'short',
													year: 'numeric'
												})}
											</td>
										</tr>
									{/each}
								{:else}
									<tr>
										<td colspan="6" class="p-6 text-center text-sm text-muted-foreground">
											Belum ada data pengguna yang terdaftar.
										</td>
									</tr>
								{/if}
							</tbody>
						</table>
					</div>
				</Card.Content>
			</Card.Root>
		{:else}
			<!-- TAB 2: AUDIT LOGS KEPATUHAN (SOC 2 COMPLIANCE) -->
			<Card.Root>
				<Card.Header class="pb-3">
					<div class="flex items-center justify-between">
						<div class="flex items-center gap-2">
							<FileText class="size-5 text-muted-foreground" />
							<div>
								<Card.Title class="text-lg">Catatan Audit Keamanan (Tamper-Evident Security Log)</Card.Title>
								<Card.Description class="text-xs">
									Catatan peristiwa mutasi keamanan permanen di database yang diproses secara asinkron tanpa membebani latency.
								</Card.Description>
							</div>
						</div>
						<span class="rounded-full bg-muted px-2.5 py-0.5 text-xs font-medium text-muted-foreground">
							Total Log: {data.totalAuditLogs ?? 0}
						</span>
					</div>
				</Card.Header>

				<Card.Content>
					<div class="relative overflow-x-auto rounded-lg border">
						<table class="w-full text-left text-sm">
							<thead class="bg-muted/50 text-xs uppercase text-muted-foreground">
								<tr>
									<th scope="col" class="px-3 py-2.5">Waktu</th>
									<th scope="col" class="px-3 py-2.5">Aktor</th>
									<th scope="col" class="px-3 py-2.5">Aksi (Security Event)</th>
									<th scope="col" class="px-3 py-2.5">Resource</th>
									<th scope="col" class="px-3 py-2.5">IP Address</th>
									<th scope="col" class="px-3 py-2.5">Status</th>
								</tr>
							</thead>
							<tbody class="divide-y divide-border">
								{#if data.auditLogs && data.auditLogs.length > 0}
									{#each data.auditLogs as log (log.id)}
										<tr class="hover:bg-muted/30 transition-colors">
											<td class="px-3 py-2.5 text-xs text-muted-foreground whitespace-nowrap">
												{new Date(log.created_at).toLocaleString('id-ID', {
													day: '2-digit',
													month: 'short',
													hour: '2-digit',
													minute: '2-digit',
													second: '2-digit'
												})}
											</td>
											<td class="px-3 py-2.5">
												<div class="font-medium text-xs text-foreground truncate max-w-44">{log.actor_email || 'Sistem / Anonim'}</div>
												{#if log.actor_role}
													<span class="text-[10px] text-muted-foreground uppercase">{log.actor_role}</span>
												{/if}
											</td>
											<td class="px-3 py-2.5 font-mono text-xs font-semibold text-foreground">
												{log.action}
											</td>
											<td class="px-3 py-2.5 text-xs text-muted-foreground capitalize">
												{log.resource_type}
											</td>
											<td class="px-3 py-2.5 font-mono text-xs text-muted-foreground">
												{log.ip_address || '-'}
											</td>
											<td class="px-3 py-2.5">
												{#if log.status === 'success'}
													<span class="inline-flex items-center gap-1 rounded-full bg-emerald-500/10 px-2 py-0.5 text-[11px] font-semibold text-emerald-600 dark:text-emerald-400 border border-emerald-500/20">
														<CircleCheck class="size-3" />
														Success
													</span>
												{:else}
													<span class="inline-flex items-center gap-1 rounded-full bg-destructive/10 px-2 py-0.5 text-[11px] font-semibold text-destructive border border-destructive/20">
														<CircleAlert class="size-3" />
														Failed
													</span>
												{/if}
											</td>
										</tr>
									{/each}
								{:else}
									<tr>
										<td colspan="6" class="p-6 text-center text-sm text-muted-foreground">
											Belum ada catatan audit log yang terekam.
										</td>
									</tr>
								{/if}
							</tbody>
						</table>
					</div>
				</Card.Content>
			</Card.Root>
		{/if}
	</div>
</main>
