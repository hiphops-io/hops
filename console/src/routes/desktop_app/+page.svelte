<script lang="ts">
	import AutomationDetailView from '$lib/desktop_app/AutomationDetailView.svelte';
	import SidebarNav from '$lib/desktop_app/SidebarNav.svelte';
	import { activeItemStore } from '$lib/store';

	let dummyData = [
		{
			name: 'Automation 1'
		},
		{
			name: 'Automation 2'
		}
	];

	$: activeItem = $activeItemStore;
</script>

<svelte:head>
	<title>Hiphops</title>
	<meta name="description" content="Desktop app" />
</svelte:head>

<!--Page container-->
<div class="flex">
	<!--Sidebar-->
	<SidebarNav {dummyData} />

	{#if activeItem === 'New automation'}
		<div>New automation</div>
	{:else if activeItem === 'Connect apps'}
		<div>Connect apps</div>
	{:else if activeItem === 'Events'}
		<div>Events</div>
	{:else if activeItem === 'Settings'}
		<div>Setting</div>
	{:else}
		<AutomationDetailView />
	{/if}
</div>
