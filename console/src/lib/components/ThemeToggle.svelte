<script lang="ts">
	import Moon from '$lib/icons/Moon.svelte';
	import Sun from '$lib/icons/Sun.svelte';
	import { onMount } from 'svelte';

	let dark: boolean;
	let hidden = true;

	onMount(() => {
		// use the existence of the dark class on the html element for the initial value
		dark = document.documentElement.classList.contains('dark');

		// show UI controls
		hidden = false;

		// listen for changes so we auto-adjust based on system settings
		const matcher = window.matchMedia('(prefers-color-scheme: dark)');
		matcher.addEventListener('change', handleChange);
		return () => matcher.removeEventListener('change', handleChange);
	});

	function handleChange({ matches: dark }: MediaQueryListEvent) {
		// only set if we haven't overridden the theme
		if (!localStorage.theme) {
			setMode(dark);
		}
	}

	function toggle() {
		setMode(!dark);
	}

	function setMode(value: boolean) {
		dark = value;

		// update page styling
		if (dark) {
			document.documentElement.classList.add('dark');
		} else {
			document.documentElement.classList.remove('dark');
		}

		// store the theme as a local override
		localStorage.theme = dark ? 'dark' : 'light';

		// if the toggled-to theme matches the system defined theme, clear the local override
		// this effectively provides a way to override or revert to "automatic" setting mode
		if (window.matchMedia(`(prefers-color-scheme: ${localStorage.theme})`).matches) {
			localStorage.removeItem('theme');
		}
	}
</script>

<!-- Apply dark mode class based on user preference and device settings (in head to avoid unstyled flashes) -->
<svelte:head>
	<script>
		if (
			localStorage.theme === 'dark' ||
			(!localStorage.theme && window.matchMedia('(prefers-color-scheme: dark)').matches)
		) {
			document.documentElement.classList.add('dark');
		} else {
			document.documentElement.classList.remove('dark');
		}
	</script>
</svelte:head>

<button
	class="{dark
		? 'bg-gray-600 focus:ring-fuchsia-600 ring-offset-gray-700'
		: 'bg-gray-200 focus:ring-fuchsia-600 ring-offset-white'} relative inline-flex flex-shrink-0 h-5 w-9 border-2 border-transparent rounded-full cursor-pointer transition-colors ease-in-out duration-200 focus:outline-none focus:ring-2 focus:ring-offset-2 mt-3 ml-4 mt-8 mb-4 md:mb-0 md:mt-2"
	class:hidden
	type="button"
	on:click={toggle}
>
	<span class="sr-only">Toggle Dark Mode</span>
	<span
		class="{dark
			? 'translate-x-0 bg-gray-300'
			: 'translate-x-4 bg-white'} pointer-events-none relative inline-block h-4 w-4 rounded-full shadow transform ring-0 transition ease-in-out duration-200"
	>
		<span
			class="{dark
				? 'opacity-100 ease-in duration-200'
				: 'opacity-0 ease-out duration-100'} absolute inset-0 h-full w-full flex items-center justify-center transition-opacity text-slate-700"
			aria-hidden="true"
		>
			<Moon />
		</span>

		<span
			class="{dark
				? 'opacity-0 ease-out duration-100'
				: 'opacity-100 ease-in duration-200'} absolute inset-0 h-full w-full flex items-center justify-center transition-opacity text-amber-500"
			aria-hidden="true"
		>
			<Sun />
		</span>
	</span>
</button>
