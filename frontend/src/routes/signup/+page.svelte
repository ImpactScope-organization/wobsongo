<script lang="ts">
	import { resolve } from '$app/paths';
	import { enhance } from '$app/forms';
	import logo from '#lib/assets/ws-logo.svg';
	import SignupForm from '#lib/components/signup-form.svelte';
	import type { PageProps } from './$types';

	let { form }: PageProps = $props();
</script>

<svelte:head>
	<title>Sign up · WobSongo</title>
</svelte:head>

<main class="flex min-h-svh flex-col gap-4 p-6 md:p-10">
	<div class="flex flex-1 flex-col items-center justify-center">
		<a
			href={resolve('/(app)')}
			class="mb-8 flex max-w-48 items-center justify-center self-center font-semibold"
		>
			<img src={logo} alt="" class="h-full w-full" />
		</a>
		<div class="w-full max-w-xs">
			<SignupForm values={form?.values} error={form?.message} {enhance} />
		</div>
	</div>
</main>
