import type { Preview } from '@storybook/sveltekit';
import { isLocale, setLocale } from '../src/lib/paraglide/runtime.js';
import '../src/routes/layout.css';

const preview: Preview = {
	parameters: {
		controls: {
			matchers: {
				color: /(background|color)$/i,
				date: /Date$/i
			}
		},
		a11y: {
			// 'todo' reports violations in the a11y panel without failing tests.
			test: 'todo'
		}
	},
	globalTypes: {
		locale: {
			description: 'Interface language',
			toolbar: {
				icon: 'globe',
				items: [
					{ value: 'fr', title: 'Français' },
					{ value: 'en', title: 'English' }
				],
				dynamicTitle: true
			}
		}
	},
	// Play functions read English, like the server tests; switch in the toolbar.
	initialGlobals: { locale: 'en' },
	decorators: [
		(Story, { globals }) => {
			if (isLocale(globals.locale)) setLocale(globals.locale, { reload: false });
			return Story();
		}
	]
};

export default preview;
