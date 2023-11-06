<script lang="ts">
	import { writable } from 'svelte/store';
	import type { PageData } from './$types';
	import {
		ArrowRightSolid,
		CheckCircleOutline,
		ExclamationCircleOutline
	} from 'flowbite-svelte-icons';
	import { useForm } from 'svelte-use-form';

	import type { Task, TaskRunResponse } from '$lib/tasks/api';
	import TaskNav from '$lib/tasks/TaskNav.svelte';
	import Textarea from '$lib/tasks/Textarea.svelte';
	import NumberInput from '$lib/tasks/NumberInput.svelte';
	import Checkbox from '$lib/tasks/Checkbox.svelte';
	import Input from '$lib/tasks/Input.svelte';
	import { PUBLIC_BACKEND_URL } from '$env/static/public';
	import ky, { HTTPError } from 'ky';

	export let data: PageData;
	export let task = data.task as Task;
	task.params = task.params || [];

	const form = useForm();
	const formValues = writable<{ [key: string]: unknown }>({});
	let formStatus: 'ready' | 'invalid' | 'submitting' | 'error' | 'success' = 'ready';
	let errorResponse: TaskRunResponse;

	$: $form.valid || $form.touched,
		(formStatus = $form.touched && !$form.valid ? 'invalid' : 'ready');

	const onSubmit = async () => {
		$form.touched = true;

		if (!$form.valid) {
			return;
		}

		formStatus = 'submitting';

		try {
			$formValues['timestamp'] = Date.now();

			const respJson = (await ky
				.post(`${PUBLIC_BACKEND_URL}/tasks/${task.name}`, { json: $formValues })
				.json()) as TaskRunResponse;

			formStatus = 'success';
			updateUrlParam('sequence_id', respJson.sequence_id);
		} catch (error) {
			formStatus = 'error';

			if (error instanceof HTTPError) {
				errorResponse = (await error.response.json()) as TaskRunResponse;
			}
		}
	};

	// Update the URL's query params without navigating
	const updateUrlParam = (key: string, value: string) => {
		const url = new URL(window.location.toString());
		url.searchParams.set(encodeURIComponent(key), encodeURIComponent(value));
		history.replaceState({}, '', url);
	};
</script>

<svelte:head>
	<title>{task.display_name} | Task</title>
	<meta name="description" content="Run the '{task.display_name}' task" />
</svelte:head>

<TaskNav />

<!--Page title-->
<div class="md:w-2/4 mx-8 md:mx-auto mt-16 pb-4">
	<h1 class="text-left text-6xl font-normal mb-4 dark:text-white">{task.display_name}</h1>
	{#if !!task.description}
		<h2 class="text-2xl mb-6 text-grey dark:text-white dark:text-opacity-50">
			{task.description}
		</h2>
	{/if}
</div>

{#if formStatus === 'success'}
	<!-- Success message -->
	<div class="md:w-2/4 mx-8 md:m-auto mt-4 pb-4">
		<div
			class="bg-[url('/images/grain.jpg')] bg-cover items-center justify-center text-center rounded-lg mt-4 py-40 md:!ml-0"
		>
			<CheckCircleOutline color="white" class="h-16 w-16 m-auto mb-4" strokeWidth="1" />
			<p class="text-white text-base font-medium">Task created</p>
		</div>
	</div>
{:else if formStatus === 'error'}
	<!-- Error message -->
	<div class="md:w-2/4 mx-8 md:m-auto mt-4 pb-4">
		<div
			class="bg-[url('/images/grain.jpg')] bg-cover items-center justify-center text-center rounded-lg mt-4 py-40 md:!ml-0"
		>
			<ExclamationCircleOutline color="white" class="h-16 w-16 m-auto mb-4" strokeWidth="1" />
			<p class="text-white text-base font-medium mb-8">There was an error creating this task</p>
			{#if errorResponse}
				<p class="text-white text-base font-small">{errorResponse.message}</p>
				{#if errorResponse.errors}
					<ul class="text-white text-base font-small">
						{#each Object.entries(errorResponse.errors) as [field, errors]}
							<li>{field}: {errors}</li>
						{/each}
					</ul>
				{/if}
			{/if}
		</div>
	</div>
{:else}
	<!--Task form-->
	<form use:form>
		<div
			class={task.params.length !== 0
				? 'md:w-2/4 mx-8 md:mx-auto mb-20 px-8 py-2 pb-32 md:space-x-4 bg-grey bg-opacity-5 dark:bg-white dark:bg-opacity-10 rounded-xl space-y-12 pt-12'
				: 'md:w-2/4 md:mx-auto md:pb-24'}
		>
			{#each task.params as param}
				{#if param.type === 'string'}
					<Input {param} {formValues} />
				{:else if param.type === 'text'}
					<Textarea {param} {formValues} />
				{:else if param.type === 'bool'}
					<!--Checkbox-->
					<Checkbox {param} {formValues} />
				{:else if param.type === 'number'}
					<!--Input-->
					<NumberInput {param} {formValues} />
				{:else}
					<!-- TODO: Should handle this with an error state for the whole page. -->
					<div>
						Unknown field: {JSON.stringify(param)}
					</div>
				{/if}
			{/each}

			<!--Run task button for a task with fields-->
			<button
				class={`
			px-6 py-4 flex items-center
			font-semibold text-white dark:text-black
			bg-black dark:bg-white disabled:dark:bg-grey
			active:shadow-md active:shadow-purple active:disabled:shadow-black
			hover:scale-105
			transition-[transform] transition-[box-shadow] duration-200 ` +
					(task.params.length !== 0 ? 'float-right mt-12' : 'w-80 justify-between')}
				type="submit"
				disabled={formStatus !== 'ready'}
				on:click={onSubmit}
			>
				{#if formStatus === 'submitting'}
					Working on it
					<span class="ml-14">
						<CheckCircleOutline class="text-white dark:text-black" />
					</span>
				{:else}
					Run task
					<span class="ml-14">
						<ArrowRightSolid class="text-white dark:text-black" />
					</span>
				{/if}
			</button>
		</div>
	</form>
{/if}
