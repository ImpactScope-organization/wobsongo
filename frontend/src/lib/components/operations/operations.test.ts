import { describe, expect, it } from 'vitest';
import { render } from 'svelte/server';
import { isHighFollowUp, priorityQuadrant, prioritySeverity } from '#lib/domain/operations.js';
import {
	debunkOutcomes,
	expertChecks,
	knowledgeGaps,
	OPERATIONS_NOW,
	priorityPoints
} from '#lib/fixtures/operations.js';
import { withLocale } from '#lib/testing/locale.js';
import { stripComments } from '#lib/testing/snippet.js';
import DebunkEffectiveness from './debunk-effectiveness.svelte';
import ExpertCheckList from './expert-check-list.svelte';
import KnowledgeGapList from './knowledge-gap-list.svelte';
import PriorityMatrix from './priority-matrix.svelte';

describe('isHighFollowUp', () => {
	it('flags follow-up rates above 25%', () => {
		expect(isHighFollowUp({ ...debunkOutcomes[0], followUpPct: 25 })).toBe(false);
		expect(isHighFollowUp({ ...debunkOutcomes[0], followUpPct: 26 })).toBe(true);
	});
});

describe('priorityQuadrant', () => {
	it('splits at harm 3 and velocity 0.5, both inclusive', () => {
		expect(priorityQuadrant({ harm: 3, velocity: 0.5 })).toBe('urgent');
		expect(priorityQuadrant({ harm: 3, velocity: 0.49 })).toBe('critical');
		expect(priorityQuadrant({ harm: 2, velocity: 0.5 })).toBe('campaign');
		expect(priorityQuadrant({ harm: 2, velocity: 0.49 })).toBe('monitor');
	});
});

describe('prioritySeverity', () => {
	it('follows the high-harm threshold, then the matrix split', () => {
		expect(prioritySeverity({ harm: 4 })).toBe('high');
		expect(prioritySeverity({ harm: 3 })).toBe('elevated');
		expect(prioritySeverity({ harm: 2 })).toBe('low');
	});
});

describe('ExpertCheckList', () => {
	const renderList = (checks = expertChecks) =>
		stripComments(
			render(ExpertCheckList, {
				props: {
					checks,
					action: '?/review',
					overrideHrefFor: (id: string) => `/ops/checks/${id}/override`
				}
			}).body
		);

	it('shows each engine result with its verdict, confidence and claim', () => {
		const body = renderList();
		expect(body).toContain('<ol aria-label="Expert review queue"');
		expect(body).toContain('data-verdict="misleading"');
		expect(body).toContain('conf. 0.58');
		expect(body).toMatch(/<h3[^>]*>Contraceptive injections cause cancer<\/h3>/);
		expect(body).toContain('Injectable contraceptives are not known to cause cancer.');
	});

	it('explains why each result was flagged', () => {
		const body = renderList();
		expect(body).toMatch(/Flagged for:<\/span> Harm 4 · spiking/);
		expect(body).toContain('Low confidence · Mooré audio');
		expect(body).toContain('Partly true');
	});

	it('posts each item to the review action', () => {
		const body = renderList();
		const ids = [...body.matchAll(/name="id" value="([^"]+)"/g)].map((match) => match[1]);
		expect(ids).toEqual(['C-0201', 'C-0202', 'C-0203', 'C-0204', 'C-0205']);
		expect(body).not.toContain('value="override"');
		expect(body).toContain('<a href="/ops/checks/C-0203/override"');
	});

	it('dims dismissed items', () => {
		const body = renderList([
			{ ...expertChecks[0], review: { kind: 'decided', decision: 'dismiss' } }
		]);
		expect(body).toContain('data-decision="dismiss"');
	});

	it('shows an empty state', () => {
		expect(renderList([])).toContain('Nothing waiting for an expert.');
	});
});

describe('KnowledgeGapList', () => {
	const renderList = (gaps = knowledgeGaps) =>
		stripComments(
			render(KnowledgeGapList, { props: { gaps, now: OPERATIONS_NOW, action: '?/gap' } }).body
		);

	it('shows the cluster, an example, languages, age and volume', () => {
		const body = renderList();
		expect(body).toMatch(/<h3[^>]*>Sayana Press causes weight gain/);
		expect(body).toContain('“Is it true the injection in the arm');
		expect(body).toMatch(/Mooré, French · first asked <time[^>]*>3 d ago<\/time>/);
		expect(body).toMatch(/288\s*<\/span>/);
	});

	it('offers add fact-check or dismiss', () => {
		const body = renderList();
		expect(body).toMatch(/<form[^>]*action="\?\/gap"/);
		expect(body).toContain('value="add-fact-check"');
		expect(body).not.toContain('value="approve"');
	});

	it('confirms a created draft', () => {
		const body = renderList([
			{ ...knowledgeGaps[0], review: { kind: 'decided', decision: 'add-fact-check' } }
		]);
		expect(body).toContain('Fact-check draft created');
	});

	it('shows an empty state', () => {
		expect(renderList([])).toContain('Every claim matched the knowledge base.');
	});
});

describe('DebunkEffectiveness', () => {
	it('is a table of sent, helpful and asked-again rates', () => {
		const body = stripComments(
			render(DebunkEffectiveness, { props: { outcomes: debunkOutcomes } }).body
		);
		expect(body).toContain('<caption class="sr-only">Debunk effectiveness</caption>');
		expect(body).toMatch(
			/Implants do not cause permanent infertility[\s\S]*3,412[\s\S]*78%[\s\S]*12%/
		);
	});

	it('highlights high follow-up rates', () => {
		const { body } = render(DebunkEffectiveness, { props: { outcomes: debunkOutcomes } });
		expect(body.match(/data-high/g)).toHaveLength(1);
		// Not by colour alone.
		expect(body.match(/<span class="sr-only">(&nbsp;|\u00a0)\(high\)<\/span>/g)).toHaveLength(1);
	});
});

describe('PriorityMatrix', () => {
	const renderMatrix = (points = priorityPoints) =>
		stripComments(render(PriorityMatrix, { props: { points } }).body);

	it('renders numbered bubbles on the server', () => {
		const body = renderMatrix();
		expect(body.match(/<circle/g)).toHaveLength(7);
		expect(body).toContain('viewBox="0 0 480 300"');
	});

	it('sorts narratives into quadrants by harm and velocity', () => {
		const body = renderMatrix();
		expect(body.match(/data-quadrant="urgent"/g)).toHaveLength(3);
		expect(body.match(/data-quadrant="critical"/g)).toHaveLength(2);
		expect(body).toMatch(
			/role="img"[^>]*aria-label="[^"]*Urgent debunk: Compulsory contraceptive trial/
		);
	});

	it('draws each bubble inside the quadrant it is classed in', () => {
		const body = renderMatrix();
		// Splits: velocity 0.5 is mid-width; harm 2.5 sits 3/5 down the 0.5–5.5 axis.
		const midX = 240;
		const midY = 180;
		const bubbles = [
			...body.matchAll(
				/data-quadrant="(\w+)"><circle[^>]*\bcx="([\d.]+)"[^>]*\bcy="([\d.]+)"[^>]*\br="([\d.]+)"/g
			)
		];
		expect(bubbles).toHaveLength(7);
		for (const [, quadrant, cx, cy, r] of bubbles) {
			const [x, y, radius] = [Number(cx), Number(cy), Number(r)];
			const right = quadrant === 'urgent' || quadrant === 'campaign';
			const top = quadrant === 'urgent' || quadrant === 'critical';
			expect(x > midX, `${quadrant} x`).toBe(right);
			expect(y < midY, `${quadrant} y`).toBe(top);
			expect(x - radius).toBeGreaterThanOrEqual(0);
			expect(x + radius).toBeLessThanOrEqual(480);
		}
	});

	it('keeps the biggest bubble inside the chart at the extremes', () => {
		const body = renderMatrix([
			{ ...priorityPoints[0], velocity: 1 },
			{ ...priorityPoints[1], velocity: 0 }
		]);
		const circles = [...body.matchAll(/<circle[^>]*\bcx="([\d.]+)"[^>]*\br="([\d.]+)"/g)];
		for (const [, cx, r] of circles) {
			expect(Number(cx) - Number(r)).toBeGreaterThanOrEqual(0);
			expect(Number(cx) + Number(r)).toBeLessThanOrEqual(480);
		}
	});

	it('lists every bubble, linking narratives and describing each for screen readers', () => {
		const body = renderMatrix();
		expect(body).toContain('href="/narratives/N-0412"');
		expect(body).toContain('Menstrual stigma / hygiene taboos');
		expect(body).toContain(
			'(Urgent debunk: harm 4 of 5, spread velocity 90%, 2,031 messages in 7 days)'
		);
	});

	it('shows an empty state', () => {
		expect(renderMatrix([])).toContain('No active narratives.');
	});
});

describe('in French', () => {
	it('words the matrix and the knowledge gaps in French', () => {
		const [matrix, gaps] = withLocale('fr', () => [
			stripComments(render(PriorityMatrix, { props: { points: priorityPoints } }).body),
			render(KnowledgeGapList, {
				props: { gaps: knowledgeGaps, now: OPERATIONS_NOW, action: '?/gap' }
			}).body
		]);
		expect(matrix).toContain('Démenti urgent');
		expect(matrix).toMatch(
			/\(Démenti urgent\u00a0: préjudice 4 sur 5, vitesse de propagation 90%, 2\u202f031 messages en 7 jours\)/
		);
		expect(gaps).toContain('première demande');
		expect(gaps).toContain('il y a 3 j');
	});
});
