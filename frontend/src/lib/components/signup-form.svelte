<script lang="ts">
	import { Button } from '#lib/components/ui/button/index.js';
	import * as Field from '#lib/components/ui/field/index.js';
	import { Input } from '#lib/components/ui/input/index.js';
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
			<h1 class="text-2xl font-bold">Create your account</h1>
			<p class="text-sm text-balance text-muted-foreground">
				Fill in the form below to create your account
			</p>
		</div>
		{#if error}
			<p id="error-{id}" role="alert" class="text-sm font-medium text-destructive">{error}</p>
		{/if}
		<Field.Field>
			<Field.Label for="name-{id}">Full name</Field.Label>
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
			<Field.Label for="email-{id}">Email</Field.Label>
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
				We'll use this to contact you. We will not share your email with anyone else.
			</Field.Description>
		</Field.Field>
		<Field.Field>
			<Field.Label for="password-{id}">Password</Field.Label>
			<Input
				id="password-{id}"
				name="password"
				type="password"
				autocomplete="new-password"
				minlength={8}
				required
				aria-describedby="password-note-{id}"
			/>
			<Field.Description id="password-note-{id}"
				>Must be at least 8 characters long.</Field.Description
			>
		</Field.Field>
		<Field.Field>
			<Field.Label for="confirm-password-{id}">Confirm password</Field.Label>
			<Input
				id="confirm-password-{id}"
				name="confirmPassword"
				type="password"
				autocomplete="new-password"
				minlength={8}
				required
			/>
		</Field.Field>
		<Field.Field>
			<Button type="submit">Create account</Button>
			<Field.Description class="px-6 text-center">
				Already have an account?
				<a href={loginHref} class="underline underline-offset-4">Log in</a>
			</Field.Description>
		</Field.Field>
	</Field.Group>
</form>
