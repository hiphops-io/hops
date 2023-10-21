<script lang="ts">
	import type { Writable } from 'svelte/store';
	import { Checkbox } from 'flowbite-svelte';

	import type { BoolParam } from './api';

	export let param: BoolParam;
	export let id = '';
	export let label = '';
	export let formValues: Writable<{ [key: string]: unknown }>;

	const fieldprops = {
		id,
		name: param.name,
		label,
		checked: param.default || false
	};

	let fieldValue = param.default;
	$: fieldValue, setValue();

	const setValue = () => {
		if (fieldValue) {
			$formValues[param.name] = true;
			return;
		}

		$formValues[param.name] = false;
	};
</script>

<div class="md:!mx-0 border-b border-solid pb-4 border-midgrey dark:border-grey">
	<Checkbox class="items-start pb-5" spacing="p-2" {...fieldprops} bind:checked={fieldValue}>
		<h5 class="text-black dark:text-white font-semibold ml-1">
			{param.display_name}
			<br />
			<div class="mb-1" />
			{#if !!param.help}
				<span class="text-grey dark:text-white dark:text-opacity-60 text-opacity-90 font-normal"
					>{param.help}</span
				>
			{/if}
		</h5>
	</Checkbox>
</div>
