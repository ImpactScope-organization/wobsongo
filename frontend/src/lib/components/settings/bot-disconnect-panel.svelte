<script lang="ts">
	import type { BotPlatform } from '#lib/domain/account.js';
	import { optionalEnhance, type FormEnhance } from '#lib/utils/enhance.js';
	import { cn } from '#lib/utils.js';
	import { BOT_LABEL } from '../common/labels';
	import { SECONDARY } from './styles';

	interface Props {
		bot: BotPlatform;
		/** Form action URL, e.g. "?/disconnect". Posts `bot`. */
		action: string;
		/** Back to settings without changing anything. */
		cancelHref: string;
		/** Heading level of the title: 1 on its own page, 2 in a dialog. */
		level?: 1 | 2;
		/** Id for the title, so a dialog can point `aria-labelledby` at it. */
		titleId?: string;
		enhance?: FormEnhance;
		class?: string;
	}

	let {
		bot,
		action,
		cancelHref,
		level = 2,
		titleId: givenTitleId,
		enhance,
		class: className
	}: Props = $props();

	const ownTitleId = $props.id();
	const titleId = $derived(givenTitleId ?? ownTitleId);
</script>

<section aria-labelledby={titleId} class={cn('flex flex-col gap-3 p-[1.375rem]', className)}>
	<svelte:element
		this={level === 1 ? 'h1' : 'h2'}
		id={titleId}
		class="font-heading text-[1.1875rem] font-bold">Disconnect {BOT_LABEL[bot]}?</svelte:element
	>
	<p class="text-[0.84375rem]/normal text-pretty text-verdict-unverified-foreground">
		The bot will stop recognizing this number as an expert account. Messages you send afterwards are
		treated as a regular user. You can reconnect anytime.
	</p>
	<!-- A plain POST form: works without JavaScript; `enhance` upgrades it. -->
	<form
		method="POST"
		{action}
		use:optionalEnhance={enhance}
		class="mt-1.5 flex flex-wrap justify-end gap-2"
	>
		<input type="hidden" name="bot" value={bot} />
		<a href={cancelHref} class={SECONDARY}>Cancel</a>
		<button
			type="submit"
			class="rounded-[0.4375rem] bg-destructive px-3.5 py-[0.5625rem] text-[0.8125rem] font-semibold whitespace-nowrap text-white outline-none hover:bg-destructive/85 focus-visible:ring-2 focus-visible:ring-ring"
		>
			Disconnect
		</button>
	</form>
</section>
