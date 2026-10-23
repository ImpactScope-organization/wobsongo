<script lang="ts">
	import { resolve } from '$app/paths';
	import ComingSoon from '#lib/components/common/coming-soon.svelte';
	import { m } from '#lib/paraglide/messages.js';
	import type { PageProps } from './$types';

	let { data }: PageProps = $props();
</script>

<svelte:head>
	<title>{m.page_title({ title: data.title })}</title>
</svelte:head>

<ComingSoon title={data.title} backHref={resolve('/(app)')} />
