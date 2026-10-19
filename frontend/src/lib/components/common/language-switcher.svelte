<script lang="ts">
	import GlobeIcon from '@lucide/svelte/icons/globe';
	import { cn } from '#lib/utils.js';

	export type HeaderLang = 'fr' | 'en';

	interface Props {
		/** Current active language code. Defaults to 'fr'. */
		lang?: HeaderLang;
		/** Callback when language is switched. */
		onLangChange?: (lang: HeaderLang) => void;
		class?: string;
	}

	let { lang = $bindable('fr'), onLangChange, class: className }: Props = $props();

	function setLang(newLang: HeaderLang) {
		if (lang === newLang) return;
		lang = newLang;
		onLangChange?.(newLang);
	}
</script>

<div
	role="group"
	aria-label="Language"
	class={cn(
		'inline-flex items-center gap-x-2 rounded-md border border-inverse-border bg-inverse-raised p-0.5',
		className
	)}
>
	<GlobeIcon
		class="mr-0.5 ml-1.5 size-3.5 shrink-0 text-inverse-muted-foreground"
		aria-hidden="true"
	/>
	<button
		type="button"
		onclick={() => setLang('fr')}
		aria-pressed={lang === 'fr'}
		aria-label="Français"
		class={cn(
			'h-6 min-w-7 rounded px-1.5 text-[0.6875rem] font-semibold transition-colors outline-none focus-visible:ring-1 focus-visible:ring-inverse-signal',
			lang === 'fr'
				? 'bg-inverse-border text-inverse-foreground'
				: 'text-inverse-muted-foreground hover:bg-transparent hover:text-inverse-foreground'
		)}
	>
		FR
	</button>
	<button
		type="button"
		onclick={() => setLang('en')}
		aria-pressed={lang === 'en'}
		aria-label="English"
		class={cn(
			'h-6 min-w-7 rounded px-1.5 text-[0.6875rem] font-semibold transition-colors outline-none focus-visible:ring-1 focus-visible:ring-inverse-signal',
			lang === 'en'
				? 'bg-inverse-border text-inverse-foreground'
				: 'text-inverse-muted-foreground hover:bg-transparent hover:text-inverse-foreground'
		)}
	>
		EN
	</button>
</div>
