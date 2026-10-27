import { describe, expect, it } from 'vitest';
import { render } from 'svelte/server';
import {
	MONITOR_NOW,
	monitorSnapshot,
	monitorWindows,
	processingRuns,
	runningRun
} from '#lib/fixtures/monitor.js';
import { withLocale } from '#lib/testing/locale.js';
import { stripComments } from '#lib/testing/snippet.js';
import MonitorView from './monitor-view.svelte';
import ProcessingVolumeChart from './processing-volume-chart.svelte';
import RunPlatformBreakdown from './run-platform-breakdown.svelte';
import RunStatusBreakdown from './run-status-breakdown.svelte';
import RunTable from './run-table.svelte';

describe('RunTable', () => {
	const renderTable = (props = {}) =>
		stripComments(
			render(RunTable, { props: { runs: processingRuns, now: MONITOR_NOW, ...props } }).body
		);

	it('lists each run with its time, type, author and numbers', () => {
		const body = renderTable();
		expect(body).toContain('<caption class="sr-only">Processing runs</caption>');
		expect(body).toContain('Today 14:00');
		expect(body).toContain('Yesterday 22:41');
		expect(body).toContain('data-type="manual"');
		expect(body).toContain('Admin 2');
		expect(body).toContain('System');
		expect(body).toContain('412');
		expect(body).toContain('23 min');
		expect(body).toContain('$2.84');
	});

	it('links each run and marks the selected one', () => {
		const body = renderTable({ selectedId: processingRuns[2].id });
		expect(body).toContain(`href="?run=${processingRuns[0].id}"`);
		expect(body.match(/aria-current="true"/g)).toHaveLength(1);
		expect(body).toMatch(new RegExp(`href="\\?run=${processingRuns[2].id}" aria-current="true"`));
	});

	it('shows a running run without numbers', () => {
		const body = renderTable({ runs: [runningRun] });
		expect(body).toContain('Running…');
		expect(body).toContain(
			'<span aria-hidden="true">…</span><span class="sr-only">pending until the run finishes</span>'
		);
		expect(body).not.toContain('$');
	});

	it('shows an empty state', () => {
		expect(renderTable({ runs: [] })).toContain('No runs in this window.');
	});
});

describe('RunStatusBreakdown', () => {
	it('counts the new posts by status, marking final ones', () => {
		const body = stripComments(
			render(RunStatusBreakdown, { props: { run: processingRuns[1] } }).body
		);
		expect(body).toContain('121 newly discovered posts');
		expect(body).not.toContain('still processing');
		expect(body).toMatch(/data-status="clustered"[\s\S]*?Clustered[\s\S]*?final/);
		expect(body).toMatch(/data-status="discovered" data-empty="true"/);
		expect(body).toContain('21 min');
		expect(body).toContain('$2.61');
		expect(body).toMatch(/\d+<span class="sr-only">(&nbsp;|\u00a0)new posts<\/span>/);
	});

	it('says when some posts are still processing', () => {
		const body = render(RunStatusBreakdown, { props: { run: processingRuns[0] } }).body;
		expect(body).toContain('138 newly discovered posts · some still processing');
	});

	it('waits for a running run', () => {
		const body = render(RunStatusBreakdown, { props: { run: runningRun } }).body;
		expect(body).toContain('Run in progress…');
		expect(body).toContain('No data');
	});

	it('reads French', () => {
		const body = withLocale(
			'fr',
			() => render(RunStatusBreakdown, { props: { run: processingRuns[1] } }).body
		);
		expect(body).toContain('121 nouveaux posts');
		expect(body).toContain('Regroupé');
	});
});

describe('RunPlatformBreakdown', () => {
	it('describes the split and tabulates it', () => {
		const counts = processingRuns[1].result!.platformCounts;
		const body = stripComments(render(RunPlatformBreakdown, { props: { counts } }).body);
		expect(body).toMatch(/aria-label="121 new posts by platform: Facebook \d+%, TikTok \d+%/);
		expect(body).toMatch(/<tfoot class="sr-only"><tr><th scope="row">Total<\/th><td>121<\/td>/);
		expect(body).toContain('Instagram');
		expect(body).toContain('YouTube');
	});

	it('shows dashes while the run is going', () => {
		const body = render(RunPlatformBreakdown, { props: { counts: null } }).body;
		expect(body).toContain('aria-label="No new posts yet"');
		expect(body).toContain('<span class="sr-only">pending until the run finishes</span>');
		expect(body).not.toContain('<tfoot');
	});
});

describe('ProcessingVolumeChart', () => {
	it('labels 2-hour buckets by clock time, one bar per bucket', () => {
		const { volume } = monitorWindows['24h'];
		const body = stripComments(render(ProcessingVolumeChart, { props: volume }).body);
		expect(body).toMatch(/role="toolbar"[^>]*aria-label="Posts processed and failed over time"/);
		expect(body).toContain('aria-label="14:00–16:00: 62 processed, 1 failed, 98% success"');
		expect(body.match(/<button/g)).toHaveLength(volume.buckets.length);
	});

	it('shows the newest bucket until one is picked, with a single tab stop', () => {
		const { volume } = monitorWindows['24h'];
		const body = stripComments(render(ProcessingVolumeChart, { props: volume }).body);
		expect(body.match(/tabindex="0"/g)).toHaveLength(1);
		expect(body).toMatch(/tabindex="0" aria-pressed="true" aria-label="12:00–14:00:/);
		expect(body).toMatch(/84\s+processed successfully/);
		// The class names mention data-dimmed too; only the attribute counts.
		expect(body).not.toMatch(/\sdata-dimmed(?=[\s=>])/);
	});

	it('labels daily buckets by date', () => {
		const { volume } = monitorWindows['7d'];
		const body = stripComments(render(ProcessingVolumeChart, { props: volume }).body);
		expect(body).toContain('aria-label="Oct 12: 1,086 processed, 21 failed, 98% success"');
	});

	it('shows an empty state', () => {
		const body = render(ProcessingVolumeChart, { props: { buckets: [], bucketHours: 2 } }).body;
		expect(body).toContain('No data in this window.');
	});
});

describe('MonitorView', () => {
	const renderView = (props = {}) =>
		stripComments(
			render(MonitorView, {
				props: {
					snapshot: monitorSnapshot,
					range: '24h' as const,
					rangeHref: (range: string) => `?range=${range}`,
					...props
				}
			}).body
		);

	it('shows the headline numbers for the window', () => {
		const body = renderView();
		expect(body).toMatch(/<h1[^>]*>\s*Monitor\s*<\/h1>/);
		expect(body).toContain('Content discovery &amp; processing · last 24 h');
		expect(body).toContain('762');
		expect(body).toContain('21 failed · 24 h');
		expect(body).toContain('138 newly discovered');
		expect(body).toContain('latest run 23 min');
		expect(body).toContain('$24.86');
		expect(body).toContain('Showing 11 of 12 runs');
		expect(body).toContain('<dl role="group" aria-label="Processing figures, last 24 h"');
	});

	it('inspects the newest run unless one is picked', () => {
		expect(renderView()).toContain('Posts in run Today 14:00');
		const picked = renderView({ selectedRunId: processingRuns[2].id });
		expect(picked).toContain('Posts in run Today 11:20');
		expect(picked).toContain('Today 11:20 · Manual · Admin 1');
	});

	it('names the picked run in the status heading and announces it', () => {
		const body = renderView({ selectedRunId: processingRuns[2].id });
		expect(body).toMatch(
			/<h2[^>]*><span>Processing status<\/span> <span class="sr-only">,<\/span> <span[^>]*>Today 11:20 · Manual · Admin 1<\/span><\/h2>/
		);
		expect(body).toContain(
			'<p class="sr-only" aria-live="polite">Showing details for the run at Today 11:20</p>'
		);
		expect(body).toContain('data-sveltekit-keepfocus');
	});

	it('posts "Run now" to the action and disables it while a run is going', () => {
		expect(renderView()).toMatch(/<form method="POST" action="\?\/run">/);
		const busy = renderView({
			snapshot: { ...monitorSnapshot, runs: [runningRun, ...processingRuns] }
		});
		expect(busy).toMatch(/<button[^>]*aria-disabled="true"[^>]*>[\s\S]*?Running…/);
		expect(busy).not.toMatch(/<button[^>]*\sdisabled[\s=>]/);
	});

	it('shows why a run could not start', () => {
		expect(renderView({ runError: 'A run is already in progress.' })).toContain('role="alert"');
	});
});
