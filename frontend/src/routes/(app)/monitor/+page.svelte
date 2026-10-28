<script lang="ts">
	import { enhance } from '$app/forms';
	import { page } from '$app/state';
	import MonitorView from '#lib/components/monitor/monitor-view.svelte';
	import { m } from '#lib/paraglide/messages.js';
	import { withSearchParam } from '#lib/utils/url.js';
	import type { PageProps } from './$types';

	let { data, form }: PageProps = $props();
</script>

<svelte:head>
	<title>{m.page_title({ title: m.nav_monitor() })}</title>
</svelte:head>

<MonitorView
	snapshot={data.snapshot}
	range={data.range}
	selectedRunId={data.selectedRunId}
	rangeHref={(range) => withSearchParam(page.url, 'range', range)}
	runHref={(id) => withSearchParam(page.url, 'run', id)}
	runError={form?.error}
	{enhance}
/>
