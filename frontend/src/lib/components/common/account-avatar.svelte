<script lang="ts">
	import { initials } from '#lib/domain/account.js';
	import { cn } from '#lib/utils.js';

	interface Props {
		name: string;
		/** `sm` for the header, `lg` for the profile card. */
		size?: 'sm' | 'lg';
		class?: string;
	}

	let { name, size = 'sm', class: className }: Props = $props();
</script>

<!-- Decorative: the name is always shown or announced next to it. -->
<span
	aria-hidden="true"
	data-size={size}
	class={cn(
		'grid shrink-0 place-items-center rounded-full bg-platform-whatsapp font-mono font-semibold text-white data-[size=lg]:size-14 data-[size=lg]:text-lg data-[size=sm]:size-8 data-[size=sm]:text-[0.71875rem]',
		className
	)}
>
	{initials(name)}
</span>
