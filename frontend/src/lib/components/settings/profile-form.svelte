<script lang="ts">
	import type { AccountProfile } from '#lib/domain/account.js';
	import { optionalEnhance, type FormEnhance } from '#lib/utils/enhance.js';
	import { cn } from '#lib/utils.js';
	import AccountAvatar from '../common/account-avatar.svelte';
	import { ACCOUNT_ROLE_LABEL } from '../common/labels';
	import { Input } from '../ui/input/index.js';
	import { FIELD, INPUT, LABEL, PRIMARY } from './styles';

	interface Props {
		profile: AccountProfile;
		/** Form action URL, e.g. "?/profile". Posts `name` and `email`. */
		action: string;
		/** Values to refill after a failed save. */
		values?: { name?: string; email?: string };
		/** Why the last save failed, from the server. */
		error?: string;
		/** The last save succeeded. */
		saved?: boolean;
		enhance?: FormEnhance;
		class?: string;
	}

	let {
		profile,
		action,
		values = {},
		error,
		saved = false,
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
	class={cn('flex flex-col', className)}
	aria-describedby={error ? `error-${id}` : undefined}
>
	<div class="mb-[1.125rem] flex flex-wrap items-center gap-4 border-b border-rule pb-[1.125rem]">
		<AccountAvatar name={profile.name} size="lg" />
		<div class="min-w-44 flex-1">
			<p class="text-base font-semibold">{profile.name}</p>
			<p class="text-[0.8125rem] text-muted-foreground">{profile.email}</p>
		</div>
		<span
			class="rounded bg-status-verified px-2 py-1 font-mono text-[0.65625rem] font-semibold tracking-[0.05em] text-status-verified-foreground uppercase"
		>
			{ACCOUNT_ROLE_LABEL[profile.role]}
		</span>
	</div>

	{#if error}
		<p id="error-{id}" role="alert" class="mb-3.5 text-[0.8125rem] font-medium text-destructive">
			{error}
		</p>
	{/if}

	<div
		class="grid grid-cols-[repeat(auto-fit,minmax(min(100%,15rem),1fr))] gap-x-[1.125rem] gap-y-3.5"
	>
		<div class={FIELD}>
			<label for="name-{id}" class={LABEL}>Full name</label>
			<Input
				id="name-{id}"
				name="name"
				autocomplete="name"
				value={values.name ?? profile.name}
				required
				class={INPUT}
			/>
		</div>
		<div class={FIELD}>
			<label for="email-{id}" class={LABEL}>Email</label>
			<Input
				id="email-{id}"
				name="email"
				type="email"
				autocomplete="email"
				value={values.email ?? profile.email}
				required
				class={INPUT}
			/>
		</div>
	</div>

	<div class="mt-[1.125rem] flex items-center justify-end gap-3">
		{#if saved}
			<span role="status" class="text-[0.8125rem] font-medium text-status-verified-foreground">
				✓ Changes saved
			</span>
		{/if}
		<button type="submit" class={PRIMARY}>Save changes</button>
	</div>
</form>
