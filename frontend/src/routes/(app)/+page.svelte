<script lang="ts">
	import { page } from '$app/state';
	import PulseView from '#lib/components/pulse/pulse-view.svelte';
	import { withSearchParam } from '#lib/utils/url.js';
	import type { PageProps } from './$types';

	let { data }: PageProps = $props();
</script>

<svelte:head>
	<title>Pulse · WobSongo</title>
</svelte:head>

<PulseView
	snapshot={data.snapshot}
	range={data.range}
	topic={data.topic}
	rangeHref={(range) => withSearchParam(page.url, 'range', range)}
	topicHref={(topic) => withSearchParam(page.url, 'topic', topic)}
/>
