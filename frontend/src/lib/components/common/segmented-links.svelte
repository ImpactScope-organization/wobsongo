<script lang="ts" module>
	export interface LinkOption<V extends string = string> {
		value: V;
		label: string;
		href: string;
	}
</script>

<script lang="ts" generics="T extends string">
	import { cn } from '#lib/utils.js';

	interface Props {
		/** Accessible name for the group, e.g. "Time range". */
		label: string;
		options: readonly LinkOption<T>[];
		current: T;
		/** `segmented` is a joined toggle; `pills` are separate rounded chips. */
		variant?: 'segmented' | 'pills';
		class?: string;
	}

	let { label, options, current, variant = 'segmented', class: className }: Props = $props();
</script>

<!--
	Plain links, so switching works without JavaScript: each option is a URL
	(usually a search param) the page's load function reads.
-->
<nav aria-label={label} class={className}>
	<ul
		class={cn(
			'flex flex-wrap',
			variant === 'segmented' && 'overflow-hidden rounded-md border bg-card',
			variant === 'pills' && 'gap-1.5'
		)}
	>
		{#each options as option (option.value)}
			<li>
				<a
					href={option.href}
					aria-current={option.value === current ? 'true' : undefined}
					data-sveltekit-noscroll
					class={cn(
						'block font-medium whitespace-nowrap transition-colors outline-none focus-visible:ring-2 focus-visible:ring-ring',
						variant === 'segmented' &&
							'px-3 py-[0.4375rem] text-xs text-muted-foreground hover:bg-muted hover:text-foreground aria-[current]:bg-primary aria-[current]:text-primary-foreground',
						variant === 'pills' &&
							'rounded-full border px-2.5 py-1 text-[0.6875rem] text-muted-foreground hover:border-foreground hover:text-foreground aria-[current]:border-primary aria-[current]:bg-primary aria-[current]:text-primary-foreground'
					)}
				>
					{option.label}
				</a>
			</li>
		{/each}
	</ul>
</nav>
