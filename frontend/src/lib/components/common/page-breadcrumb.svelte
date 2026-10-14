<script lang="ts" module>
	export interface Crumb {
		label: string;
		/** Omit for crumbs that aren't pages of their own. */
		href?: string;
		/** Render in the mono font, e.g. for ids. */
		mono?: boolean;
	}
</script>

<script lang="ts">
	import * as Breadcrumb from '#lib/components/ui/breadcrumb/index.js';
	import { cn } from '#lib/utils.js';

	interface Props {
		/** Ancestors first; the last crumb is the current page. */
		crumbs: readonly Crumb[];
		class?: string;
	}

	let { crumbs, class: className }: Props = $props();
</script>

<Breadcrumb.Root class={className}>
	<Breadcrumb.List class="gap-2 text-xs font-medium text-muted-foreground sm:gap-2">
		{#each crumbs as crumb, index (index)}
			{@const last = index === crumbs.length - 1}
			<Breadcrumb.Item>
				{#if last}
					<Breadcrumb.Page class={cn('text-muted-foreground', crumb.mono && 'font-mono')}>
						{crumb.label}
					</Breadcrumb.Page>
				{:else if crumb.href}
					<Breadcrumb.Link
						href={crumb.href}
						class={cn('underline underline-offset-[3px]', crumb.mono && 'font-mono')}
					>
						{crumb.label}
					</Breadcrumb.Link>
				{:else}
					<span class={cn(crumb.mono && 'font-mono')}>{crumb.label}</span>
				{/if}
			</Breadcrumb.Item>
			{#if !last}
				<Breadcrumb.Separator>/</Breadcrumb.Separator>
			{/if}
		{/each}
	</Breadcrumb.List>
</Breadcrumb.Root>
