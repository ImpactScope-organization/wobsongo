<script lang="ts">
	import type { Count } from '#lib/domain/count.js';
	import { m } from '#lib/paraglide/messages.js';
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
		title={m.common_suppressed_title({ threshold: count.threshold })}
	>
		&lt; {count.threshold}<span class="sr-only">{m.common_suppressed_sr()}</span>
	</span>
{/if}
