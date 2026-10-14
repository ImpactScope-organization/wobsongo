/**
 * Sample data for the high-harm view, taken from the Infodemic Dashboard
 * design. Read-only; shared by stories, tests and stubs.
 */
import type { HarmAlert, SubstanceMention } from '#lib/domain/high-harm.js';
import { activity, PULSE_NOW } from './pulse.js';

export const HIGH_HARM_NOW = PULSE_NOW;

export const harmAlerts: HarmAlert[] = [
	{
		id: 'H-0091',
		quote:
			'A woman in our quarter said drinking potash with lemon will remove the pregnancy in two days.',
		substance: 'Potash (kanwa) + lemon · ingestion',
		risk: 'critical',
		riskReviewed: false,
		platform: 'whatsapp',
		channel: 'voice note',
		language: 'Mooré',
		region: 'Centre-Nord',
		similarCount: 14,
		detectedAt: new Date('2026-10-12T13:27:00Z'),
		engine: {
			verdict: 'false',
			confidence: 0.94,
			explanation:
				'Drinking potash does not safely end a pregnancy and can cause serious poisoning. Unsafe methods put lives at risk.'
		},
		review: { kind: 'pending' }
	},
	{
		id: 'H-0090',
		quote: 'If you wash inside with bleach after, you cannot get pregnant.',
		substance: 'Bleach · vaginal douching',
		risk: 'critical',
		riskReviewed: false,
		platform: 'facebook',
		channel: 'public group',
		language: 'Dioula',
		region: 'Hauts-Bassins',
		similarCount: 6,
		detectedAt: new Date('2026-10-12T12:05:00Z'),
		engine: {
			verdict: 'false',
			confidence: 0.89,
			explanation:
				'Bleach does not prevent pregnancy. Putting bleach inside the body causes burns and infections.'
		},
		review: { kind: 'pending' }
	},
	{
		id: 'H-0088',
		quote: 'Take ten malaria tablets at once to bring your period back.',
		substance: 'Antimalarials · overdose',
		risk: 'high',
		riskReviewed: true,
		platform: 'whatsapp',
		channel: 'text',
		language: 'French',
		region: 'Centre',
		similarCount: 3,
		detectedAt: new Date('2026-10-12T09:05:00Z'),
		engine: {
			verdict: 'false',
			confidence: 0.92,
			explanation:
				'Taking many antimalarial tablets at once does not bring back periods and can be fatal.'
		},
		review: { kind: 'pending' }
	}
];

export const substances: SubstanceMention[] = [
	{ id: 'potash', name: 'Potash (kanwa)', practice: 'Ingestion', count: 21 },
	{ id: 'herbal', name: 'Herbal concoctions', practice: 'Ingestion', count: 12 },
	{ id: 'bleach', name: 'Bleach', practice: 'Douching', count: 9 },
	{ id: 'antimalarials', name: 'Antimalarials', practice: 'Overdose', count: 5 },
	{ id: 'objects', name: 'Objects inserted', practice: 'Invasive procedure', count: 3 }
];

/** The high-harm activity log; the same events the Pulse view shows. */
export const highHarmActivity = activity;
