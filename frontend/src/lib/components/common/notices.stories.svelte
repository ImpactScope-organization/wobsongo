<script module lang="ts">
	import { defineMeta } from '@storybook/addon-svelte-csf';
	import { expect, within } from 'storybook/test';
	import ComingSoon from './coming-soon.svelte';
	import ErrorNotice from './error-notice.svelte';

	// Whole-page messages: views not built yet, and error pages.
	const { Story } = defineMeta({
		title: 'Common/Notices',
		parameters: { layout: 'fullscreen' }
	});
</script>

<Story
	name="Coming soon"
	play={async ({ canvasElement }) => {
		const canvas = within(canvasElement);
		await expect(canvas.getByRole('heading', { level: 1 })).toHaveTextContent(
			'Narratives is coming soon'
		);
		await expect(canvas.getByRole('link', { name: 'Back to Pulse' })).toHaveAttribute('href', '/');
	}}
>
	{#snippet template()}
		<ComingSoon title="Narratives" backHref="/" />
	{/snippet}
</Story>

<Story name="Not found">
	{#snippet template()}
		<ErrorNotice status={404} message="Not found" backHref="/" />
	{/snippet}
</Story>

<Story name="Not available yet">
	{#snippet template()}
		<ErrorNotice status={501} message="Pulse is not available yet." backHref="/" />
	{/snippet}
</Story>

<Story name="Server error">
	{#snippet template()}
		<ErrorNotice
			status={500}
			message="Something went wrong on our side. Try again in a moment."
			backHref="/"
		/>
	{/snippet}
</Story>
