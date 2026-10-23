<script lang="ts">
	import EyeIcon from '@lucide/svelte/icons/eye';
	import EyeOffIcon from '@lucide/svelte/icons/eye-off';
	import { PASSWORD_MIN_LENGTH } from '#lib/domain/account.js';
	import { Button } from '#lib/components/ui/button/index.js';
	import * as Field from '#lib/components/ui/field/index.js';
	import { Input } from '#lib/components/ui/input/index.js';
	import { m } from '#lib/paraglide/messages.js';
	import { optionalEnhance, type FormEnhance } from '#lib/utils/enhance.js';
	import { cn } from '#lib/utils.js';

	interface Props {
		/** Form action URL, e.g. "?/signup". Posts `name`, `email`, `password` and `confirmPassword`. */
		action?: string;
		/** Values to refill after a failed attempt. Passwords are never refilled. */
		values?: { name?: string; email?: string };
		/** Why the last attempt failed, from the server. */
		error?: string;
		loginHref?: string;
		enhance?: FormEnhance;
		class?: string;
	}

	let {
		action = '',
		values = {},
		error,
		loginHref = '/login',
		enhance,
		class: className
	}: Props = $props();

	const id = $props.id();

	let showPassword = $state(false);
</script>

<!-- A plain POST form: works without JavaScript; `enhance` upgrades it. -->
<form
	method="POST"
	{action}
	use:optionalEnhance={enhance}
	class={cn('flex flex-col gap-6', className)}
	aria-describedby={error ? `error-${id}` : undefined}
>
	<Field.Group>
		<div class="flex flex-col items-center gap-1 text-center">
			<h1 class="text-2xl font-bold">{m.signup_heading()}</h1>
			<p class="text-sm text-balance text-muted-foreground">{m.signup_intro()}</p>
		</div>
		{#if error}
			<p id="error-{id}" role="alert" class="text-sm font-medium text-destructive">{error}</p>
		{/if}
		<Field.Field>
			<Field.Label for="name-{id}">{m.field_full_name()}</Field.Label>
			<Input
				id="name-{id}"
				name="name"
				type="text"
				autocomplete="name"
				value={values.name ?? ''}
				required
			/>
		</Field.Field>
		<Field.Field>
			<Field.Label for="email-{id}">{m.field_email()}</Field.Label>
			<Input
				id="email-{id}"
				name="email"
				type="email"
				autocomplete="email"
				placeholder="m@example.com"
				value={values.email ?? ''}
				required
				aria-describedby="email-note-{id}"
			/>
			<Field.Description id="email-note-{id}">
				{m.signup_email_note()}
			</Field.Description>
		</Field.Field>
		<Field.Field>
			<Field.Label for="password-{id}">{m.field_password()}</Field.Label>
			<div class="relative">
				<Input
					id="password-{id}"
					name="password"
					type={showPassword ? 'text' : 'password'}
					autocomplete="new-password"
					minlength={PASSWORD_MIN_LENGTH}
					required
					aria-describedby="password-note-{id}"
					class="pr-10"
				/>
				<Button
					type="button"
					variant="ghost"
					size="icon-sm"
					class="absolute top-1/2 right-1 -translate-y-1/2 text-muted-foreground hover:text-foreground"
					onclick={() => (showPassword = !showPassword)}
					aria-label={showPassword ? m.password_hide() : m.password_show()}
				>
					{#if showPassword}
						<EyeOffIcon class="size-4" />
					{:else}
						<EyeIcon class="size-4" />
					{/if}
				</Button>
			</div>
			<Field.Description id="password-note-{id}">
				{m.signup_password_note({ min: PASSWORD_MIN_LENGTH })}
			</Field.Description>
		</Field.Field>
		<Field.Field>
			<Field.Label for="confirm-password-{id}">{m.field_confirm_password()}</Field.Label>
			<Input
				id="confirm-password-{id}"
				name="confirmPassword"
				type={showPassword ? 'text' : 'password'}
				autocomplete="new-password"
				minlength={PASSWORD_MIN_LENGTH}
				required
			/>
		</Field.Field>
		<Field.Field>
			<Button type="submit">{m.signup_submit()}</Button>
			<Field.Description class="px-6 text-center">
				{m.signup_have_account()}
				<a href={loginHref} class="underline underline-offset-4">{m.signup_login_link()}</a>
			</Field.Description>
		</Field.Field>
	</Field.Group>
</form>
