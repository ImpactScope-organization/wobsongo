<script lang="ts">
	import EyeIcon from '@lucide/svelte/icons/eye';
	import EyeOffIcon from '@lucide/svelte/icons/eye-off';
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
			<div class="relative">
				<Input
					id="password-{id}"
					name="password"
					type={showPassword ? 'text' : 'password'}
					autocomplete="current-password"
					required
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
