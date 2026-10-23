<script lang="ts">
	import type { ContentCreator } from '#lib/domain/overview.js';
	import { m } from '#lib/paraglide/messages.js';
	import { formatNumber, formatPercent, intlLocale } from '#lib/utils/format.js';
	import { cn } from '#lib/utils.js';
	import PlatformIcon from '../common/platform-icon.svelte';
	import RiskGauge from '../common/risk-gauge.svelte';
	import VerdictMeter from '../common/verdict-meter.svelte';

	interface Props {
		/** Content creators to display. Ranked by highest volume (posts) first. */
		creators: readonly ContentCreator[];
		label?: string;
		emptyText?: string;
		class?: string;
		/**
		 * When true, or when hasAccess is false, privacy mode swaps creator
		 * names with pseudonyms, hides handles, and disables profile links.
		 */
		privacyMode?: boolean;
		/**
		 * Whether the logged in user has access to view creator identities.
		 * Defaults to true. If false, privacy mode is activated.
		 */
		hasAccess?: boolean;
		/**
		 * Link destination for creator profile, or function to resolve it per creator.
		 * When privacy mode is active (!hasAccess or privacyMode=true), links are omitted.
		 * Defaults to (creator) => creator.profileUrl.
		 */
		profileUrl?: string | ((creator: ContentCreator) => string | undefined);
	}

	let {
		creators,
		label = m.overview_creators_title(),
		emptyText = m.overview_creators_empty(),
		class: className,
		privacyMode = false,
		hasAccess = true,
		profileUrl = (creator) => creator.profileUrl
	}: Props = $props();

	const isPrivate = $derived(privacyMode || !hasAccess);

	function getVolume(creator: ContentCreator): number {
		return (
			creator.postsCount ??
			(creator.rated.false || 0) +
				(creator.rated.misleading || 0) +
				(creator.rated.unverified || 0) +
				(creator.rated.accurate || 0)
		);
	}

	const rankedCreators = $derived(
		creators.toSorted((a, b) => {
			const diff = getVolume(b) - getVolume(a);
			return diff !== 0 ? diff : (b.followers ?? 0) - (a.followers ?? 0);
		})
	);

	function formatCompact(count: number | undefined): string {
		if (count === undefined) return '-';
		if (count < 1_000) return formatNumber(count);
		// "12.3K" in English, "12,3 k" in French.
		return new Intl.NumberFormat(intlLocale(), {
			notation: 'compact',
			maximumFractionDigits: 1
		}).format(count);
	}

	function getPseudonym(creator: ContentCreator, index: number): string {
		return (
			creator.pseudoname || creator.pseudonym || m.overview_creator_pseudonym({ number: index + 1 })
		);
	}

	function getRiskLevel(creator: ContentCreator): number {
		if (creator.riskLevel !== undefined) {
			return creator.riskLevel;
		}
		const misinfo = (creator.rated.false || 0) + (creator.rated.misleading || 0);
		const total =
			(creator.rated.false || 0) +
			(creator.rated.misleading || 0) +
			(creator.rated.unverified || 0) +
			(creator.rated.accurate || 0);
		return total > 0 ? Math.round((misinfo / total) * 100) : 0;
	}

	function getCreatorProfileUrl(creator: ContentCreator): string | undefined {
		if (isPrivate) return undefined;
		if (typeof profileUrl === 'function') {
			return profileUrl(creator);
		}
		return profileUrl ?? creator.profileUrl;
	}
</script>

{#snippet rowContent(creator: ContentCreator, index: number, isLink: boolean)}
	{@const misinfo = (creator.rated.false || 0) + (creator.rated.misleading || 0)}
	{@const total =
		(creator.rated.false || 0) +
		(creator.rated.misleading || 0) +
		(creator.rated.unverified || 0) +
		(creator.rated.accurate || 0)}
	{@const riskLevel = getRiskLevel(creator)}

	<!-- Mobile card layout (< md) -->
	<div class="flex flex-col gap-2.5 px-4 py-3 md:hidden">
		<!-- Top row: Number, Platform, Name & Handle (or Pseudonym), Stats -->
		<div class="flex items-center gap-3">
			<!-- 1. Numbering -->
			<span class="w-4 shrink-0 text-center font-mono text-xs font-semibold text-faint-foreground">
				{index + 1}
			</span>

			<!-- 2. Platform icon -->
			<span
				title={creator.platform}
				class={cn(
					'flex size-7 shrink-0 items-center justify-center rounded-md border transition-colors',
					creator.platform === 'tiktok' && 'border-border bg-muted text-foreground',
					creator.platform === 'instagram' &&
						'border-pink-200 bg-pink-50 text-pink-600 dark:border-pink-900/60 dark:bg-pink-950/40 dark:text-pink-400',
					creator.platform === 'facebook' &&
						'border-blue-200 bg-blue-50 text-blue-600 dark:border-blue-900/60 dark:bg-blue-950/40 dark:text-blue-400',
					creator.platform === 'x' && 'border-border bg-muted text-foreground'
				)}
			>
				<PlatformIcon platform={creator.platform} class="size-4" />
			</span>
			<span class="sr-only">{creator.platform}</span>

			<!-- 3. Display name and handle -->
			<div class="min-w-0 flex-1">
				<div
					class={cn(
						'truncate text-sm font-semibold text-foreground',
						isLink && 'group-hover:underline'
					)}
				>
					{isPrivate ? getPseudonym(creator, index) : creator.name}
				</div>
				{#if !isPrivate && creator.handle}
					<div class="truncate font-mono text-xs text-muted-foreground">
						{creator.handle}
					</div>
				{/if}
			</div>

			<!-- 4. Followers & Posts summary on mobile -->
			<div class="shrink-0 text-right font-mono">
				<div class="text-[0.8125rem] font-semibold text-foreground">
					{#if isPrivate || creator.followers === undefined}
						-
					{:else}
						{formatCompact(creator.followers)}
					{/if}
					<span class="text-[0.6875rem] font-normal text-muted-foreground"
						>{m.overview_creators_followers_unit()}</span
					>
				</div>
				<div class="text-[0.75rem] text-muted-foreground">
					{#if isPrivate || creator.postsCount === undefined}
						<span>- {m.overview_creators_posts_unit()}</span>
					{:else}
						<span class="font-semibold text-foreground">{formatNumber(creator.postsCount)}</span>
						<span class="text-[0.6875rem]">{m.overview_creators_posts_unit()}</span>
					{/if}
				</div>
			</div>
		</div>

		<!-- 5. False / Misleading section & Risk gauge on mobile -->
		<div class="flex items-center gap-3.5 pt-0.5 pl-7">
			<div class="flex shrink-0 flex-col items-center gap-0.5">
				<RiskGauge value={riskLevel} size={34} />
				<span class="text-[0.625rem] font-medium tracking-wide text-muted-foreground uppercase">
					{m.overview_creators_risk()}
				</span>
			</div>

			<div class="flex min-w-0 flex-1 flex-col gap-1.5">
				<div class="flex items-baseline justify-between font-mono text-xs">
					<span class="text-[0.6875rem] font-medium tracking-wider text-muted-foreground uppercase">
						{m.overview_creators_rated_header()}
					</span>
					<div class="flex items-baseline gap-1">
						{#if isPrivate}
							<span class="font-semibold text-verdict-false-strong">
								{formatPercent(total > 0 ? (misinfo / total) * 100 : 0)}
							</span>
						{:else}
							<span class="font-semibold text-verdict-false-strong">
								{formatNumber(misinfo)}
							</span>
							<span class="text-[0.6875rem] text-muted-foreground">
								/ {formatNumber(total)}
							</span>
						{/if}
						<span class="sr-only">{m.overview_creators_rated_sr()}</span>
					</div>
				</div>
				<VerdictMeter
					counts={creator.rated}
					describe={isPrivate ? 'percent' : 'count'}
					class="h-1.5 w-full"
				/>
			</div>
		</div>
	</div>

	<!-- Desktop table layout (>= md) -->
	<div class="hidden min-w-180 items-center gap-x-6 px-5 py-3.5 md:flex">
		<!-- 1. Numbering -->
		<span
			class="w-5 shrink-0 text-center font-mono text-[0.8125rem] font-semibold text-faint-foreground"
		>
			{index + 1}
		</span>

		<!-- 2. Display name and handle -->
		<div class="min-w-0 flex-1">
			<div
				class={cn(
					'truncate text-[0.875rem] font-semibold text-foreground',
					isLink && 'group-hover:underline'
				)}
			>
				{isPrivate ? getPseudonym(creator, index) : creator.name}
			</div>
			{#if !isPrivate && creator.handle}
				<div class="truncate font-mono text-xs text-muted-foreground">
					{creator.handle}
				</div>
			{/if}
		</div>

		<!-- 3. Platform icon -->
		<div class="flex w-12 shrink-0 items-center justify-center">
			<span
				title={creator.platform}
				class={cn(
					'flex size-6 items-center justify-center rounded-md border transition-colors',
					creator.platform === 'tiktok' && 'border-border bg-muted text-foreground',
					creator.platform === 'instagram' &&
						'border-pink-200 bg-pink-50 text-pink-600 dark:border-pink-900/60 dark:bg-pink-950/40 dark:text-pink-400',
					creator.platform === 'facebook' &&
						'border-blue-200 bg-blue-50 text-blue-600 dark:border-blue-900/60 dark:bg-blue-950/40 dark:text-blue-400',
					creator.platform === 'x' && 'border-border bg-muted text-foreground'
				)}
			>
				<PlatformIcon platform={creator.platform} class="size-3.5" />
			</span>
			<span class="sr-only">{creator.platform}</span>
		</div>

		<!-- 4. Followers -->
		<div class="w-20 shrink-0 text-right font-mono">
			<div class="text-[0.8125rem] font-semibold text-foreground">
				{#if isPrivate || creator.followers === undefined}
					-
				{:else}
					{formatCompact(creator.followers)}
				{/if}
			</div>
		</div>

		<!-- 5. Posts if countable -->
		<div class="w-16 shrink-0 text-right font-mono">
			{#if isPrivate || creator.postsCount === undefined}
				<span class="text-xs text-muted-foreground">-</span>
			{:else}
				<div class="text-[0.8125rem] font-semibold text-foreground">
					{formatNumber(creator.postsCount)}
				</div>
			{/if}
		</div>

		<!-- 6. Risk level (radial gauge) -->
		<div class="flex w-20 shrink-0 items-center justify-center">
			<RiskGauge value={riskLevel} size={34} />
		</div>

		<!-- 7. Rated false or misleading & VerdictMeter -->
		<div class="flex w-28 shrink-0 flex-col items-end gap-1.5 sm:w-32">
			<div class="flex items-baseline gap-1 font-mono text-xs">
				{#if isPrivate}
					<span class="font-semibold text-verdict-false-strong">
						{formatPercent(total > 0 ? (misinfo / total) * 100 : 0)}
					</span>
				{:else}
					<span class="font-semibold text-verdict-false-strong">
						{formatNumber(misinfo)}
					</span>
					<span class="text-[0.6875rem] text-muted-foreground">
						/ {formatNumber(total)}
					</span>
				{/if}
				<span class="sr-only">{m.overview_creators_rated_sr()}</span>
			</div>
			<VerdictMeter
				counts={creator.rated}
				describe={isPrivate ? 'percent' : 'count'}
				class="h-1.5 w-full min-w-20"
			/>
		</div>
	</div>
{/snippet}

{#if creators.length === 0}
	<p
		class={cn(
			'border-t border-rule px-5 py-8 text-center text-sm text-muted-foreground',
			className
		)}
	>
		{emptyText}
	</p>
{:else}
	<div class={cn('overflow-x-auto', className)}>
		<!-- Table column headers (desktop only) -->
		<div
			class="hidden min-w-180 items-center gap-x-6 border-t border-rule bg-muted/40 px-5 py-2 text-[0.6875rem] font-semibold text-muted-foreground select-none md:flex"
		>
			<span class="w-5 shrink-0 text-center">#</span>
			<span class="min-w-0 flex-1">{m.overview_creators_creator()}</span>
			<span class="w-12 shrink-0 text-center">{m.overview_creators_platform()}</span>
			<span class="w-20 shrink-0 text-right">{m.overview_creators_followers()}</span>
			<span class="w-16 shrink-0 text-right">{m.overview_creators_posts()}</span>
			<span class="w-20 shrink-0 text-center">{m.overview_creators_risk_level()}</span>
			<span class="w-28 shrink-0 text-right sm:w-32">{m.overview_creators_rated_header()}</span>
		</div>

		<ol aria-label={label} class="mb-4 divide-y divide-rule border-t border-rule md:border-t-0">
			{#each rankedCreators as creator, index (creator.id)}
				{@const targetUrl = getCreatorProfileUrl(creator)}
				{@const isExternal = targetUrl ? /^https?:\/\//i.test(targetUrl) : false}
				<li class="relative transition-colors hover:bg-muted/60 has-focus-visible:bg-muted/60">
					{#if targetUrl}
						<a
							href={targetUrl}
							target={isExternal ? '_blank' : undefined}
							rel={isExternal ? 'noopener noreferrer' : undefined}
							class="group block outline-none"
						>
							{@render rowContent(creator, index, true)}
						</a>
					{:else}
						<div class="block">
							{@render rowContent(creator, index, false)}
						</div>
					{/if}
				</li>
			{/each}
		</ol>
	</div>
{/if}
