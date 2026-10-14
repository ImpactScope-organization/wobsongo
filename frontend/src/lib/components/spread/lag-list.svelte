<script lang="ts">
	import type { NarrativeLag } from '#lib/domain/spread.js';
	import { cn } from '#lib/utils.js';
	import { PLATFORM_SHORT_LABEL } from '../common/labels';
	import { PLATFORM_TONE } from '../common/tones';

	interface Props {
		/** Already sorted for display. */
		lags: readonly NarrativeLag[];
		/** End of the time axis; longer lags fill the track. */
		maxHours?: number;
		hrefFor?: (id: string) => string;
		emptyText?: string;
		class?: string;
	}

	let {
		lags,
		maxHours = 72,
		hrefFor = (id) => `/narratives/${encodeURIComponent(id)}`,
		emptyText = 'No narrative has reached WhatsApp in this window.',
		class: className
	}: Props = $props();

	// A zero or negative axis would put every marker at NaN%.
	const axis = $derived(maxHours > 0 ? maxHours : 1);
	const ticks = $derived(
		[0, 1, 2, 3].map((step) => Math.round(((axis / 3) * step + Number.EPSILON) * 10) / 10)
	);
	const position = (hours: number) => Math.min(Math.max(hours, 0) / axis, 1) * 100;

	const ROW = 'grid grid-cols-[min(44%,18.75rem)_minmax(0,1fr)] items-center gap-3.5';
</script>

{#if lags.length === 0}
	<p class={cn('text-[0.8125rem] text-muted-foreground', className)}>{emptyText}</p>
{:else}
	<div class={className}>
		<div class={cn(ROW, 'mb-1')} aria-hidden="true">
			<span></span>
			<div
				class="flex justify-between font-mono text-[0.65625rem] font-medium text-faint-foreground"
			>
				{#each ticks as tick, index (index)}<span>{tick} h</span>{/each}
			</div>
		</div>

		<ol aria-label="Public to WhatsApp lag by narrative">
			{#each lags as lag (lag.narrativeId)}
				{@const at = position(lag.lagHours)}
				<li class={cn(ROW, 'relative border-t border-rule py-2.5 hover:bg-muted')}>
					<div class="min-w-0">
						<a
							href={hrefFor(lag.narrativeId)}
							class="text-[0.84375rem]/[1.3] font-medium text-pretty outline-none after:absolute after:inset-0 focus-visible:underline"
						>
							{lag.title}
						</a>
						<p
							class="mt-1 flex items-center gap-1.5 text-[0.6875rem] font-medium text-muted-foreground"
						>
							<span class={cn('size-2 rounded-[2px]', PLATFORM_TONE[lag.origin].swatch)}></span>
							First seen on: {PLATFORM_SHORT_LABEL[lag.origin]}
						</p>
					</div>
					<div
						class="relative h-[1.375rem] rounded-[3px] bg-[repeating-linear-gradient(90deg,var(--rule)_0_1px,transparent_1px_33.333%)]"
					>
						<div
							class={cn(
								'absolute top-[0.3125rem] left-0 h-3 rounded-[2px] opacity-85',
								PLATFORM_TONE[lag.origin].swatch
							)}
							style:width="{at}%"
							aria-hidden="true"
						></div>
						<div
							class="absolute top-0 h-full w-[3px] rounded-[1px] bg-platform-whatsapp"
							style:left="{at}%"
							aria-hidden="true"
						></div>
						<span
							data-flip={at > 75 || undefined}
							class="absolute top-[0.1875rem] translate-x-2 font-mono text-xs font-semibold whitespace-nowrap data-flip:-translate-x-[calc(100%+0.5rem)] data-flip:rounded-sm data-flip:bg-card data-flip:px-1"
							style:left="{at}%"
						>
							+{lag.lagHours} h<span class="sr-only">&nbsp;to WhatsApp</span>
						</span>
					</div>
				</li>
			{/each}
		</ol>

		<ul
			class="mt-2.5 flex flex-wrap gap-4 text-[0.71875rem] font-medium text-muted-foreground"
			aria-hidden="true"
		>
			<li class="flex items-center gap-1.5">
				<span class="h-2 w-3.5 rounded-[2px] bg-platform-x"></span>Public platforms only
			</li>
			<li class="flex items-center gap-1.5">
				<span class="h-3 w-[3px] bg-platform-whatsapp"></span>First WhatsApp question
			</li>
		</ul>
	</div>
{/if}
