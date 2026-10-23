import { describe, expect, it } from 'vitest';
import { render } from 'svelte/server';
import { harmAlerts, HIGH_HARM_NOW, substances } from '#lib/fixtures/high-harm.js';
import { withLocale } from '#lib/testing/locale.js';
import { stripComments } from '#lib/testing/snippet.js';
import HarmAlertCard from './harm-alert-card.svelte';
import HarmAlertList from './harm-alert-list.svelte';
import SubstanceList from './substance-list.svelte';

const [potash, bleach, antimalarials] = harmAlerts;

describe('HarmAlertCard', () => {
	const renderCard = (alert = potash) =>
		stripComments(
			render(HarmAlertCard, {
				props: { alert, now: HIGH_HARM_NOW, action: '?/review', overrideHref: '/safety/override' }
			}).body
		);

	it('names the card by its substance and shows the quote, source and age', () => {
		const body = renderCard();
		expect(body).toMatch(/<article aria-labelledby="([^"]+)"[\s\S]*<h3 id="\1"/);
		expect(body).toContain('Potash (kanwa) + lemon · ingestion');
		expect(body).toContain('“A woman in our quarter said drinking potash');
		expect(body).toContain('WhatsApp · voice note · Mooré');
		expect(body).toContain('Centre-Nord');
		expect(body).toContain('14 similar messages');
		expect(body).toMatch(/<time datetime="2026-10-12T13:27:00.000Z"[^>]*>\s*38 min ago/);
	});

	it('marks the risk level, and whether an expert reviewed it', () => {
		expect(renderCard()).toContain('data-risk="critical"');
		expect(renderCard()).toMatch(/Critical\s*<\/span>/);
		const high = renderCard(antimalarials);
		expect(high).toContain('data-risk="high"');
		expect(high).toContain('High · reviewed');
	});

	it('shows the engine reply for the expert to check', () => {
		const body = renderCard(bleach);
		expect(body).toContain('Engine reply · False · conf. 0.89');
		expect(body).toContain('Bleach does not prevent pregnancy.');
	});

	it('posts the alert id with approve or dismiss, and links to the override editor', () => {
		const body = renderCard();
		expect(body).toMatch(/<form[^>]*method="POST"[^>]*action="\?\/review"/);
		expect(body).toContain('name="id" value="H-0091"');
		for (const decision of ['approve', 'dismiss']) {
			expect(body).toContain(`name="decision" value="${decision}"`);
		}
		expect(body).not.toContain('value="override"');
		expect(body).toMatch(/<a href="\/safety\/override"[^>]*>\s*Override…/);
	});

	it('dims a dismissed alert and offers undo', () => {
		const body = renderCard({ ...potash, review: { kind: 'decided', decision: 'dismiss' } });
		expect(body).toContain('data-decision="dismiss"');
		expect(body).toContain('value="undo"');
	});

	it('uses the singular for one similar message', () => {
		expect(renderCard({ ...potash, similarCount: 1 })).toContain('1 similar message<');
	});
});

describe('HarmAlertList', () => {
	it('lists alerts in the order given', () => {
		const { body } = render(HarmAlertList, {
			props: {
				alerts: harmAlerts,
				now: HIGH_HARM_NOW,
				action: '?/review',
				overrideHrefFor: (id: string) => `/safety/${id}/override`
			}
		});
		expect(body).toContain('<ol aria-label="Dangerous practice alerts"');
		const ids = [...body.matchAll(/name="id" value="([^"]+)"/g)].map((match) => match[1]);
		expect(ids).toEqual(['H-0091', 'H-0090', 'H-0088']);
	});

	it('shows an empty state', () => {
		const { body } = render(HarmAlertList, {
			props: { alerts: [], now: HIGH_HARM_NOW, action: '?/review', overrideHrefFor: () => '' }
		});
		expect(body).toContain('No dangerous-practice alerts.');
		expect(body).not.toContain('<ol');
	});
});

describe('SubstanceList', () => {
	it('is a table of substances, practices and counts', () => {
		const body = stripComments(render(SubstanceList, { props: { substances } }).body);
		expect(body).toContain('<caption class="sr-only">Substance and practice mentions</caption>');
		expect(body).toMatch(/Potash \(kanwa\)[\s\S]*Ingestion[\s\S]*21/);
		expect(body).toContain('Objects inserted');
	});

	it('scales bars to the most mentioned substance', () => {
		const { body } = render(SubstanceList, { props: { substances } });
		expect(body).toContain('width: 100%');
		expect(body).toContain(`width: ${(3 / 21) * 100}%`);
	});

	it('marks frequently mentioned substances', () => {
		const { body } = render(SubstanceList, { props: { substances } });
		// Potash 21, herbal 12 and bleach 9 are frequent; antimalarials 5 and objects 3 are not.
		expect(body.match(/\sdata-frequent(?=[\s=>])/g)).toHaveLength(3);
	});

	it('shows an empty state', () => {
		const { body } = render(SubstanceList, { props: { substances: [] } });
		expect(body).toContain('No lexicon matches.');
	});
});

describe('in French', () => {
	it('words the alert card, keeping 1 and 0 singular', () => {
		const body = withLocale('fr', () =>
			stripComments(
				render(HarmAlertCard, {
					props: {
						alert: { ...potash, similarCount: 0 },
						now: HIGH_HARM_NOW,
						action: '?/review',
						overrideHref: '/safety/override'
					}
				}).body
			)
		);
		expect(body).toContain('0 message similaire');
		expect(body).toContain('Réponse du moteur · Faux · conf. 0.94');
		expect(body).toContain('il y a 38 min');
		expect(body).toMatch(/>\s*Approuver\s*</);
		expect(body).toMatch(/>\s*Corriger…\s*</);
	});

	it('words the substance table', () => {
		const body = withLocale('fr', () => render(SubstanceList, { props: { substances } }).body);
		expect(body).toContain('Mentions de substances et de pratiques');
		expect(body).toContain('Substance ou pratique');
	});
});
