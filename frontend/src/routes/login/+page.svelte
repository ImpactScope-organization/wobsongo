<script lang="ts">
	import { resolve } from '$app/paths';
	import { enhance } from '$app/forms';
	import logo from '#lib/assets/wobsongo-mark.jpg';
	import LoginForm from '#lib/components/login-form.svelte';
	import type { PageProps } from './$types';

	let { form }: PageProps = $props();
</script>

<svelte:head>
	<title>Log in · WobSongo</title>
</svelte:head>

<main class="flex min-h-svh flex-col gap-4 p-6 md:p-10">
	<a href={resolve('/')} class="flex items-center gap-2 self-center font-semibold md:self-start">
		<img src={logo} alt="" class="size-7 rounded-md" />
		WobSongo
	</a>
	<div class="flex flex-1 items-center justify-center">
		<div class="w-full max-w-xs">
			<LoginForm email={form?.email} error={form?.message} {enhance} />
		</div>
	</div>
</main>
