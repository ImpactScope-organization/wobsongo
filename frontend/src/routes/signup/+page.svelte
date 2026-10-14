<script lang="ts">
	import { resolve } from '$app/paths';
	import { enhance } from '$app/forms';
	import logo from '#lib/assets/wobsongo-mark.jpg';
	import SignupForm from '#lib/components/signup-form.svelte';
	import type { PageProps } from './$types';

	let { form }: PageProps = $props();
</script>

<svelte:head>
	<title>Sign up · WobSongo</title>
</svelte:head>

<main class="flex min-h-svh flex-col gap-4 p-6 md:p-10">
	<a href={resolve('/')} class="flex items-center gap-2 self-center font-semibold md:self-start">
		<img src={logo} alt="" class="size-7 rounded-md" />
		WobSongo
	</a>
	<div class="flex flex-1 items-center justify-center">
		<div class="w-full max-w-xs">
			<SignupForm values={form?.values} error={form?.message} {enhance} />
		</div>
	</div>
</main>
