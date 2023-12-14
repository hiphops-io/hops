<script lang="ts">
	import AutomationDetailView from '$lib/desktop_app/AutomationDetailView.svelte';
	import ConnectApps from '$lib/desktop_app/ConnectApps.svelte';
	import EventsDetailView from '$lib/desktop_app/EventsDetailView.svelte';
	import SettingsDetailView from '$lib/desktop_app/SettingsDetailView.svelte';
	import NewAutomation from '$lib/desktop_app/NewAutomation.svelte';
	import SidebarNav from '$lib/desktop_app/SidebarNav.svelte';
	import { activeItemStore, activeAutomationStore } from '$lib/store';

	let dummyAutomationsData = [
		{
			name: 'Automation 1 (Task)',
			type: 'task',
			hops: 'Hops blob 1',
			files: [
				{
					name: 'file1',
					content: console.log('Hello world')
				},
				{
					name: 'file2',
					content: console.log('Hello world')
				}
			]
		},
		{
			name: 'Automation 2 (Schedule)',
			type: 'schedule',
			hops: 'Hops blob 2',
			files: [
				{
					name: 'file3',
					content: console.log('This is a schedule')
				}
			]
		},
		{
			name: 'Automation 3 (Event)',
			type: 'event',
			hops: 'Hops blob 3',
			files: [
				{
					name: 'file4',
					content: console.log('This is an event')
				}
			]
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
			name: 'Launch the rocket',
			emoji: '🚀'
		},
		{
			name: 'Share the love',
			emoji: '💓'
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
