<script>
	import ThemeToggle from '$lib/components/ThemeToggle.svelte';
	import { base } from '$app/paths';

	import { page } from '$app/stores';
	import { Navbar, NavLi, NavUl, NavHamburger } from 'flowbite-svelte';
	$: activeUrl = $page.url.pathname;
</script>

<div
	class="pl-8 pr-8 md:pl-20 md:pr-20 pt-4 pb-4 w-full flex justify-between border-solid border-b items-center border-lightgrey dark:border-white dark:border-opacity-20"
>
	<Navbar
		let:hidden
		let:toggle
		fluid={true}
		class="dark:bg-black px-0 sm:px-0"
		navDivClass="flex flex-wrap justify-between items-center"
	>
		<div>
			<a href={base} class="block logo">
				<img src={`${base}/images/casey-full.svg`} alt="Hiphops Logo" class="block dark:hidden" />
				<img
					src={`${base}/images/casey-full-dark.svg`}
					alt="Hiphops Logo"
					class="hidden dark:block"
				/>
			</a>
		</div>

		<NavHamburger on:click={toggle} />
		<NavUl
			{activeUrl}
			{hidden}
			classUl="dark:bg-black dark:text-white p-0 md:items-center md:space-x-4 mx-2"
			activeClass="text-white dark:text-black bg-black dark:bg-white rounded-full md:px-4 md:py-2 font-semibold text-base"
			nonActiveClass="md:px-4 md:py-2 text-base rounded-full dark:hover:bg-grey dark:hover:bg-opacity-20 hover:bg-lightgrey hover:bg-opacity-40"
		>
			<NavLi href="/console">Tasks</NavLi>
			<NavLi href="https://docs.hiphops.io">Docs</NavLi>
			<div>
				<ThemeToggle />
			</div>
		</NavUl>
	</Navbar>
</div>
