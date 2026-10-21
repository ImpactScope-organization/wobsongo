import { describe, expect, it } from 'vitest';
import { render } from 'svelte/server';
import RiskGauge from './risk-gauge.svelte';

describe('RiskGauge', () => {
	it('renders with meter role and value attributes', () => {
		const { body } = render(RiskGauge, { props: { value: 75 } });
		expect(body).toContain('role="meter"');
		expect(body).toContain('aria-valuenow="75"');
		expect(body).toContain('aria-valuemin="0"');
		expect(body).toContain('aria-valuemax="100"');
		expect(body).toContain('75');
	});

	it('assigns high risk tier for values >= 70 (red)', () => {
		const { body } = render(RiskGauge, { props: { value: 85 } });
		expect(body).toContain('High');
		expect(body).toContain('text-verdict-false-strong');
	});

	it('assigns medium risk tier for values between 40 and 69 (orange/amber)', () => {
		const { body } = render(RiskGauge, { props: { value: 55 } });
		expect(body).toContain('Medium');
		expect(body).toContain('text-verdict-misleading-foreground');
	});

	it('assigns low risk tier for values < 40 (green)', () => {
		const { body } = render(RiskGauge, { props: { value: 20 } });
		expect(body).toContain('Low');
		expect(body).toContain('text-verdict-accurate-strong');
	});

	it('clamps values below 0 and above 100', () => {
		const low = render(RiskGauge, { props: { value: -15 } }).body;
		expect(low).toContain('aria-valuenow="0"');
		expect(low).toContain('0');

		const high = render(RiskGauge, { props: { value: 150 } }).body;
		expect(high).toContain('aria-valuenow="100"');
		expect(high).toContain('100');
	});

	it('renders stroke-dashoffset matching the clamped percentage', () => {
		// size=36, strokeWidth=3 => radius=16.5 => circumference = 2 * PI * 16.5 ≈ 103.67
		const { body } = render(RiskGauge, { props: { value: 50, size: 36, strokeWidth: 3 } });
		expect(body).toContain('stroke-dasharray="103.67');
		expect(body).toContain('stroke-dashoffset="51.83');
	});
});
