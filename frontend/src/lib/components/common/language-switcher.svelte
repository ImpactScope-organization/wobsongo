<script lang="ts" module>
	import type { Locale } from '#lib/paraglide/runtime.js';

	/** Each language is named in itself, so people find their own. */
	const LANGUAGES: readonly { code: Locale; short: string; name: string }[] = [
		{ code: 'fr', short: 'FR', name: 'Français' },
		{ code: 'en', short: 'EN', name: 'English' }
	];
</script>

<script lang="ts">
	import GlobeIcon from '@lucide/svelte/icons/globe';
	import { m } from '#lib/paraglide/messages.js';
	import { getLocale } from '#lib/paraglide/runtime.js';
	import { cn } from '#lib/utils.js';

	interface Props {
		/** This page's path and query, to come back to after switching. */
		returnTo?: string;
		/** Endpoint that stores the choice in a cookie. */
		action?: string;
		/** `inverse` for the dark app header, `plain` on light pages. */
		tone?: 'inverse' | 'plain';
		class?: string;
	}

	let { returnTo = '/', action = '/lang', tone = 'inverse', class: className }: Props = $props();

	// Switching reloads the page, so this never changes while it is shown.
	const current = getLocale();
</script>

<!-- A plain POST form: works without JavaScript; the server sets the cookie and sends the person back. -->
<form method="POST" {action} class={cn('inline-flex', className)}>
	<input type="hidden" name="redirectTo" value={returnTo} />
	<div
		role="group"
		aria-label={m.language_switcher_label()}
		data-tone={tone}
		class="group/lang inline-flex items-center gap-x-2 rounded-md border p-0.5 data-[tone=inverse]:border-inverse-border data-[tone=inverse]:bg-inverse-raised data-[tone=plain]:border-border data-[tone=plain]:bg-muted"
	>
		<GlobeIcon
			class="mr-0.5 ml-1.5 size-3.5 shrink-0 group-data-[tone=inverse]/lang:text-inverse-muted-foreground group-data-[tone=plain]/lang:text-muted-foreground"
			aria-hidden="true"
		/>
		{#each LANGUAGES as language (language.code)}
			<button
				type="submit"
				name="lang"
				value={language.code}
				aria-pressed={current === language.code}
				aria-label={language.name}
				lang={language.code}
				class={cn(
					'h-6 min-w-7 rounded px-1.5 text-[0.6875rem] font-semibold transition-colors outline-none',
					tone === 'inverse'
						? 'focus-visible:ring-1 focus-visible:ring-inverse-signal aria-pressed:bg-inverse-border aria-pressed:text-inverse-foreground aria-[pressed=false]:text-inverse-muted-foreground aria-[pressed=false]:hover:text-inverse-foreground'
						: 'focus-visible:ring-1 focus-visible:ring-ring aria-pressed:bg-background aria-pressed:text-foreground aria-pressed:shadow-xs aria-[pressed=false]:text-muted-foreground aria-[pressed=false]:hover:text-foreground'
				)}
			>
				{language.short}
			</button>
		{/each}
	</div>
</form>
