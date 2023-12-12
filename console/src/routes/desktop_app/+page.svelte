<script lang="ts">
	import AutomationDetailView from '$lib/desktop_app/AutomationDetailView.svelte';
	import ConnectApps from '$lib/desktop_app/ConnectApps.svelte';
	import EventsDetailView from '$lib/desktop_app/EventsDetailView.svelte';
	import SettingsDetailView from '$lib/desktop_app/SettingsDetailView.svelte';
	import NewAutomation from '$lib/desktop_app/NewAutomation.svelte';
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
		<NewAutomation />
	{:else if activeItem === 'Connect apps'}
		<ConnectApps />
	{:else if activeItem === 'Events'}
		<EventsDetailView />
	{:else if activeItem === 'Settings'}
		<SettingsDetailView />
	{:else}
		<AutomationDetailView />
	{/if}
</div>
