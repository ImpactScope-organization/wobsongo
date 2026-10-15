<script lang="ts">
	import { page } from '$app/state';
	import { resolve } from '$app/paths';
	import ErrorNotice from '#lib/components/common/error-notice.svelte';
</script>

<svelte:head>
	<title>Error {page.status} · WobSongo</title>
</svelte:head>

<ErrorNotice
	status={page.status}
	message={page.error?.message ?? 'Unknown error'}
	backHref={resolve('/(app)')}
/>
