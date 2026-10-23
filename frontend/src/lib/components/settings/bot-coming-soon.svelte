<script lang="ts">
	import { botLinkMessage, type BotPlatform } from '#lib/domain/account.js';
	import { m } from '#lib/paraglide/messages.js';
	import { cn } from '#lib/utils.js';
	import { BOT_LABEL, BOT_MONOGRAM } from '../common/labels';
	import { BOT_TONE } from '../common/tones';
	import { PRIMARY } from './styles';

	interface Props {
		bot?: BotPlatform;
		class?: string;
	}

	let { bot = 'whatsapp', class: className }: Props = $props();

	const name = $derived(BOT_LABEL[bot]);
	// A made-up code: linking isn't built yet, this only shows how it will work.
	const example = botLinkMessage({ code: '123456' });
</script>

<!--
	Linking a messaging account isn't built yet; until it is, the bot answers
	everyone. This previews the flow with a disabled Connect button.
-->
<div
	data-status="coming-soon"
	class={cn(
		'flex flex-wrap items-center gap-x-[1.125rem] gap-y-3.5 border-t border-rule px-[1.375rem] py-4',
		className
	)}
>
	<span
		aria-hidden="true"
		class={cn(
			'grid size-10 shrink-0 place-items-center rounded-[0.5625rem] font-mono text-[0.8125rem] font-semibold',
			BOT_TONE[bot]
		)}
	>
		{BOT_MONOGRAM[bot]}
	</span>
	<div class="min-w-0 flex-[1_1_15rem]">
		<div class="flex flex-wrap items-center gap-2">
			<h3 class="text-[0.9375rem] font-semibold">{name}</h3>
			<span
				class="rounded-full bg-secondary px-2 py-0.5 text-[0.71875rem] font-medium text-muted-foreground"
			>
				{m.settings_bot_coming_soon()}
			</span>
		</div>
		<p class="mt-[0.1875rem] text-[0.78125rem]/[1.45] text-pretty text-muted-foreground">
			{m.settings_bot_coming_soon_intro()}
		</p>
		<p class="mt-2 flex flex-wrap items-center gap-2 text-[0.75rem] text-muted-foreground">
			<span>{m.settings_bot_example()}</span>
			<code class="rounded-[0.3125rem] border bg-background px-2 py-1 font-mono text-foreground/70">
				{example}
			</code>
		</p>
	</div>
	<button
		type="button"
		disabled
		class={cn(PRIMARY, 'ml-auto w-full disabled:pointer-events-none disabled:opacity-50 md:w-auto')}
	>
		{m.settings_bot_connect()}<span class="sr-only"
			>&nbsp;{m.settings_bot_connect_coming_soon_suffix({ name })}</span
		>
	</button>
</div>
