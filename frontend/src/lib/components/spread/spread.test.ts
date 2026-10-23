import { describe, expect, it } from 'vitest';
import { render } from 'svelte/server';
import {
	dailyIntents,
	formatWeeks,
	languageFormats,
	lags,
	platformWeeks
} from '#lib/fixtures/spread.js';
import { withLocale } from '#lib/testing/locale.js';
import { stripComments } from '#lib/testing/snippet.js';
import FormatShareChart from './format-share-chart.svelte';
import IntentBreakdown from './intent-breakdown.svelte';
import LagList from './lag-list.svelte';
import LanguageFormatMatrix from './language-format-matrix.svelte';
import PlatformShareChart from './platform-share-chart.svelte';

describe('PlatformShareChart', () => {
	it('labels weeks by ISO week and tabulates platform shares', () => {
		const body = stripComments(
			render(PlatformShareChart, { props: { weeks: platformWeeks } }).body
		);
		expect(body).toContain('<caption>Share of incoming claims by platform, weekly</caption>');
		expect(body).toMatch(/<th scope="row">W34<\/th><td>52%<\/td><td>24%<\/td>/);
		expect(body).toContain('<th scope="row">W41</th>');
		expect(body).toContain('WhatsApp (bot)');
	});
});

describe('FormatShareChart', () => {
	it('tabulates format shares by week', () => {
		const body = stripComments(render(FormatShareChart, { props: { weeks: formatWeeks } }).body);
		expect(body).toMatch(
			/<th scope="row">W41<\/th><td>52%<\/td><td>23%<\/td><td>17%<\/td><td>8%<\/td>/
		);
		expect(body).toContain('Voice note');
	});
});

describe('LagList', () => {
	const renderList = (props = {}) =>
		stripComments(render(LagList, { props: { lags, ...props } }).body);

	it('links each narrative and states its origin and lag', () => {
		const body = renderList();
		expect(body).toContain('href="/narratives/N-0412"');
		expect(body).toContain('Compulsory contraceptive trial for schoolgirls');
		expect(body).toContain('First seen on: TikTok');
		expect(body).toMatch(/\+18 h<span class="sr-only">\u00a0to WhatsApp<\/span>/);
	});

	it('places the WhatsApp marker on a 72-hour axis', () => {
		const body = renderList();
		expect(body).toMatch(/0 h<\/span><span>24 h<\/span><span>48 h<\/span><span>72 h/);
		expect(body).toContain(`left: ${(18 / 72) * 100}%`);
		// Only the 63 h lag sits past 75% of the track; the class names mention data-flip too.
		expect(body.match(/\sdata-flip(?=[\s=>])/g)).toHaveLength(1);
	});

	it('keeps a short or empty axis readable', () => {
		const short = renderList({ maxHours: 2 });
		expect(short).toMatch(/0 h<\/span><span>0\.7 h<\/span><span>1\.3 h<\/span><span>2 h/);
		const empty = renderList({ maxHours: 0 });
		expect(empty).not.toContain('NaN');
	});

	it('caps lags longer than the axis at its end', () => {
		const body = renderList({ lags: [{ ...lags[0], lagHours: 90 }] });
		expect(body).toContain('left: 100%');
		expect(body).toContain('+90 h');
	});

	it('shows an empty state', () => {
		expect(renderList({ lags: [] })).toContain('No narrative has reached WhatsApp');
	});
});

describe('LanguageFormatMatrix', () => {
	it('is a table of row percentages', () => {
		const body = stripComments(
			render(LanguageFormatMatrix, { props: { rows: languageFormats } }).body
		);
		expect(body).toContain("Share of each language's messages by format");
		expect(body).toMatch(/Mooré<\/th>(<td[^>]*>\s*61%\s*<\/td>)/);
		expect(body).toContain('Row percentages.');
	});

	it('shades cells by share and darkens text on dark cells', () => {
		const { body } = render(LanguageFormatMatrix, {
			props: { rows: [{ language: 'Mooré', counts: { audio: 70, video: 30, text: 0, image: 0 } }] }
		});
		expect(body).toContain('color-mix(in srgb, var(--primary) 60%, var(--muted))');
		expect(body.match(/\sdata-dark(?=[\s=>])/g)).toHaveLength(1);
	});
});

describe('IntentBreakdown', () => {
	it('shows one described bar per day, labelled by weekday', () => {
		const body = stripComments(render(IntentBreakdown, { props: { days: dailyIntents } }).body);
		expect(body.match(/<dt/g)).toHaveLength(7);
		expect(body).toMatch(/<time datetime="2026-10-05">Mon<\/time>/);
		expect(body).toContain(
			'aria-label="Verification request 58%, General SRH question 36%, Out of scope 6%"'
		);
		expect(body.match(/data-size="md"/g)).toHaveLength(7);
	});

	it('explains each intent in a legend screen readers can read', () => {
		const body = render(IntentBreakdown, { props: { days: dailyIntents } }).body;
		expect(body).toContain('Personal medical questions the platform does not handle');
		expect(body).not.toMatch(/<ul aria-hidden="true"[^>]*grid/);
	});
});

describe('Spread in French', () => {
	it('words the lag list in French', () => {
		const body = withLocale('fr', () => stripComments(render(LagList, { props: { lags } }).body));
		expect(body).toContain('aria-label="Délai entre plateformes publiques et WhatsApp, par récit"');
		expect(body).toContain('Vu d’abord sur\u00a0: TikTok');
		expect(body).toContain('Plateformes publiques uniquement');
	});

	it('labels weeks and formats in French', () => {
		const body = withLocale(
			'fr',
			() => render(FormatShareChart, { props: { weeks: formatWeeks } }).body
		);
		expect(body).toContain('Part des affirmations par format, par semaine');
		expect(body).toContain('Note vocale');
		expect(body).toMatch(/>S\d+</);
	});
});
