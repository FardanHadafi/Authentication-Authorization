<script lang="ts">
	import type { PageData } from './$types';
	import * as Card from '#lib/components/ui/card/index.ts';
	import { Button } from '#lib/components/ui/button/index.ts';
	import Shield from '@lucide/svelte/icons/shield';
	import ArrowLeft from '@lucide/svelte/icons/arrow-left';
	import Users from '@lucide/svelte/icons/users';

	let { data }: { data: PageData } = $props();
</script>

<svelte:head>
	<title>Panel Administrator - Kelola Pengguna</title>
	<meta name="description" content="Halaman manajemen pengguna khusus administrator" />
</svelte:head>

<main class="min-h-screen bg-background p-4 sm:p-6 lg:p-8">
	<div class="mx-auto max-w-5xl space-y-6">
		<!-- Header Panel -->
		<div class="flex flex-col gap-4 sm:flex-row sm:items-center sm:justify-between">
			<div class="space-y-1">
				<div class="inline-flex items-center gap-1.5 rounded-md bg-destructive/10 px-2.5 py-1 text-xs font-semibold text-destructive">
					<Shield class="size-3.5" />
					Area Terbatas (RBAC: Admin Only)
				</div>
				<h1 class="text-2xl font-bold tracking-tight">Manajemen Pengguna</h1>
				<p class="text-sm text-muted-foreground">
					Daftar seluruh akun yang terdaftar dalam sistem autentikasi & otorisasi.
				</p>
			</div>

			<Button href="/dashboard" variant="outline" size="sm" class="w-fit">
				<ArrowLeft class="mr-2 size-4" />
				Kembali ke Dashboard
			</Button>
		</div>

		<!-- Tabel / Kartu Daftar Pengguna -->
		<Card.Root>
			<Card.Header class="pb-3">
				<div class="flex items-center justify-between">
					<div class="flex items-center gap-2">
						<Users class="size-5 text-muted-foreground" />
						<Card.Title class="text-lg">Daftar Pengguna</Card.Title>
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
									<td colspan="4" class="p-6 text-center text-sm text-muted-foreground">
										Belum ada data pengguna yang terdaftar.
									</td>
								</tr>
							{/if}
						</tbody>
					</table>
				</div>
			</Card.Content>
		</Card.Root>
	</div>
</main>
