<script lang="ts">
	import type { ReviewStatus } from '#lib/domain/narrative.js';
	import { m } from '#lib/paraglide/messages.js';
	import { cn } from '#lib/utils.js';
	import { REVIEW_STATUS_LABEL } from './labels';

	interface Props {
		status: ReviewStatus;
		class?: string;
	}

	let { status, class: className }: Props = $props();

	const TONE: Record<ReviewStatus, string> = {
		'needs-check': 'border-signal-soft bg-card text-signal',
		'auto-checked':
			'border-verdict-unverified bg-verdict-unverified text-verdict-unverified-foreground',
		'expert-verified': 'border-status-verified bg-status-verified text-status-verified-foreground',
		'not-in-kb': 'border-verdict-false bg-verdict-false text-verdict-false-foreground'
	};
</script>

<span
	data-status={status}
	class={cn(
		'inline-flex items-center justify-center rounded-full border px-2.5 py-1 text-[0.71875rem] font-medium whitespace-nowrap',
		TONE[status],
		className
	)}
>
	<span class="sr-only">{m.common_review_status_prefix()}</span>
	{REVIEW_STATUS_LABEL[status]}
</span>
