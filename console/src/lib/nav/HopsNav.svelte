<script>
	import ThemeToggle from '$lib/components/ThemeToggle.svelte';
	import { base } from '$app/paths';

	import { page } from '$app/stores';
	$: activeUrl = $page.url.pathname;
	import { Navbar, Dropdown, DropdownItem } from 'flowbite-svelte';
	import { DotsHorizontalOutline } from 'flowbite-svelte-icons';
</script>

<div
	class="pl-8 pr-8 md:pl-20 md:pr-20 pt-4 pb-4 w-full flex justify-between border-solid border-b items-center border-lightgrey dark:border-white dark:border-opacity-10"
>
	<Navbar
		fluid={true}
		class="dark:bg-black dark:bg-opacity-0 px-0 sm:px-0"
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

		<DotsHorizontalOutline class="dots-menu dark:text-white" />
		<Dropdown
			triggeredBy=".dots-menu"
			containerClass="text-left border border-lightgrey dark:bg-black"
		>
			<DropdownItem defaultClass="hover:bg-almostblack px-4 py-2 text-sm">
				<a href="/console">Tasks</a>
			</DropdownItem>

			<DropdownItem defaultClass="hover:bg-almostblack px-4 py-2 text-sm">
				<a href="/console/advanced">Advanced</a>
			</DropdownItem>

			<DropdownItem defaultClass="hover:bg-almostblack px-4 py-2 text-sm">
				<a href="https://docs.hiphops.io" target="_blank">Docs</a>
			</DropdownItem>
			<!-- 
			<div>
				<ThemeToggle />
			</div> -->
		</Dropdown>
	</Navbar>
</div>
