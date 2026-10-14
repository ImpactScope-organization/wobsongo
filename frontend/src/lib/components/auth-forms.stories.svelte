<script module lang="ts">
	import { defineMeta } from '@storybook/addon-svelte-csf';
	import { expect, userEvent, within } from 'storybook/test';
	import LoginForm from './login-form.svelte';
	import SignupForm from './signup-form.svelte';

	const { Story } = defineMeta({
		title: 'Auth/Forms',
		component: LoginForm,
		tags: ['autodocs'],
		parameters: { layout: 'centered' },
		args: { action: '?/login' }
	});
</script>

<Story
	name="Log in"
	play={async ({ canvasElement }) => {
		const canvas = within(canvasElement);
		const form = canvasElement.querySelector('form')!;
		let posted: FormData | undefined;
		form.addEventListener('submit', (event) => {
			event.preventDefault();
			posted = new FormData(form);
		});

		await userEvent.type(canvas.getByLabelText('Email'), 'awa@example.org');
		await userEvent.type(canvas.getByLabelText('Password'), 'correct horse');
		await userEvent.click(canvas.getByRole('button', { name: 'Log in' }));
		await expect(posted?.get('email')).toBe('awa@example.org');
		await expect(posted?.get('password')).toBe('correct horse');
	}}
>
	{#snippet template(args)}
		<LoginForm {...args} class="w-80" />
	{/snippet}
</Story>

<Story
	name="Log in, failed"
	args={{ email: 'awa@example.org', error: 'Wrong email or password.' }}
	play={async ({ canvasElement }) => {
		await expect(within(canvasElement).getByRole('alert')).toHaveTextContent(
			'Wrong email or password.'
		);
	}}
>
	{#snippet template(args)}
		<LoginForm {...args} class="w-80" />
	{/snippet}
</Story>

<Story name="Sign up">
	{#snippet template()}
		<SignupForm action="?/signup" class="w-80" />
	{/snippet}
</Story>
