import { describe, expect, it } from 'vitest';
import { render } from 'svelte/server';
import LoginForm from './login-form.svelte';
import SignupForm from './signup-form.svelte';

describe('LoginForm', () => {
	it('posts named fields, so it works without JavaScript', () => {
		const { body } = render(LoginForm, { props: { action: '?/login' } });
		expect(body).toMatch(/<form[^>]*method="POST"[^>]*action="\?\/login"/);
		expect(body).toMatch(/<input(?=[^>]*name="email")(?=[^>]*type="email")/);
		expect(body).toMatch(/<input(?=[^>]*name="password")(?=[^>]*type="password")/);
		expect(body).toContain('type="submit"');
	});

	it('refills the email and announces the error after a failed attempt', () => {
		const { body } = render(LoginForm, {
			props: { email: 'a@b.org', error: 'Wrong email or password.' }
		});
		expect(body).toContain('value="a@b.org"');
		expect(body).toMatch(/role="alert"[^>]*>Wrong email or password\.</);
	});

	it('links to sign-up and has no dead links', () => {
		const { body } = render(LoginForm, {});
		expect(body).toContain('href="/signup"');
		expect(body).not.toMatch(/href="#/);
	});
});

describe('SignupForm', () => {
	it('posts named fields, never refilling passwords', () => {
		const { body } = render(SignupForm, {
			props: { values: { name: 'Awa', email: 'awa@example.org' } }
		});
		expect(body).toMatch(/<form[^>]*method="POST"/);
		for (const name of ['name', 'email', 'password', 'confirmPassword']) {
			expect(body).toContain(`name="${name}"`);
		}
		expect(body).toContain('value="Awa"');
		expect(body).not.toMatch(/<input(?=[^>]*type="password")(?=[^>]*value="[^"]+")/);
	});

	it('uses unique ids, so two forms on one page do not clash', () => {
		const { body } = render(SignupForm, {});
		const ids = [...body.matchAll(/\sid="([^"]+)"/g)].map((match) => match[1]);
		expect(ids).not.toContain('email');
		expect(new Set(ids).size).toBe(ids.length);
	});
});
