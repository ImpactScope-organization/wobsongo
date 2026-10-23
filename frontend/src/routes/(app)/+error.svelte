<script lang="ts">
	import { page } from '$app/state';
	import { resolve } from '$app/paths';
	import ErrorNotice from '#lib/components/common/error-notice.svelte';
	import { m } from '#lib/paraglide/messages.js';
</script>

<svelte:head>
	<title>{m.page_title({ title: m.error_status({ status: page.status }) })}</title>
</svelte:head>

<ErrorNotice
	status={page.status}
	message={page.error?.message ?? m.error_unknown()}
	backHref={resolve('/(app)')}
/>
