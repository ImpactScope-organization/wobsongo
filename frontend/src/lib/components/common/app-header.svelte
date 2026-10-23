<script lang="ts">
	import type { Snippet } from 'svelte';
	import { resolveNavIcon, type NavItem } from '#lib/navigation.js';
	import { cn } from '#lib/utils.js';
	import AccountAvatar from './account-avatar.svelte';
	import DockMenu from './dock-menu.svelte';
	import { m } from '#lib/paraglide/messages.js';
	import LanguageSwitcher from './language-switcher.svelte';

	interface Props {
		/** Logo image URL; omitted shows the name only. */
		logoSrc?: string;
		items: readonly NavItem[];
		/** href of the current section, matched against `items`. */
		currentHref: string;
		/** Right-hand controls, e.g. an export link. */
		actions?: Snippet;
		/** The signed-in person; shows their avatar linking to settings. */
		account?: { name: string; href: string };
		/** This page's path and query, where switching language comes back to. */
		returnTo?: string;
		/** Include the bottom docker menu on mobile screen. Defaults to false. */
		mobileDock?: boolean;
		class?: string;
	}

	let {
		logoSrc,
		items,
		currentHref,
		actions,
		account,
		returnTo,
		mobileDock = false,
		class: className
	}: Props = $props();
</script>

<header class={cn('bg-inverse px-[clamp(1rem,3vw,2rem)] text-inverse-foreground', className)}>
	<div class="mx-auto flex min-h-14 max-w-[85rem] flex-wrap items-center gap-x-5 gap-y-2 py-2">
		<a
			href="/"
			aria-label={m.nav_home_label()}
			class="flex items-center gap-2.5 rounded-sm outline-none focus-visible:ring-2 focus-visible:ring-inverse-signal"
		>
			{#if logoSrc}
				<img src={logoSrc} alt="" class="h-[20px] w-auto" />
			{/if}
		</a>

		<nav
			aria-label={m.nav_main_label()}
			class="order-last hidden min-w-0 basis-full overflow-x-auto md:order-0 md:block md:flex-1 md:basis-auto"
		>
			<ul class="flex gap-0.5">
				{#each items as item (item.href)}
					{@const Icon = resolveNavIcon(item)}
					<li>
						{#if item.disabled}
							<span
								aria-disabled="true"
								title={m.nav_coming_soon()}
								class="flex cursor-not-allowed items-center gap-1.5 rounded-md px-3 py-2 text-[0.8125rem] font-medium whitespace-nowrap text-inverse-muted-foreground opacity-60"
							>
								<Icon class="size-4 shrink-0 stroke-[1.75]" aria-hidden="true" />
								{item.label}<span class="sr-only">&nbsp;{m.nav_coming_soon_suffix()}</span>
							</span>
						{:else}
							<a
								href={item.href}
								aria-current={item.href === currentHref ? 'page' : undefined}
								class="flex items-center gap-1.5 rounded-md px-3 py-2 text-[0.8125rem] font-medium whitespace-nowrap transition-colors outline-none hover:bg-inverse-raised focus-visible:ring-2 focus-visible:ring-inverse-signal aria-[current=page]:bg-inverse-border"
							>
								<Icon class="size-4 shrink-0 stroke-[1.75]" aria-hidden="true" />
								{item.label}
								{#if item.count}
									<span
										class="rounded-full bg-signal px-1.5 py-px font-mono text-[0.625rem] font-semibold text-signal-foreground"
									>
										{item.count}<span class="sr-only">&nbsp;{m.nav_count_open()}</span>
									</span>
								{/if}
							</a>
						{/if}
					</li>
				{/each}
			</ul>
		</nav>
		<div class="ml-auto flex items-center gap-2.5">
			<LanguageSwitcher {returnTo} />

			{@render actions?.()}
			{#if account}
				<a
					href={account.href}
					aria-current={account.href === currentHref ? 'page' : undefined}
					title={m.nav_settings()}
					class="rounded-full border-2 border-transparent outline-none focus-visible:ring-2 focus-visible:ring-inverse-signal aria-[current=page]:border-inverse-foreground"
				>
					<AccountAvatar name={account.name} />
					<span class="sr-only">{m.nav_settings_for({ name: account.name })}</span>
				</a>
			{/if}
		</div>
	</div>
</header>

{#if mobileDock}
	<DockMenu {items} {currentHref} />
{/if}
