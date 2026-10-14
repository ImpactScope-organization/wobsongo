<script lang="ts" module>
	import type { RatingOutcome } from '#lib/domain/narrative.js';

	/** Number of rated claims per outcome. */
	export type VerdictCounts = Record<RatingOutcome, number>;
</script>

<script lang="ts">
	import { RATING_OUTCOME_LABEL } from './labels';
	import StackedBar from './stacked-bar.svelte';

	interface Props {
		counts: VerdictCounts;
		class?: string;
	}

	let { counts, class: className }: Props = $props();

	const TONE: Record<RatingOutcome, string> = {
		false: 'bg-verdict-false-strong',
		misleading: 'bg-verdict-misleading-strong',
		unverified: 'bg-verdict-unverified-strong',
		accurate: 'bg-verdict-accurate-strong'
	};
	const OUTCOMES: readonly RatingOutcome[] = ['false', 'misleading', 'unverified', 'accurate'];

	// Read as "14 false, 3 misleading", so labels are lowercased.
	const segments = $derived(
		OUTCOMES.map((outcome) => ({
			key: outcome,
			label: RATING_OUTCOME_LABEL[outcome].toLowerCase(),
			value: counts[outcome],
			class: TONE[outcome]
		}))
	);
</script>

<StackedBar {segments} emptyText="No rated narratives" class={className} />
