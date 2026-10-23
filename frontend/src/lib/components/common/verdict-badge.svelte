<script lang="ts">
	import type { Verdict } from '#lib/domain/narrative.js';
	import { m } from '#lib/paraglide/messages.js';
	import { cn } from '#lib/utils.js';
	import { VERDICT_LABEL } from './labels';

	interface Props {
		verdict: Verdict;
		class?: string;
	}

	let { verdict, class: className }: Props = $props();

	const TONE: Record<Verdict, string> = {
		false: 'bg-verdict-false text-verdict-false-foreground',
		misleading: 'bg-verdict-misleading text-verdict-misleading-foreground',
		unverified: 'bg-verdict-unverified text-verdict-unverified-foreground'
	};
</script>

<span
	data-verdict={verdict}
	class={cn(
		'inline-flex items-center rounded px-1.5 py-0.5 font-mono text-[0.65625rem] font-semibold tracking-[0.03em] whitespace-nowrap uppercase',
		TONE[verdict],
		className
	)}
>
	<span class="sr-only">{m.common_verdict_prefix()}</span>
	{VERDICT_LABEL[verdict]}
</span>
