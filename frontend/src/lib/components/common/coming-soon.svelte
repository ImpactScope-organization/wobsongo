<script lang="ts">
	import HourglassIcon from '@lucide/svelte/icons/hourglass';
	import * as Empty from '../ui/empty/index.js';

	interface Props {
		/** The view that isn't built yet, e.g. "Narratives". */
		title: string;
		/** Where to go instead. */
		backHref: string;
		backLabel?: string;
	}

	let { title, backHref, backLabel = 'Back to Overview' }: Props = $props();
</script>

<main class="mx-auto flex max-w-[85rem] p-[clamp(1rem,3vw,2rem)]">
	<Empty.Root class="min-h-[50svh]">
		<Empty.Header>
			<Empty.Media variant="icon"><HourglassIcon aria-hidden="true" /></Empty.Media>
			<Empty.Title><h1>{title} is coming soon</h1></Empty.Title>
			<Empty.Description>This part of the dashboard isn’t available yet.</Empty.Description>
		</Empty.Header>
		<Empty.Content>
			<a href={backHref} class="text-sm font-semibold underline underline-offset-4">{backLabel}</a>
		</Empty.Content>
	</Empty.Root>
</main>
