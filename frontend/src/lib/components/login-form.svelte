<script lang="ts">
	import { Button } from '#lib/components/ui/button/index.js';
	import {
		FieldGroup,
		Field,
		FieldLabel,
		FieldDescription
	} from '#lib/components/ui/field/index.js';
	import { Input } from '#lib/components/ui/input/index.js';
	import { optionalEnhance, type FormEnhance } from '#lib/utils/enhance.js';
	import { cn } from '#lib/utils.js';

	interface Props {
		/** Form action URL, e.g. "?/login". Posts `email` and `password`. */
		action?: string;
		/** Email to refill after a failed attempt. */
		email?: string;
		/** Why the last attempt failed, from the server. */
		error?: string;
		signupHref?: string;
		enhance?: FormEnhance;
		class?: string;
	}

	let {
		action = '',
		email = '',
		error,
		signupHref = '/signup',
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
	<FieldGroup>
		<div class="flex flex-col items-center gap-1 text-center">
			<h1 class="text-2xl font-bold">Log in to your account</h1>
			<p class="text-sm text-balance text-muted-foreground">
				Enter your email and password to continue
			</p>
		</div>
		{#if error}
			<p id="error-{id}" role="alert" class="text-sm font-medium text-destructive">{error}</p>
		{/if}
		<Field>
			<FieldLabel for="email-{id}">Email</FieldLabel>
			<Input
				id="email-{id}"
				name="email"
				type="email"
				autocomplete="email"
				placeholder="m@example.com"
				value={email}
				required
			/>
		</Field>
		<Field>
			<FieldLabel for="password-{id}">Password</FieldLabel>
			<Input
				id="password-{id}"
				name="password"
				type="password"
				autocomplete="current-password"
				required
			/>
		</Field>
		<Field>
			<Button type="submit">Log in</Button>
			<FieldDescription class="text-center">
				Don't have an account?
				<a href={signupHref} class="underline underline-offset-4">Sign up</a>
			</FieldDescription>
		</Field>
	</FieldGroup>
</form>
