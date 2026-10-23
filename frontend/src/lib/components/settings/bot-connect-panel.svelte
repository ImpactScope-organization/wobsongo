<script lang="ts">
	import { onMount } from 'svelte';
	import {
		BOT_LINK_PREFIX,
		botLinkMessage,
		type BotContact,
		type BotLinkCode,
		type BotLinkPhase
	} from '#lib/domain/account.js';
	import { optionalEnhance, type FormEnhance } from '#lib/utils/enhance.js';
	import { m } from '#lib/paraglide/messages.js';
	import { formatCountdown } from '#lib/utils/format.js';
	import { cn } from '#lib/utils.js';
	import { BOT_LABEL } from '../common/labels';
	import { PRIMARY, TEXT_LINK } from './styles';

	interface Props {
		contact: BotContact;
		phase: BotLinkPhase;
		/** The code issued by the server; required in the `code` phase. */
		code?: BotLinkCode;
		/** Pass the injected clock's now; the countdown runs from here. */
		now: Date;
		/** Form action for "I've sent it", e.g. "?/sent". */
		sentAction: string;
		/** Form action that issues a fresh code, e.g. "?/regenerate". */
		regenerateAction: string;
		/** Back to settings: the close button and "Done". */
		closeHref: string;
		/** While waiting: reloads to check again (the route polls when it can). */
		checkHref: string;
		/** Heading level of the title: 1 on its own page, 2 in a dialog. */
		level?: 1 | 2;
		/** Id for the title, so a dialog can point `aria-labelledby` at it. */
		titleId?: string;
		enhance?: FormEnhance;
		class?: string;
	}

	let {
		contact,
		phase,
		code,
		now,
		sentAction,
		regenerateAction,
		closeHref,
		checkHref,
		level = 2,
		titleId: givenTitleId,
		enhance,
		class: className
	}: Props = $props();

	const ownTitleId = $props.id();
	const titleId = $derived(givenTitleId ?? ownTitleId);

	const name = $derived(BOT_LABEL[contact.bot]);
	const message = $derived(code ? botLinkMessage(code) : '');

	// Time passed since render, measured locally so the browser's clock can be wrong.
	let elapsed = $state(0);
	$effect(() => {
		if (phase !== 'code' || !code) return;
		const start = performance.now();
		elapsed = 0;
		const timer = setInterval(() => (elapsed = performance.now() - start), 1000);
		return () => clearInterval(timer);
	});
	const left = $derived(code ? code.expiresAt.getTime() - now.getTime() - elapsed : 0);
	const expired = $derived(left <= 0);

	// Copying needs the Clipboard API, so the button only appears where it works.
	let canCopy = $state(false);
	// Remembers what was copied, so a regenerated code reads "Copy message" again.
	let copiedMessage = $state<string | null>(null);
	const copied = $derived(copiedMessage === message);
	// After mount, not derived: the server can't know, and hydration must match it.
	onMount(() => {
		canCopy = !!navigator.clipboard;
	});
	async function copy() {
		try {
			await navigator.clipboard.writeText(message);
			copiedMessage = message;
		} catch {
			canCopy = false;
		}
	}
</script>

<section aria-labelledby={titleId} class={cn('flex flex-col', className)}>
	<header class="flex items-start justify-between gap-3 px-[1.375rem] pt-5">
		<div>
			<p
				class="mb-1 font-mono text-[0.6875rem] font-medium tracking-[0.06em] text-muted-foreground uppercase"
			>
				{name} · {contact.address}
			</p>
			<svelte:element
				this={level === 1 ? 'h1' : 'h2'}
				id={titleId}
				class="font-heading text-xl font-bold"
				>{m.settings_bot_connect_title({ name })}</svelte:element
			>
		</div>
		<a
			href={closeHref}
			aria-label={m.settings_close()}
			class="rounded px-1 py-0.5 text-[1.375rem]/none text-muted-foreground outline-none hover:text-foreground focus-visible:ring-2 focus-visible:ring-ring"
			>×</a
		>
	</header>

	{#if phase === 'code' && code}
		<div class="flex flex-col gap-4 px-[1.375rem] pt-4 pb-[1.375rem]">
			<ol class="flex list-decimal flex-col gap-1.5 pl-5 text-[0.84375rem]/[1.45]">
				<li>{m.settings_bot_step_copy()}</li>
				<li>{m.settings_bot_step_send({ address: contact.address })}</li>
				<li>{m.settings_bot_step_confirm()}</li>
			</ol>

			<div
				data-expired={expired || undefined}
				class="group flex flex-col gap-2.5 rounded-[0.5625rem] bg-inverse px-4 py-3.5 text-inverse-foreground"
			>
				<div class="flex items-center justify-between gap-2.5">
					<span
						class="font-mono text-[0.65625rem] font-semibold tracking-[0.06em] text-inverse-muted-foreground uppercase"
					>
						{m.settings_bot_auth_message()}
					</span>
					<span
						class={cn(
							'font-mono text-[0.71875rem] font-medium',
							left > 60_000 ? 'text-inverse-muted-foreground' : 'text-inverse-signal'
						)}
					>
						{#if expired}
							{m.settings_bot_expired()}
						{:else}
							{m.settings_bot_expires_in()}
							<time datetime="PT{Math.ceil(left / 1000)}S">{formatCountdown(left)}</time>
						{/if}
					</span>
				</div>
				<!-- The countdown isn't live (it would speak every second); expiry is. -->
				<span role="status" class="sr-only">{expired ? m.settings_bot_code_expired() : ''}</span>
				<p
					class="font-mono text-[clamp(1.0625rem,4.4vw,1.3125rem)]/[1.3] font-semibold tracking-[0.02em] break-all select-all group-data-expired:text-inverse-muted-foreground group-data-expired:line-through"
				>
					{message}
				</p>
				{#if !expired}
					<div class="flex flex-wrap gap-2">
						{#if canCopy}
							<button
								type="button"
								onclick={copy}
								class="min-w-30 flex-1 rounded-md bg-inverse-foreground px-3 py-[0.5625rem] text-[0.8125rem] font-semibold text-inverse outline-none focus-visible:ring-2 focus-visible:ring-inverse-signal"
							>
								{copied ? m.settings_bot_copied() : m.settings_bot_copy_message()}
							</button>
						{/if}
						<a
							href={contact.chatUrl(message)}
							target="_blank"
							rel="noopener noreferrer"
							class="min-w-30 flex-1 rounded-md border border-inverse-border px-3 py-[0.5625rem] text-center text-[0.8125rem] font-medium outline-none hover:bg-inverse-raised focus-visible:ring-2 focus-visible:ring-inverse-signal"
						>
							{m.settings_bot_open({ name })}<span class="sr-only"
								>&nbsp;{m.settings_opens_new_tab()}</span
							>
						</a>
					</div>
				{/if}
			</div>

			<p class="text-xs/normal text-pretty text-muted-foreground">
				{m.settings_bot_prefix_note({ prefix: BOT_LINK_PREFIX })}
			</p>

			<div class="flex flex-wrap items-center justify-between gap-2.5 border-t border-rule pt-3">
				<form method="POST" action={regenerateAction} use:optionalEnhance={enhance}>
					<button type="submit" class={cn(expired ? PRIMARY : cn(TEXT_LINK, 'py-1'))}>
						{m.settings_bot_generate_code()}
					</button>
				</form>
				{#if !expired}
					<form method="POST" action={sentAction} use:optionalEnhance={enhance}>
						<button type="submit" class={PRIMARY}>{m.settings_bot_sent_it()}</button>
					</form>
				{/if}
			</div>
		</div>
	{:else if phase === 'waiting'}
		<div class="flex flex-col items-center gap-3 px-[1.375rem] pt-7 pb-[1.625rem] text-center">
			<span
				aria-hidden="true"
				class="size-[2.125rem] rounded-full border-3 border-rule border-t-foreground motion-safe:animate-spin"
			></span>
			<p role="status" class="text-[0.9375rem] font-semibold">{m.settings_bot_waiting()}</p>
			<p class="max-w-[21.25rem] text-[0.8125rem]/[1.45] text-muted-foreground">
				{m.settings_bot_waiting_note()}
			</p>
			<a href={checkHref} class={TEXT_LINK}>{m.settings_bot_check_again()}</a>
		</div>
	{:else}
		<div
			class="flex flex-col items-center gap-2.5 px-[1.375rem] pt-[1.625rem] pb-[1.375rem] text-center"
		>
			<span
				aria-hidden="true"
				class="grid size-11 place-items-center rounded-full bg-status-verified text-xl font-bold text-status-verified-foreground"
				>✓</span
			>
			<p role="status" class="text-base font-semibold">{m.settings_bot_account_connected()}</p>
			<p class="text-[0.8125rem]/[1.45] text-muted-foreground">
				{m.settings_bot_can_review({ name })}
			</p>
			<a href={closeHref} class={cn(PRIMARY, 'mt-2')}>{m.settings_done()}</a>
		</div>
	{/if}
</section>
