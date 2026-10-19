/**
 * Sample data for the operations view, taken from the Infodemic Dashboard
 * design. Read-only; shared by stories, tests and stubs.
 */
import type {
	DebunkOutcome,
	ExpertCheck,
	KnowledgeGap,
	PriorityPoint
} from '#lib/domain/operations.js';
import { narratives, OVERVIEW_NOW } from './overview.js';

export const OPERATIONS_NOW = OVERVIEW_NOW;

const pending = { kind: 'pending' } as const;
const daysAgo = (days: number) => new Date(OPERATIONS_NOW.getTime() - days * 86_400_000);

export const expertChecks: ExpertCheck[] = [
	{
		id: 'C-0201',
		title: narratives[0].title,
		engine: {
			verdict: 'false',
			confidence: 0.91,
			explanation:
				'There is no compulsory contraceptive trial in Burkina Faso. No such trial is registered with the Ministry of Health. Contraception is always voluntary.'
		},
		reasons: ['spiking'],
		harm: 4,
		review: pending
	},
	{
		id: 'C-0202',
		title: 'Contraceptive injections cause cancer',
		engine: {
			verdict: 'misleading',
			confidence: 0.58,
			explanation:
				'Injectable contraceptives are not known to cause cancer. Research shows small changes in risk for some cancers and lower risk for others.'
		},
		reasons: ['low-confidence'],
		review: pending
	},
	{
		id: 'C-0203',
		title: 'Breastfeeding always prevents pregnancy',
		engine: {
			verdict: 'misleading',
			confidence: 0.71,
			explanation:
				'Breastfeeding delays fertility only if it is exclusive, the baby is under 6 months and periods have not returned.'
		},
		reasons: ['partly-true'],
		review: pending
	},
	{
		id: 'C-0204',
		title: 'Girls must not bathe during menstruation',
		engine: {
			verdict: 'false',
			confidence: 0.64,
			explanation: 'Washing during your period is safe and helps prevent infection.'
		},
		reasons: ['low-confidence'],
		note: 'Mooré audio',
		review: pending
	},
	{
		id: 'C-0205',
		title: narratives[4].title,
		engine: {
			verdict: 'unverified',
			confidence: 0.49,
			explanation:
				'We could not verify this claim. Condoms distributed by health centres are quality-tested before distribution.'
		},
		reasons: ['low-confidence'],
		review: pending
	}
];

const point = (
	id: string,
	index: number | null,
	velocity: number,
	harm: PriorityPoint['harm'],
	volume7d: number,
	title?: string
): PriorityPoint => ({
	id,
	narrativeId: index === null ? null : narratives[index].id,
	title: index === null ? title! : narratives[index].title,
	harm,
	velocity,
	volume7d
});

export const priorityPoints: PriorityPoint[] = [
	point('p1', 0, 0.9, 4, 2031),
	point('p2', 2, 0.62, 5, 1180),
	point('p3', 4, 0.66, 3, 820),
	point('p4', 1, 0.36, 3, 1920),
	point('p5', 3, 0.18, 3, 1640),
	point('p6', null, 0.76, 2, 610, 'Menstrual stigma / hygiene taboos'),
	point('p7', null, 0.24, 1, 410, 'Fertility & diet myths')
];

export const knowledgeGaps: KnowledgeGap[] = [
	{
		id: 'G-031',
		label: 'Sayana Press causes weight gain and stops periods forever',
		example:
			'Is it true the injection in the arm makes you fat and you never see your period again?',
		messageCount: 288,
		languages: ['Mooré', 'French'],
		firstAskedAt: daysAgo(3),
		review: pending
	},
	{
		id: 'G-030',
		label: 'Moringa tea prevents pregnancy',
		example: 'My aunt says moringa tea every morning is enough so you will not get pregnant.',
		messageCount: 164,
		languages: ['Dioula', 'Mooré'],
		firstAskedAt: daysAgo(2),
		review: pending
	},
	{
		id: 'G-028',
		label: 'Fertility does not return after implant removal',
		example: 'After they remove the implant, can I still have children?',
		messageCount: 150,
		languages: ['French'],
		firstAskedAt: daysAgo(6),
		review: pending
	},
	{
		id: 'G-029',
		label: 'HIV self-test kits always show negative',
		example: 'The HIV test you do yourself is fake, it always says negative.',
		messageCount: 120,
		languages: ['French', 'Dioula'],
		firstAskedAt: daysAgo(4),
		review: pending
	},
	{
		id: 'G-032',
		label: 'Lemon juice works as a spermicide',
		example: 'If you put lemon inside before, the sperm dies.',
		messageCount: 72,
		languages: ['Mooré'],
		firstAskedAt: daysAgo(1),
		review: pending
	}
];

export const debunkOutcomes: DebunkOutcome[] = [
	{
		id: 'D-014',
		title: 'Implants do not cause permanent infertility',
		formats: 'Audio · Mooré, Dioula',
		sent: 3412,
		helpfulPct: 78,
		followUpPct: 12
	},
	{
		id: 'D-011',
		title: 'HPV vaccine is safe for girls',
		formats: 'Text · French, Audio · Mooré',
		sent: 2210,
		helpfulPct: 69,
		followUpPct: 17
	},
	{
		id: 'D-009',
		title: 'Condom myths explained',
		formats: 'Text · French',
		sent: 960,
		helpfulPct: 52,
		followUpPct: 31
	},
	{
		id: 'D-016',
		title: 'Potash drinks are dangerous',
		formats: 'Audio · Mooré',
		sent: 418,
		helpfulPct: 81,
		followUpPct: 9
	}
];
