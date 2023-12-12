<script lang="ts">
	import { activeItemStore } from '$lib/store';
	import NavItem from './NavItem.svelte';
	// Icons
	import Plus from '$lib/icons/Plus.svelte';
	import Apps from '$lib/icons/Apps.svelte';
	import Events from '$lib/icons/Events.svelte';
	import Settings from '$lib/icons/Settings.svelte';

	let activeItem = 'New automation';

	function setActiveItem(navItem: string) {
		activeItem = navItem;
		activeItemStore.set(activeItem);
	}

	// Automations dummy data
	export let dummyData: any;

	$: console.log(activeItem);
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
			{#each dummyData as automations}
				<li>
					<button on:click={() => setActiveItem(automations.name)}>
						<NavItem name={automations.name} active={activeItem === automations.name} />
					</button>
				</li>
			{/each}
		</ul>
	</div>
</div>
