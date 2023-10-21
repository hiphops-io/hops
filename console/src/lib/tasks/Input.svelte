<script lang="ts">
	import type { Writable } from 'svelte/store';
	import { validators, required, type Validator, Hint } from 'svelte-use-form';

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
		required: param.required
	};

	let fieldValidators: Validator[] = [];
	if (param.required) {
		fieldValidators.push(required);
	}

	let fieldValue = param.default;
	$: fieldValue, setValue();

	const setValue = () => {
		if (!!fieldValue) {
			$formValues[param.name] = fieldValue;
			return;
		}

		delete $formValues[param.name];
	};
</script>

<!-- Input -->
<div class="md:!mx-0">
	<h5 class="text-black dark:text-white font-semibold text-sm mb-1">
		{param.display_name || param.name}
		{#if !param.required}
			<span
				class="text-sm text-grey dark:text-white dark:text-opacity-60 text-opacity-60 font-semibold"
			>
				(optional)</span
			>
		{/if}
	</h5>
	{#if !!param.help}
		<p class="text-grey dark:text-white dark:text-opacity-60 text-opacity-90 font-normal text-sm">
			{param.help}
		</p>
	{/if}

	<input
		{...fieldprops}
		use:validators={fieldValidators}
		bind:value={fieldValue}
		class="mt-4 py-4 px-6 dark:text-white dark:placeholder:text-darkgrey bg-white dark:bg-white dark:bg-opacity-10 resize-none font-medium text-base rounded-none w-full border border-midgrey dark:border-grey focus:outline-none focus:ring-1 focus:ring-purple"
	/>
	<Hint class="text-error font-medium text-sm mt-2" for={param.name} on="required"
		>This field is required</Hint
	>
</div>
