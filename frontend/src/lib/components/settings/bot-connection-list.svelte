<script lang="ts">
	import type { BotConnection, BotContact, BotPlatform } from '#lib/domain/account.js';
	import { formatDate } from '#lib/utils/format.js';
	import { m } from '#lib/paraglide/messages.js';
	import { cn } from '#lib/utils.js';
	import { BOT_LABEL, BOT_MONOGRAM } from '../common/labels';
	import { BOT_TONE } from '../common/tones';
	import { PRIMARY, SECONDARY } from './styles';

	interface Props {
		connections: readonly BotConnection[];
		contacts: Readonly<Record<BotPlatform, BotContact>>;
		/** Page that issues a link code, e.g. "/settings/bots/whatsapp". */
		connectHrefFor: (bot: BotPlatform) => string;
		/** Page that confirms unlinking, e.g. "/settings/bots/telegram/disconnect". */
		disconnectHrefFor: (bot: BotPlatform) => string;
		class?: string;
	}

	let {
		connections,
		contacts,
		connectHrefFor,
		disconnectHrefFor,
		class: className
	}: Props = $props();
</script>

<ul aria-label={m.settings_bot_connections_label()} class={className}>
	{#each connections as connection (connection.bot)}
		{@const name = BOT_LABEL[connection.bot]}
		{@const connected = connection.status === 'connected'}
		<li
			data-status={connection.status}
			class="flex flex-wrap items-center gap-x-[1.125rem] gap-y-3.5 border-t border-rule px-[1.375rem] py-4"
		>
			<span
				aria-hidden="true"
				class={cn(
					'grid size-10 shrink-0 place-items-center rounded-[0.5625rem] font-mono text-[0.8125rem] font-semibold',
					BOT_TONE[connection.bot]
				)}
			>
				{BOT_MONOGRAM[connection.bot]}
			</span>
			<div class="min-w-0 flex-[1_1_15rem]">
				<div class="flex flex-wrap items-center gap-2">
					<h3 class="text-[0.9375rem] font-semibold">{name}</h3>
					<span
						class={cn(
							'flex items-center gap-[0.3125rem] rounded-full px-2 py-0.5 text-[0.71875rem] font-medium',
							connected
								? 'bg-status-verified text-status-verified-foreground'
								: 'bg-secondary text-muted-foreground'
						)}
					>
						<span aria-hidden="true" class="size-1.5 rounded-full bg-current"></span>
						{connected ? m.settings_bot_connected() : m.settings_bot_not_connected()}
					</span>
				</div>
				<p class="mt-[0.1875rem] text-[0.78125rem]/[1.45] text-pretty text-muted-foreground">
					{#if connection.status === 'connected'}
						{m.settings_bot_linked_to({ handle: connection.handle })}
						<time datetime={connection.since.toISOString()}>{formatDate(connection.since)}</time>
					{:else}
						{m.settings_bot_address({ address: contacts[connection.bot].address })}
					{/if}
				</p>
			</div>
			{#if connected}
				<a href={disconnectHrefFor(connection.bot)} class={SECONDARY}>
					{m.settings_bot_disconnect()}<span class="sr-only">&nbsp;{name}</span>
				</a>
			{:else}
				<a href={connectHrefFor(connection.bot)} class={PRIMARY}>
					{m.settings_bot_connect()}<span class="sr-only">&nbsp;{name}</span>
				</a>
			{/if}
		</li>
	{/each}
</ul>
