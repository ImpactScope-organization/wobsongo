<script lang="ts">
	import type { HarmAlert } from '#lib/domain/high-harm.js';
	import type { FormEnhance } from '#lib/utils/enhance.js';
	import { cn } from '#lib/utils.js';
	import HarmAlertCard from './harm-alert-card.svelte';

	interface Props {
		/** Most urgent first; the server decides the order. */
		alerts: readonly HarmAlert[];
		now: Date;
		/** Form action URL for every alert's review form, e.g. "?/review". */
		action: string;
		/** Editor page behind each alert's "Override…". */
		overrideHrefFor: (alertId: string) => string;
		enhance?: FormEnhance;
		label?: string;
		emptyText?: string;
		class?: string;
	}

	let {
		alerts,
		now,
		action,
		overrideHrefFor,
		enhance,
		label = 'Dangerous practice alerts',
		emptyText = 'No dangerous-practice alerts.',
		class: className
	}: Props = $props();
</script>

{#if alerts.length === 0}
	<p class={cn('text-[0.8125rem] text-muted-foreground', className)}>{emptyText}</p>
{:else}
	<ol aria-label={label} class={cn('flex flex-col gap-3', className)}>
		{#each alerts as alert (alert.id)}
			<li>
				<HarmAlertCard {alert} {now} {action} overrideHref={overrideHrefFor(alert.id)} {enhance} />
			</li>
		{/each}
	</ol>
{/if}
