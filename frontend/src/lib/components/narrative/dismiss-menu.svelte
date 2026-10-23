<script lang="ts">
	import ChevronDownIcon from '@lucide/svelte/icons/chevron-down';
	import { DISMISS_REASONS } from '#lib/domain/narrative-detail.js';
	import { optionalEnhance, type FormEnhance } from '#lib/utils/enhance.js';
	import { m } from '#lib/paraglide/messages.js';
	import { cn } from '#lib/utils.js';
	import { DISMISS_REASON_LABEL } from '../common/labels';

	interface Props {
		narrativeId: string;
		/** Form action URL, e.g. "?/dismiss". Posts `id` and `reason`. */
		action: string;
		/** Progressive enhancement, e.g. SvelteKit's `enhance`. */
		enhance?: FormEnhance;
		/** Start open, e.g. to show the menu in stories. */
		open?: boolean;
		class?: string;
	}

	let { narrativeId, action, enhance, open = false, class: className }: Props = $props();

	let details = $state<HTMLDetailsElement>();

	// Enhancement only: <details> already opens and closes without JavaScript.
	function closeOnOutsideClick(event: MouseEvent) {
		if (details?.open && !details.contains(event.target as Node)) details.open = false;
	}

	function closeOnEscape(event: KeyboardEvent) {
		if (event.key === 'Escape' && details?.open) {
			details.open = false;
			details.querySelector('summary')?.focus();
		}
	}
</script>

<svelte:window onclick={closeOnOutsideClick} onkeydown={closeOnEscape} />

<details bind:this={details} {open} class={cn('group relative', className)}>
	<summary
		class="flex cursor-pointer list-none items-center gap-1.5 rounded-md border bg-card px-3.5 py-[0.5625rem] text-[0.8125rem] font-medium outline-none select-none hover:border-foreground focus-visible:ring-2 focus-visible:ring-ring [&::-webkit-details-marker]:hidden"
	>
		{m.narrative_dismiss()}
		<ChevronDownIcon
			class="size-3.5 transition-transform group-open:rotate-180"
			aria-hidden="true"
		/>
	</summary>

	<form
		method="POST"
		{action}
		use:optionalEnhance={enhance}
		class="absolute top-[calc(100%+0.375rem)] right-0 z-10 w-65 rounded-lg border bg-popover p-1.5 whitespace-normal shadow-[0_10px_28px_rgb(28_27_24/0.14)]"
	>
		<input type="hidden" name="id" value={narrativeId} />
		<fieldset>
			<legend
				class="text-eyebrow px-2 pt-1.5 pb-1 text-[0.65625rem] font-semibold text-faint-foreground"
			>
				{m.narrative_dismiss_legend()}
			</legend>
			{#each DISMISS_REASONS as reason (reason)}
				<button
					type="submit"
					name="reason"
					value={reason}
					class="block w-full rounded-md p-2 text-left outline-none hover:bg-background focus-visible:bg-background focus-visible:ring-2 focus-visible:ring-ring"
				>
					<span class="block text-[0.8125rem] font-medium"
						>{DISMISS_REASON_LABEL[reason].label}</span
					>
					<span class="block text-[0.71875rem] text-muted-foreground">
						{DISMISS_REASON_LABEL[reason].description}
					</span>
				</button>
			{/each}
		</fieldset>
		<p
			class="mt-1 border-t border-rule px-2 pt-1.5 pb-1 text-[0.71875rem]/snug text-faint-foreground"
		>
			{m.narrative_dismiss_note()}
		</p>
	</form>
</details>
