<script lang="ts">
	import { resolve } from '$app/paths';
	import ComingSoon from '#lib/components/common/coming-soon.svelte';
	import type { PageProps } from './$types';

	let { data }: PageProps = $props();
</script>

<svelte:head>
	<title>{data.title} · WobSongo</title>
</svelte:head>

<ComingSoon title={data.title} backHref={resolve('/(app)')} />
