<script lang="ts">
	import EyeIcon from '@lucide/svelte/icons/eye';
	import EyeOffIcon from '@lucide/svelte/icons/eye-off';
	import { PASSWORD_MIN_LENGTH } from '#lib/domain/account.js';
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

	let showPassword = $state(false);
	let showConfirmPassword = $state(false);
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
					aria-label={showPassword ? 'Hide password' : 'Show password'}
				>
					{#if showPassword}
						<EyeOffIcon class="size-4" />
					{:else}
						<EyeIcon class="size-4" />
					{/if}
				</Button>
			</div>
			<Field.Description id="password-note-{id}"
				>Must be at least {PASSWORD_MIN_LENGTH} characters long.</Field.Description
			>
		</Field.Field>
		<Field.Field>
			<Field.Label for="confirm-password-{id}">Confirm password</Field.Label>
			<div class="relative">
				<Input
					id="confirm-password-{id}"
					name="confirmPassword"
					type={showConfirmPassword ? 'text' : 'password'}
					autocomplete="new-password"
					minlength={PASSWORD_MIN_LENGTH}
					required
					class="pr-10"
				/>
				<Button
					type="button"
					variant="ghost"
					size="icon-sm"
					class="absolute top-1/2 right-1 -translate-y-1/2 text-muted-foreground hover:text-foreground"
					onclick={() => (showConfirmPassword = !showConfirmPassword)}
					aria-label={showConfirmPassword ? 'Hide confirm password' : 'Show confirm password'}
				>
					{#if showConfirmPassword}
						<EyeOffIcon class="size-4" />
					{:else}
						<EyeIcon class="size-4" />
					{/if}
				</Button>
			</div>
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
