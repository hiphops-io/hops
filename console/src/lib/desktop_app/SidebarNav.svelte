<script lang="ts">
	import { activeAutomationStore, activeItemStore } from '$lib/store';
	import NavItem from './NavItem.svelte';
	// Icons
	import Plus from '$lib/icons/Plus.svelte';
	import Apps from '$lib/icons/Apps.svelte';
	import Events from '$lib/icons/Events.svelte';
	import Settings from '$lib/icons/Settings.svelte';

	let activeItem = 'New automation';

	$: activeAutomation = $activeAutomationStore;

	function setActiveItem(navItem: string, automation?: any) {
		console.log(navItem, automation);
		activeItem = navItem || automation.name;
		activeItemStore.set(activeItem);
		activeAutomationStore.set(automation);
	}

	// Automations dummy data
	export let dummyAutomationsData: any;
</script>

<div class="w-[20%] h-screen text-nines">
	<ul class="space-y-2 pt-8">
		<li>
			<button on:click={() => setActiveItem('New automation')}>
				<NavItem name="New automation" icon={Plus} active={activeItem === 'New automation'} />
			</button>
		</li>
		<li>
			<button on:click={() => setActiveItem('Connect apps')}>
				<NavItem name="Connect apps" icon={Apps} active={activeItem === 'Connect apps'} />
			</button>
		</li>
		<li>
			<button on:click={() => setActiveItem('Events')}>
				<NavItem name="Events" icon={Events} active={activeItem === 'Events'} />
			</button>
		</li>
		<li>
			<button on:click={() => setActiveItem('Settings')}>
				<NavItem name="Settings" icon={Settings} active={activeItem === 'Settings'} />
			</button>
		</li>
	</ul>

	<div>
		<p class="font-semibold text-sm mt-8 px-4 pb-2">Automations</p>
		<ul>
			{#each dummyAutomationsData as automation}
				<li>
					<button on:click={() => setActiveItem(automation.name, automation)}>
						<NavItem name={automation.name} active={activeItem === automation.name} />
					</button>
					{#if automation === activeAutomation}
						<ul class="border-l border-grey ml-5 pl-2 space-y-2 mb-4">
							{#each automation.files as file}
								<li class="text-midgrey font-medium">{file}</li>
							{/each}
						</ul>
					{/if}
				</li>
			{/each}
		</ul>
	</div>
</div>
