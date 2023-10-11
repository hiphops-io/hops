<script lang="ts">
	import type { PageData } from './$types';

	import HopsNav from '$lib/nav/HopsNav.svelte';
	import TaskItemCard from '$lib/tasks/TaskItemCard.svelte';
	import type { TaskSummary } from '$lib/tasks/api';

	export let data: PageData;
	export let tasks = data.tasks as TaskSummary[];
</script>

<svelte:head>
	<title>Tasks</title>
	<meta name="description" content="Trigger task flows" />
</svelte:head>

<div class="dark:bg-grain bg-cover bg-norepeat h-screen">
	<HopsNav />

	<!--Page title-->
	<div class="mx-8 md:mx-20 mt-16 pb-4">
		<h1 class="text-left text-6xl font-normal mb-4 dark:text-white">Tasks</h1>
		<h2 class="text-2xl mb-4 text-grey dark:text-white dark:text-opacity-50">
			Make things happen in one click
		</h2>
	</div>

	<!--Tasks list-->
	<div class="mx-8 md:mx-20 pb-20 md:space-x-4">
		<div class="grid md:grid-cols-4 gap-4">
			<!--Task list items-->
			{#each tasks as command}
				<TaskItemCard {command} />
			{/each}
		</div>

		<!--Tasks empty state -->
		{#if tasks.length === 0}
			<div
				class="bg-[url('/images/grain.jpg')] bg-cover flex items-center justify-center text-center rounded-lg mt-4 py-60 md:!ml-0"
			>
				<p class="text-white text-base font-medium">No tasks yet</p>
			</div>
		{/if}
	</div>
</div>
