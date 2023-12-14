<script lang="ts">
	import ScheduleDetailCard from './ScheduleDetailCard.svelte';
	import EventDetailCard from './EventDetailCard.svelte';

	import { activeAutomationStore } from '$lib/store';
	import { onMount } from 'svelte';

	interface Automation {
		name: string;
		type: string;
		hops: string;
		// Add other properties here
	}

	let activeAutomation: Automation = { name: '', type: '', hops: '' };

	onMount(() => {
		const unsubscribe = activeAutomationStore.subscribe((value: any) => {
			activeAutomation = value;
		});

		return unsubscribe;
	});

	let showCode = false;

	function toggleShowCode() {
		showCode = !showCode;
	}
</script>

<!--Code window & detail/output container-->
<div class="w-full h-auto space-y-2 px-4">
	<!--Automation actions-->
	<div class="w-full flex justify-between items-center py-2">
		<div>
			<p class="text-nines font-semibold text-sm">{activeAutomation.name}</p>
		</div>

		<div class="flex space-x-6 items-center">
			<button on:click={toggleShowCode} class="text-nines font-semibold flex text-sm space-x-2">
				<img src="images/desktop_app/code-icon.svg" alt="Code icon" />
				<p>{showCode ? 'Hide code' : 'Show code'}</p>
			</button>
			<button
				class="bg-purple font-semibold flex px-2 py-1 text-sm rounded-sm space-x-2 items-center"
			>
				<img src="images/desktop_app/publish-icon.svg" alt="Publish icon" />
				<p>Publish</p>
			</button>
		</div>
	</div>

	<div class="flex !m-0">
		<!--Code window-->
		<div class="bg-almostblack h-[660px] rounded-lg flex {showCode ? 'w-[50%] mr-2' : 'hidden'}">
			<p class="text-error m-auto">{activeAutomation.hops}</p>
		</div>

		<!--Detail/output window-->
		<div
			class="bg-almostblack h-[660px] rounded-lg flex justify-center {showCode
				? ' w-[50%]'
				: 'w-full'}"
		>
			<div class="py-8 px-6 {showCode ? 'w-full' : 'w-[50%]'}">
				<div class="mb-4">
					<h1 class="text-white text-4xl font-normal">{activeAutomation.name}</h1>
					<h2 class="text-nines">Description</h2>
				</div>

				{#if activeAutomation.type === 'task'}
					<p>Task</p>
				{:else if activeAutomation.type === 'schedule'}
					<ScheduleDetailCard />
				{:else}
					<EventDetailCard />
				{/if}
			</div>
		</div>

		<!--END code window & detail/output container-->
	</div>

	<!--Instruction container-->
	<div class="bg-almostblack w-full h-[160px] rounded-lg" />
</div>
