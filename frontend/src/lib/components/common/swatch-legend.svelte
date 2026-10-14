<script lang="ts" module>
	export interface LegendItem {
		key: string;
		label: string;
		/** Swatch fill class, e.g. "bg-platform-whatsapp". */
		class: string;
		/** Optional second line, e.g. what the category means. */
		description?: string;
	}
</script>

<script lang="ts">
	import { cn } from '#lib/utils.js';

	interface Props {
		items: readonly LegendItem[];
		/**
		 * Hidden from screen readers by default, because the chart it explains
		 * carries its own text alternative. Set false when the legend adds meaning,
		 * e.g. descriptions.
		 */
		decorative?: boolean;
		class?: string;
	}

	let { items, decorative = true, class: className }: Props = $props();
</script>

<ul
	aria-hidden={decorative || undefined}
	class={cn(
		'flex flex-wrap gap-x-3 gap-y-1.5 text-[0.71875rem] font-medium text-muted-foreground',
		className
	)}
>
	{#each items as item (item.key)}
		<li class="flex items-start gap-1.5">
			<span
				class={cn('mt-[0.1875rem] size-2.5 shrink-0 rounded-[2px]', item.class)}
				aria-hidden="true"
			></span>
			{#if item.description}
				<span>
					<span class="block text-[0.8125rem] font-semibold text-foreground">{item.label}</span>
					<span class="block text-xs/[1.4] font-normal">{item.description}</span>
				</span>
			{:else}
				{item.label}
			{/if}
		</li>
	{/each}
</ul>
