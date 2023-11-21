<script lang="ts">
	import type { Writable } from 'svelte/store';
	import { validators, required, type Validator, Hint } from 'svelte-use-form';
	import { onMount } from 'svelte';
	import type { TextParam } from './api';

	export let param: TextParam;
	export let id = '';
	export let label = '';
	export let rows = 8;
	export let placeholder = param.default || 'Type here...';
	export let formValues: Writable<{ [key: string]: unknown }>;

	const fieldprops = {
		id,
		name: param.name,
		label,
		rows,
		placeholder,
		required: param.required
	};

	let fieldValidators: Validator[] = [];
	if (param.required) {
		fieldValidators.push(required);
	}

	onMount(() => {
		const urlParams = new URLSearchParams(window.location.search);
		for (const [key, value] of urlParams) {
			if (key === param.name) {
				fieldValue = value;
			}
		}
	});

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

<!--Textarea-->
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

	<textarea
		use:validators={fieldValidators}
		bind:value={fieldValue}
		class="mt-4 py-6 px-6 dark:text-white dark:placeholder:text-darkgrey dark:bg-white bg-white dark:bg-opacity-10 resize-none border-midgrey dark:border-grey font-medium text-base rounded-none w-full focus:outline-none focus:ring-0 focus:border-purple dark:focus:border-purple"
		{...fieldprops}
	/>
	<Hint class="text-error font-medium text-sm mt-2" for={param.name} on="required"
		>This field is required</Hint
	>
</div>
