<script lang="ts">
	import type { Count } from '#lib/domain/count.js';
	import { formatNumber } from '#lib/utils/format.js';
	import { cn } from '#lib/utils.js';

	interface Props {
		count: Count;
		class?: string;
	}

	let { count, class: className }: Props = $props();
</script>

{#if count.kind === 'value'}
	<span class={cn('font-mono', className)}>{formatNumber(count.value)}</span>
{:else}
	<span
		class={cn('font-mono text-faint-foreground', className)}
		title="Fewer than {count.threshold}: hidden to protect privacy"
	>
		&lt; {count.threshold}<span class="sr-only">, hidden to protect privacy</span>
	</span>
{/if}
