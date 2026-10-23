<script lang="ts">
	import { page } from '$app/state';
	import { resolve } from '$app/paths';
	import ErrorNotice from '#lib/components/common/error-notice.svelte';
	import { m } from '#lib/paraglide/messages.js';
</script>

<svelte:head>
	<title>{m.page_title({ title: m.error_status({ status: page.status }) })}</title>
</svelte:head>

<!-- Outside the app shell, e.g. on the login page; inside it, (app)/+error.svelte. -->
<ErrorNotice
	status={page.status}
	message={page.error?.message ?? m.error_unknown()}
	backHref={resolve('/(app)')}
	backLabel={m.back_to_app()}
/>
