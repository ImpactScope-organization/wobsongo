<script lang="ts">
	import type { HarmAlert } from '#lib/domain/high-harm.js';
	import type { FormEnhance } from '#lib/utils/enhance.js';
	import { m } from '#lib/paraglide/messages.js';
	import { formatAgo, formatLogTime, formatNumber, plural } from '#lib/utils/format.js';
	import { cn } from '#lib/utils.js';
	import { HARM_RISK_LABEL, PLATFORM_SHORT_LABEL, VERDICT_LABEL } from '../common/labels';
	import ReviewActions from '../common/review-actions.svelte';

	interface Props {
		alert: HarmAlert;
		/** Reference time for "38 min ago"; pass the injected clock's now. */
		now: Date;
		/** Form action URL, e.g. "?/review". Posts `id` and `decision`. */
		action: string;
		/** Editor page behind "Override…", for a new verdict and reply. */
		overrideHref: string;
		enhance?: FormEnhance;
		class?: string;
	}

	let { alert, now, action, overrideHref, enhance, class: className }: Props = $props();

	const headingId = $props.id();
	const similar = $derived.by(() => {
		const count = formatNumber(alert.similarCount);
		return plural(
			alert.similarCount,
			m.high_harm_similar_one({ count }),
			m.high_harm_similar_other({ count })
		);
	});
	const decision = $derived(alert.review.kind === 'decided' ? alert.review.decision : undefined);
</script>

<article
	aria-labelledby={headingId}
	data-risk={alert.risk}
	data-decision={decision}
	class={cn(
		'flex flex-col gap-2.5 rounded-lg border border-rule bg-muted px-4 py-3.5 transition-opacity data-[decision=dismiss]:opacity-55 data-[risk=critical]:border-signal-soft data-[risk=critical]:bg-card',
		className
	)}
>
	<header class="flex flex-wrap items-center gap-2">
		<span
			data-risk={alert.risk}
			class="rounded px-1.5 py-0.5 font-mono text-[0.65625rem] font-semibold tracking-[0.04em] uppercase data-[risk=critical]:bg-destructive data-[risk=critical]:text-white data-[risk=high]:bg-verdict-misleading data-[risk=high]:text-verdict-misleading-foreground"
		>
			<span class="sr-only">{m.high_harm_risk_sr()}</span>
			{HARM_RISK_LABEL[alert.risk]}{alert.riskReviewed ? ` · ${m.high_harm_reviewed()}` : ''}
		</span>
		<h3 id={headingId} class="font-sans text-xs font-medium">{alert.substance}</h3>
		<time
			datetime={alert.detectedAt.toISOString()}
			title={formatLogTime(alert.detectedAt, now)}
			class="ml-auto text-xs text-muted-foreground"
		>
			{formatAgo(alert.detectedAt, now)}
		</time>
	</header>

	<blockquote class="text-[0.9375rem]/normal text-pretty">“{alert.quote}”</blockquote>

	<ul
		class="flex flex-wrap items-center gap-x-4 gap-y-2 text-[0.71875rem] font-medium text-muted-foreground"
	>
		<li>{PLATFORM_SHORT_LABEL[alert.platform]} · {alert.channel} · {alert.language}</li>
		<li>{alert.region}</li>
		<li>{similar}</li>
	</ul>

	<div class="flex flex-col gap-1 rounded-md bg-background px-3 py-2.5">
		<p
			class="font-mono text-[0.65625rem] font-semibold tracking-[0.04em] text-muted-foreground uppercase"
		>
			{m.high_harm_engine_reply({
				verdict: VERDICT_LABEL[alert.engine.verdict],
				confidence: alert.engine.confidence.toFixed(2)
			})}
		</p>
		<p class="text-[0.84375rem]/[1.45] text-pretty text-verdict-unverified-foreground">
			{alert.engine.explanation}
		</p>
	</div>

	<ReviewActions
		itemId={alert.id}
		itemLabel={alert.substance}
		state={alert.review}
		{action}
		{overrideHref}
		{enhance}
	/>
</article>
