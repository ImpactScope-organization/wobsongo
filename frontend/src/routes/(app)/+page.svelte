<script lang="ts">
	import { page } from '$app/state';
	import OverviewView from '#lib/components/overview/overview-view.svelte';
	import { withSearchParam } from '#lib/utils/url.js';
	import type { PageProps } from './$types';

	let { data }: PageProps = $props();
</script>

<svelte:head>
	<title>Overview · WobSongo</title>
</svelte:head>

<OverviewView
	snapshot={data.snapshot}
	range={data.range}
	topic={data.topic}
	rangeHref={(range) => withSearchParam(page.url, 'range', range)}
	topicHref={(topic) => withSearchParam(page.url, 'topic', topic)}
/>
