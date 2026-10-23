<script lang="ts">
	import { resolve } from '$app/paths';
	import { enhance } from '$app/forms';
	import { page } from '$app/state';
	import logo from '#lib/assets/ws-logo.svg';
	import SignupForm from '#lib/components/signup-form.svelte';
	import LanguageSwitcher from '#lib/components/common/language-switcher.svelte';
	import { m } from '#lib/paraglide/messages.js';
	import type { PageProps } from './$types';

	let { form }: PageProps = $props();
</script>

<svelte:head>
	<title>{m.page_title({ title: m.signup_title() })}</title>
</svelte:head>

<main class="flex min-h-svh flex-col gap-4 p-6 md:p-10">
	<LanguageSwitcher tone="plain" returnTo={page.url.pathname + page.url.search} class="self-end" />
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
