<script lang="ts">
  import type { PageData } from './$types';
  import * as Card from '#lib/components/ui/card/index.ts';
  import { Button } from '#lib/components/ui/button/index.ts';

  let { data }: { data: PageData } = $props();
</script>

<svelte:head>
  <title>Dashboard - Antigravity</title>
</svelte:head>

<main class="flex min-h-screen flex-col items-center justify-center bg-background p-4 sm:p-6 lg:p-8">
  <div class="w-full max-w-lg space-y-6">
    <Card.Root>
      <Card.Header>
        <div class="inline-flex items-center gap-2 rounded-md bg-emerald-500/10 px-2.5 py-1 text-xs font-medium text-emerald-600 dark:text-emerald-400 w-fit">
          <span class="size-2 rounded-full bg-emerald-500"></span>
          Autentikasi Berhasil
        </div>
        <Card.Title class="text-2xl font-bold tracking-tight mt-2">
          Selamat Datang di Dashboard!
        </Card.Title>
        <Card.Description>
          Anda berhasil login. Sesi Anda diamankan dengan JWT httpOnly cookies dan Refresh Token Rotation.
        </Card.Description>
      </Card.Header>

      <Card.Content class="space-y-4">
        {#if data.user}
          <div class="rounded-lg border bg-muted/40 p-4 space-y-2.5 text-sm">
            <div class="flex justify-between items-center py-1 border-b border-border/50">
              <span class="text-muted-foreground">User ID</span>
              <span class="font-mono text-xs font-medium text-foreground truncate max-w-55">{data.user.id}</span>
            </div>
            <div class="flex justify-between items-center py-1 border-b border-border/50">
              <span class="text-muted-foreground">Email</span>
              <span class="font-medium text-foreground">{data.user.email}</span>
            </div>
            <div class="flex justify-between items-center py-1">
              <span class="text-muted-foreground">Role</span>
              <span class="inline-flex items-center rounded-full bg-primary/10 px-2 py-0.5 text-xs font-semibold text-primary capitalize">
                {data.user.role}
              </span>
            </div>
          </div>
        {/if}
      </Card.Content>

      <Card.Footer class="flex justify-end gap-3 pt-2">
        <form action="?/logout" method="POST">
          <Button type="submit" variant="destructive" size="sm">
            Logout
          </Button>
        </form>
      </Card.Footer>
    </Card.Root>
  </div>
</main>
