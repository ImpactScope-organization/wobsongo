<script lang="ts">
	import type { FormatCount, LanguageCount } from '#lib/domain/narrative-detail.js';
	import { cn } from '#lib/utils.js';
	import BarList from '../common/bar-list.svelte';
	import { MESSAGE_FORMAT_LABEL } from '../common/labels';
	import StackedBar from '../common/stacked-bar.svelte';
	import { FORMAT_TONE } from '../common/tones';

	interface Props {
		languages: readonly LanguageCount[];
		formats: readonly FormatCount[];
		class?: string;
	}

	let { languages, formats, class: className }: Props = $props();
</script>

<div class={cn('flex flex-col', className)}>
	<BarList
		caption="Messages by language"
		labelHeader="Language"
		items={languages.map((item) => ({
			key: item.language,
			label: item.language,
			value: item.count
		}))}
	/>

	<h3 class="text-eyebrow mt-5 mb-2.5 font-semibold tracking-[0.05em] text-muted-foreground">
		Format
	</h3>
	<StackedBar
		size="lg"
		describe="percent"
		showPercent
		segments={formats.map((item) => ({
			key: item.format,
			label: MESSAGE_FORMAT_LABEL[item.format],
			value: item.count,
			class: FORMAT_TONE[item.format].bar
		}))}
	/>
	<ul
		class="mt-2 flex flex-wrap gap-3 text-[0.71875rem] font-medium text-muted-foreground"
		aria-hidden="true"
	>
		{#each formats as item (item.format)}
			<li class="flex items-center gap-1.5">
				<span class={cn('size-2 rounded-[1px]', FORMAT_TONE[item.format].swatch)}></span>
				{MESSAGE_FORMAT_LABEL[item.format]}
			</li>
		{/each}
	</ul>
</div>
