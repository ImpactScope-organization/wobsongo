<script lang="ts">
	import type { AccountProfile } from '#lib/domain/account.js';
	import type { FormEnhance } from '#lib/utils/enhance.js';
	import PageHeader from '../common/page-header.svelte';
	import SectionCard from '../common/section-card.svelte';
	import BotComingSoon from './bot-coming-soon.svelte';
	import PasswordForm from './password-form.svelte';
	import ProfileForm from './profile-form.svelte';
	import { SECONDARY } from './styles';

	interface Props {
		profile: AccountProfile;
		/** When the password last changed; null if never. */
		passwordChangedAt: Date | null;
		/** Pass the injected clock's now. */
		now: Date;
		/** Outcome of the last profile save. */
		profileResult?: {
			values?: { name?: string; email?: string };
			error?: string;
			saved?: boolean;
		};
		/** Outcome of the last password change or reset request. */
		passwordResult?: { error?: string; updated?: boolean; resetSentTo?: string };
		/** Form action that signs out. */
		logoutAction?: string;
		enhance?: FormEnhance;
	}

	let {
		profile,
		passwordChangedAt,
		now,
		profileResult = {},
		passwordResult = {},
		logoutAction = '/logout',
		enhance
	}: Props = $props();
</script>

<main class="mx-auto flex max-w-5xl flex-col gap-5 p-[clamp(1rem,3vw,2rem)]">
	<PageHeader title="Settings" eyebrow="Your account" class="mb-0.5">
		{#snippet actions()}
			<!-- Signing out changes state, so it is a POST, never a link. -->
			<form method="POST" action={logoutAction}>
				<button type="submit" class={SECONDARY}>Log out</button>
			</form>
		{/snippet}
	</PageHeader>

	<div class="mx-auto flex w-full max-w-5xl flex-col gap-5">
		<SectionCard title="Account" description="Your profile as seen by other reviewers.">
			<ProfileForm
				{profile}
				action="?/profile"
				values={profileResult.values}
				error={profileResult.error}
				saved={profileResult.saved}
				{enhance}
			/>
		</SectionCard>

		<SectionCard title="Password">
			<PasswordForm
				action="?/password"
				resetAction="?/resetPassword"
				changedAt={passwordChangedAt}
				{now}
				error={passwordResult.error}
				updated={passwordResult.updated}
				resetSentTo={passwordResult.resetSentTo}
				{enhance}
			/>
		</SectionCard>

		<SectionCard
			title="Bot connections"
			description="Link your own messaging account to the bot to review flagged claims and approve replies from your phone."
			flush
		>
			<BotComingSoon />
		</SectionCard>
	</div>
</main>
