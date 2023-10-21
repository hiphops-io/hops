<script lang="ts">
	import type { Writable } from 'svelte/store';
	import { Select } from 'flowbite-svelte';

	import type { StrParam } from './api';

	export let param: StrParam;
	export let id = '';
	export let label = '';
	export let placeholder = param.default || 'Type here...';
	export let formValues: Writable<{ [key: string]: unknown }>;

	const fieldprops = {
		id,
		name: param.name,
		label,
		placeholder,
		require: param.required
	};

	let fieldValue = param.default;
	$: fieldValue, ($formValues[param.name] = fieldValue);

	// Mock-up data for developing
	const countries = [
		{ value: 'us', name: 'United States' },
		{ value: 'ca', name: 'Canada' },
		{ value: 'fr', name: 'France' }
	];
</script>

<!--Dropdown-->
<div class="md:!mx-0">
	<h5 class="text-black dark:text-white font-semibold text-sm">
		Dropdown <br />
		<span class="text-grey dark:text-white dark:text-opacity-60 text-opacity-60 font-medium"
			>Optional description</span
		>
	</h5>
	<Select
		class="mt-4 py-4 px-6 dark:text-white dark:placeholder:text-lightgrey bg-white dark:bg-white dark:bg-opacity-10 resize-none focus:ring-0 focus:border-purple dark:focus:border-purple focus:border font-medium text-base rounded-none border-midgrey w-full"
		items={countries}
		bind:value={fieldValue}
		{...fieldprops}
	/>
</div>
