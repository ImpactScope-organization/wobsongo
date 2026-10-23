<script lang="ts">
	import TriangleAlertIcon from '@lucide/svelte/icons/triangle-alert';
	import { m } from '#lib/paraglide/messages.js';
	import * as Empty from '../ui/empty/index.js';

	interface Props {
		/** HTTP status, e.g. 404. */
		status: number;
		/** From the server; safe to show. */
		message: string;
		/** Where to go instead. */
		backHref: string;
		backLabel?: string;
	}

	let { status, message, backHref, backLabel = m.back_to_overview() }: Props = $props();

	const title = $derived(
		status === 404
			? m.error_title_not_found()
			: status === 501
				? m.error_title_not_available()
				: status >= 500
					? m.error_title_server()
					: m.error_title_other()
	);
</script>

<main class="mx-auto flex max-w-[85rem] p-[clamp(1rem,3vw,2rem)]">
	<Empty.Root class="min-h-[50svh]">
		<Empty.Header>
			<Empty.Media variant="icon"><TriangleAlertIcon aria-hidden="true" /></Empty.Media>
			<Empty.Title><h1>{title}</h1></Empty.Title>
			<Empty.Description>
				{message}
				<span class="mt-1 block font-mono text-xs">{m.error_status({ status })}</span>
			</Empty.Description>
		</Empty.Header>
		<Empty.Content>
			<a href={backHref} class="text-sm font-semibold underline underline-offset-4">{backLabel}</a>
		</Empty.Content>
	</Empty.Root>
</main>
