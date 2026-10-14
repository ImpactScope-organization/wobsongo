import { describe, expect, it } from 'vitest';
import { render } from 'svelte/server';
import { htmlSnippet } from '#lib/testing/snippet.js';
import { anomaly } from '#lib/fixtures/pulse.js';
import AnomalyBanner from './anomaly-banner.svelte';
import AppHeader from './app-header.svelte';
import CoverageNote from './coverage-note.svelte';
import PageHeader from './page-header.svelte';
import SectionCard from './section-card.svelte';
import SegmentedLinks from './segmented-links.svelte';

describe('SectionCard', () => {
	it('labels the section with its heading', () => {
		const { body } = render(SectionCard, {
			props: { title: 'Where claims come from', children: htmlSnippet('<p>Body</p>') }
		});
		const id = body.match(/aria-labelledby="([^"]+)"/)?.[1];
		expect(id).toBeTruthy();
		expect(body).toMatch(new RegExp(`<h2[^>]*id="${id}"[^>]*>\\s*Where claims come from`));
		expect(body).toContain('<p>Body</p>');
	});

	it('supports a description, actions, footer and heading level', () => {
		const { body } = render(SectionCard, {
			props: {
				title: 'Top narratives',
				description: 'Last 24 h',
				level: 3,
				actions: htmlSnippet('<a href="?topic=all">All</a>'),
				children: htmlSnippet('<ol></ol>'),
				footer: htmlSnippet('<small>Note</small>')
			}
		});
		expect(body).toContain('<h3');
		expect(body).toContain('Last 24 h');
		expect(body).toContain('href="?topic=all"');
		expect(body).toContain('<small>Note</small>');
	});
});

describe('CoverageNote', () => {
	it('defaults to the monitoring-coverage caveat', () => {
		expect(render(CoverageNote).body).toContain('not true prevalence');
	});

	it('accepts custom text', () => {
		const { body } = render(CoverageNote, { props: { children: htmlSnippet('Custom note') } });
		expect(body).toContain('Custom note');
		expect(body).not.toContain('not true prevalence');
	});
});

describe('PageHeader', () => {
	it('renders the page title as the h1, with an optional eyebrow', () => {
		const { body } = render(PageHeader, {
			props: { title: 'Pulse', eyebrow: 'Burkina Faso · updated 14:05 GMT' }
		});
		expect(body).toMatch(/<h1[^>]*>\s*Pulse/);
		expect(body).toContain('Burkina Faso · updated 14:05 GMT');
	});
});

describe('SegmentedLinks', () => {
	const options = [
		{ value: '24h', label: '24 h', href: '?range=24h' },
		{ value: '7d', label: '7 d', href: '?range=7d' }
	] as const;

	it('renders plain links so it works without JavaScript', () => {
		const { body } = render(SegmentedLinks, {
			props: { label: 'Time range', options, current: '24h' }
		});
		expect(body).toContain('aria-label="Time range"');
		expect(body).toContain('href="?range=24h"');
		expect(body).toContain('href="?range=7d"');
		expect(body).not.toContain('<button');
	});

	it('marks only the current option', () => {
		const { body } = render(SegmentedLinks, {
			props: { label: 'Time range', options, current: '7d' }
		});
		expect(body.match(/aria-current="true"/g)).toHaveLength(1);
		expect(body).toMatch(/href="\?range=7d"[^>]*aria-current="true"/);
	});
});

describe('AppHeader', () => {
	const items = [
		{ label: 'Pulse', href: '/pulse' },
		{ label: 'High-harm', href: '/safety', count: 3 },
		{ label: 'Spread', href: '/spread', count: 0 }
	];

	it('marks the current section as the page', () => {
		const { body } = render(AppHeader, { props: { items, currentHref: '/pulse' } });
		expect(body).toContain('aria-label="Main"');
		expect(body).toMatch(/href="\/pulse"[^>]*aria-current="page"/);
		expect(body.match(/aria-current/g)).toHaveLength(1);
	});

	it('shows non-zero counts only', () => {
		const { body } = render(AppHeader, { props: { items, currentHref: '/pulse' } });
		// The space must survive SSR, or screen readers hear "3open".
		expect(body).toMatch(/3<span class="sr-only">(&nbsp;|\u00a0)open<\/span>/);
		expect(body.match(/open<\/span>/g)).toHaveLength(1);
	});

	it('renders the product name and an optional logo', () => {
		const { body } = render(AppHeader, {
			props: { items, currentHref: '/pulse', logoSrc: '/logo.png' }
		});
		expect(body).toContain('WobSongo Monitor');
		expect(body).toContain('src="/logo.png"');
	});
});

describe('AnomalyBanner', () => {
	it('describes the spike and links to the narrative', () => {
		const { body } = render(AnomalyBanner, { props: { anomaly, href: '/narratives/N-0412' } });
		expect(body).toContain('“Compulsory contraceptive trial for schoolgirls”');
		expect(body).toContain('3.4σ above the usual level for this hour');
		expect(body).toContain('+412% in 3 h');
		expect(body).toContain('href="/narratives/N-0412"');
	});
});
