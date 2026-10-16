import { describe, expect, it } from 'vitest';
import { render } from 'svelte/server';
import { htmlSnippet } from '#lib/testing/snippet.js';
import { anomaly } from '#lib/fixtures/overview.js';
import AnomalyBanner from './anomaly-banner.svelte';
import AppHeader from './app-header.svelte';
import ComingSoon from './coming-soon.svelte';
import CoverageNote from './coverage-note.svelte';
import DockMenu from './dock-menu.svelte';
import ErrorNotice from './error-notice.svelte';
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
			props: { title: 'Overview', eyebrow: 'Burkina Faso · updated 14:05 GMT' }
		});
		expect(body).toMatch(/<h1[^>]*>\s*Overview/);
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
		{ label: 'Overview', href: '/overview' },
		{ label: 'High-harm', href: '/safety', count: 3 },
		{ label: 'Spread', href: '/spread', count: 0 }
	];

	it('marks the current section as the page', () => {
		const { body } = render(AppHeader, { props: { items, currentHref: '/overview' } });
		expect(body).toContain('aria-label="Main"');
		expect(body).toMatch(/href="\/overview"[^>]*aria-current="page"/);
		expect(body.match(/aria-current/g)).toHaveLength(1);
	});

	it('shows non-zero counts only', () => {
		const { body } = render(AppHeader, { props: { items, currentHref: '/overview' } });
		// The space must survive SSR, or screen readers hear "3open".
		expect(body).toMatch(/3<span class="sr-only">(&nbsp;|\u00a0)open<\/span>/);
		expect(body.match(/open<\/span>/g)).toHaveLength(1);
	});

	it('renders an optional logo', () => {
		const { body } = render(AppHeader, {
			props: { items, currentHref: '/overview', logoSrc: '/logo.png' }
		});
		expect(body).toContain('src="/logo.png"');
	});

	it('hides desktop navigation on mobile screens', () => {
		const { body } = render(AppHeader, { props: { items, currentHref: '/overview' } });
		expect(body).toMatch(/<nav[^>]*class="[^"]*hidden[^"]*md:block/);
	});

	it('renders mobile dock when mobileDock is true', () => {
		const { body } = render(AppHeader, {
			props: { items, currentHref: '/overview', mobileDock: true }
		});
		expect(body).toContain('aria-label="Mobile Navigation"');
	});
});

describe('DockMenu', () => {
	const items = [
		{ label: 'Overview', href: '/overview', count: 5 },
		{ label: 'High-harm', href: '/safety', count: 3, disabled: true },
		{ label: 'Spread', href: '/spread', count: 0 }
	];

	it('renders mobile navigation with icons and marks current page', () => {
		const { body } = render(DockMenu, { props: { items, currentHref: '/overview' } });
		expect(body).toContain('aria-label="Mobile Navigation"');
		expect(body).toMatch(/href="\/overview"[^>]*aria-current="page"/);
		expect(body).toContain('Overview');
	});

	it('shows non-zero counts and disables coming-soon items', () => {
		const { body } = render(DockMenu, { props: { items, currentHref: '/overview' } });
		expect(body).toMatch(/5\s*<span class="sr-only">/);
		expect(body).toContain('(coming soon)');
		expect(body).not.toContain('href="/safety"');
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

describe('AppHeader disabled items', () => {
	const items = [
		{ label: 'Overview', href: '/' },
		{ label: 'High-harm', href: '/safety', count: 3, disabled: true }
	];

	it('shows views not built yet as text, not links, without their count', () => {
		const { body } = render(AppHeader, { props: { items, currentHref: '/' } });
		expect(body).not.toContain('href="/safety"');
		expect(body).toMatch(/<span aria-disabled="true"[^>]*>\s*High-harm/);
		expect(body).toContain('(coming soon)');
		expect(body).not.toContain('3<span class="sr-only">');
	});
});

describe('ComingSoon', () => {
	it('names the view and links back', () => {
		const { body } = render(ComingSoon, { props: { title: 'Spread', backHref: '/' } });
		expect(body).toMatch(/<h1>Spread is coming soon<\/h1>/);
		expect(body).toMatch(/<a href="\/"[^>]*>Back to Overview<\/a>/);
	});
});

describe('ErrorNotice', () => {
	const title = (status: number) =>
		render(ErrorNotice, { props: { status, message: 'Details.', backHref: '/' } }).body.match(
			/<h1>([^<]+)<\/h1>/
		)?.[1];

	it('titles the page by status', () => {
		expect(title(404)).toBe('Page not found');
		expect(title(501)).toBe('Not available yet');
		expect(title(500)).toBe('Something went wrong');
		expect(title(418)).toBe('This page can’t be shown');
	});

	it('shows the server message and the numeric code', () => {
		const { body } = render(ErrorNotice, {
			props: { status: 404, message: 'No such narrative.', backHref: '/' }
		});
		expect(body).toContain('No such narrative.');
		expect(body).toContain('Error 404');
		expect(body).toMatch(/<a href="\/"[^>]*>Back to Overview<\/a>/);
	});
});
