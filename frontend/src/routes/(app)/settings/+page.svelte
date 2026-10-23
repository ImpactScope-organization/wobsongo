<script lang="ts">
	import { enhance } from '$app/forms';
	import SettingsView from '#lib/components/settings/settings-view.svelte';
	import { m } from '#lib/paraglide/messages.js';
	import type { PageProps } from './$types';

	let { data, form }: PageProps = $props();
</script>

<svelte:head>
	<title>{m.page_title({ title: m.settings_title() })}</title>
</svelte:head>

<SettingsView
	profile={data.profile}
	passwordChangedAt={data.passwordChangedAt}
	now={data.now}
	profileResult={form?.form === 'profile'
		? { values: form.values, error: form.error, saved: form.saved }
		: undefined}
	passwordResult={form?.form === 'password'
		? { error: form.error, updated: form.updated, resetSentTo: form.resetSentTo }
		: undefined}
	{enhance}
/>
