<script lang="ts">
	import AutomationDetailView from '$lib/desktop_app/AutomationDetailView.svelte';
	import ConnectApps from '$lib/desktop_app/ConnectApps.svelte';
	import EventsDetailView from '$lib/desktop_app/EventsDetailView.svelte';
	import SettingsDetailView from '$lib/desktop_app/SettingsDetailView.svelte';
	import NewAutomation from '$lib/desktop_app/NewAutomation.svelte';
	import SidebarNav from '$lib/desktop_app/SidebarNav.svelte';
	import { activeItemStore } from '$lib/store';

	let dummyAutomationsData = [
		{
			name: 'Automation 1'
		},
		{
			name: 'Automation 2'
		}
	];

	let dummyTemplateData = [
		{
			name: 'Add a new user',
			emoji: '🤗'
		},
		{
			name: 'Request a pretzel',
			emoji: '🥨'
		},
		{
			name: 'Request a pretzel',
			emoji: '🥨'
		},
		{
			name: 'Request a pretzel',
			emoji: '🥨'
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
	<SidebarNav {dummyAutomationsData} />

	{#if activeItem === 'New automation'}
		<NewAutomation {dummyTemplateData} />
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
