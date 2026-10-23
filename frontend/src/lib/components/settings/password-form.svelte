<script lang="ts">
	import { PASSWORD_MIN_LENGTH } from '#lib/domain/account.js';
	import { m } from '#lib/paraglide/messages.js';
	import { optionalEnhance, type FormEnhance } from '#lib/utils/enhance.js';
	import { formatSince } from '#lib/utils/format.js';
	import { cn } from '#lib/utils.js';
	import { Input } from '../ui/input/index.js';
	import { FIELD, INPUT, LABEL, PRIMARY, TEXT_LINK } from './styles';

	interface Props {
		/** Form action URL, e.g. "?/password". Posts `currentPassword`, `newPassword`, `confirmPassword`. */
		action: string;
		/** Form action URL that emails a reset link, e.g. "?/resetPassword". Posts nothing. */
		resetAction: string;
		/** When the password last changed; null if never. */
		changedAt: Date | null;
		/** Pass the injected clock's now. */
		now: Date;
		/** Why the last change failed, from the server. */
		error?: string;
		/** The last change succeeded. */
		updated?: boolean;
		/** Set once a reset link was sent, to this address. */
		resetSentTo?: string;
		enhance?: FormEnhance;
		class?: string;
	}

	let {
		action,
		resetAction,
		changedAt,
		now,
		error,
		updated = false,
		resetSentTo,
		enhance,
		class: className
	}: Props = $props();

	const id = $props.id();
	const rule = $derived(m.settings_password_rule({ min: PASSWORD_MIN_LENGTH }));
</script>

<div class={className}>
	<p class="mb-[1.125rem] text-[0.8125rem] text-muted-foreground">
		{#if changedAt}
			{m.settings_password_last_changed()}
			<time datetime={changedAt.toISOString()}>{formatSince(changedAt, now)}</time>.
		{:else}
			{m.settings_password_never_changed()}
		{/if}
	</p>

	<!-- A plain POST form: works without JavaScript; `enhance` upgrades it. -->
	<form
		method="POST"
		{action}
		use:optionalEnhance={enhance}
		aria-describedby={error ? `error-${id}` : undefined}
	>
		{#if error}
			<p id="error-{id}" role="alert" class="mb-3.5 text-[0.8125rem] font-medium text-destructive">
				{error}
			</p>
		{/if}

		<div
			class="grid grid-cols-[repeat(auto-fit,minmax(min(100%,13.75rem),1fr))] gap-x-[1.125rem] gap-y-3.5"
		>
			<div class={FIELD}>
				<label for="current-{id}" class={LABEL}>{m.settings_current_password()}</label>
				<Input
					id="current-{id}"
					name="currentPassword"
					type="password"
					autocomplete="current-password"
					required
					class={INPUT}
				/>
			</div>
			<div class={FIELD}>
				<label for="new-{id}" class={LABEL}>{m.settings_new_password()}</label>
				<Input
					id="new-{id}"
					name="newPassword"
					type="password"
					autocomplete="new-password"
					minlength={PASSWORD_MIN_LENGTH}
					placeholder={rule}
					aria-describedby="rule-{id}"
					required
					class={INPUT}
				/>
				<span id="rule-{id}" class="sr-only">{rule}</span>
			</div>
			<div class={FIELD}>
				<label for="confirm-{id}" class={LABEL}>{m.settings_confirm_new_password()}</label>
				<Input
					id="confirm-{id}"
					name="confirmPassword"
					type="password"
					autocomplete="new-password"
					minlength={PASSWORD_MIN_LENGTH}
					required
					class={INPUT}
				/>
			</div>
		</div>

		<div class="mt-[1.125rem] flex flex-wrap items-center justify-end gap-3">
			<div class="flex w-full flex-wrap items-center gap-2.5">
				<!-- Submits the separate reset form below, so passwords are never sent with it. -->
				<button
					type="submit"
					form="reset-{id}"
					class={cn(TEXT_LINK, 'w-full py-1 text-center md:text-right')}
				>
					{m.settings_forgot_password()}
				</button>
				{#if resetSentTo}
					<span
						role="status"
						class="w-full text-[0.78125rem] font-medium text-status-verified-foreground"
					>
						{m.settings_reset_sent({ email: resetSentTo })}
					</span>
				{/if}
			</div>
			<div class="flex w-full items-center gap-3">
				{#if updated}
					<span role="status" class="text-[0.8125rem] font-medium text-status-verified-foreground">
						{m.settings_password_updated()}
					</span>
				{/if}
				<button type="submit" class={cn(PRIMARY, 'ml-auto w-full md:w-auto')}
					>{m.settings_update_password()}</button
				>
			</div>
		</div>
	</form>

	<form id="reset-{id}" method="POST" action={resetAction} use:optionalEnhance={enhance}></form>
</div>
