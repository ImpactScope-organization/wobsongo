import { describe, expect, it } from 'vitest';
import { render } from 'svelte/server';
import Sparkline from './sparkline.svelte';

describe('Sparkline', () => {
	it('renders the line on the server, labelled for screen readers', () => {
		const { body } = render(Sparkline, {
			props: { values: [1, 2, 4, 3, 9], label: 'Volume, rising' }
		});
		expect(body).toContain('role="img"');
		expect(body).toContain('aria-label="Volume, rising"');
		expect(body).toMatch(/<path[^>]*d="M/);
	});

	it('draws no line for fewer than two values', () => {
		const { body } = render(Sparkline, { props: { values: [5], label: 'Volume' } });
		expect(body).toContain('aria-label="Volume"');
		expect(body).not.toMatch(/<path[^>]*d="M/);
	});
});
