<script lang="ts">
	import { resolve } from '$app/paths';
</script>

<svelte:head>
	<title>WobSongo</title>
</svelte:head>

<main class="flex min-h-svh flex-col items-center justify-center gap-3 p-6 text-center">
	<h1 class="text-2xl font-bold">WobSongo</h1>
	<p class="text-sm text-muted-foreground">Infodemic dashboard</p>
	<a href={resolve('/login')} class="text-sm font-semibold underline underline-offset-4">Log in</a>
</main>
