<script lang="ts">
	export let icon: any;
	export let name: string;
	export let active: boolean = false;

	let strokeColor: string = '#999999';
	$: if (active) {
		strokeColor = '#FFF';
	} else {
		strokeColor = '#999999';
	}
</script>

<div
	class="flex space-x-2 items-center px-4 pb-2 {active
		? 'border-l-4 border-purple'
		: 'border-l-4 border-transparent'}"
>
	<svelte:component this={icon} {strokeColor} />
	<p class={active ? 'text-white font-semibold' : ''}>{name}</p>
</div>
