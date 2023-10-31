<script lang="ts">
	import TableHeadItem from '$lib/tables/TableHeadItem.svelte';
	import TableDataCell from '$lib/tables/TableDataCell.svelte';
	import Tag from '$lib/components/Tag.svelte';
	import HopsNav from '$lib/nav/HopsNav.svelte';
	import { PlayOutline } from 'flowbite-svelte-icons';

	let tableData = [
		{
			timestamp: '2023-10-10 10:00 AM',
			eventName: 'Event 1',
			pipelines: 3,
			pipelineNames: ['Pipeline 1', 'Pipeline 2', 'Pipeline 3'],
			hops: {
				event: 'pull_request',
				source: 'github',
				action: 'opened'
			},
			JSON: {
				status: 'Success',
				duration: '2 hours 30 minutes',
				logs: ['Data Ingestion started at 9:30 AM.', 'Data Processing completed at 12:00 PM.']
			}
		},
		{
			timestamp: '2023-10-10 10:02 AM',
			eventName: 'Event 2',
			pipelines: 1,
			pipelineNames: ['Pipeline 1'],
			hops: {
				event: 'pull_request',
				source: 'github',
				action: 'opened'
			},
			JSON: {
				status: 'Success',
				duration: '2 hours 30 minutes',
				logs: ['Data Ingestion started at 9:30 AM.', 'Data Processing completed at 12:00 PM.']
			}
		}
	];

	tableData = tableData.sort((a, b) => b.timestamp.localeCompare(a.timestamp));

	let activeRow = tableData[0];

	function setActiveRow(row: any) {
		activeRow = row;
	}
</script>

<svelte:head>
	<title>Advanced</title>
	<meta name="description" content="Trigger task flows" />
</svelte:head>

<div class="dark:bg-grain bg-cover bg-norepeat h-screen">
	<HopsNav />

	<!--Page title-->
	<div class="md:pl-20 md:pr-20 mx-8 md:mx-auto mt-16 pb-4">
		<h1 class="text-left text-5xl font-normal mb-2 dark:text-white">Events</h1>

		<h2 class="text-xl mb-6 text-grey dark:text-white dark:text-opacity-50">
			See what's going on with your tasks
		</h2>
	</div>

	<!-- Events container-->
	<div class="flex space-x-4 pl-8 pr-8 md:pl-20 md:pr-20 text-white">
		<!--Events table container-->
		<div
			class="w-1/2 dark:bg-[#191919] dark:border-none border border-lightrey overflow-scroll h-[450px] rounded-lg"
		>
			<div class="w-full">
				<table class="table-auto w-full">
					<thead class="text-left">
						<tr>
							<TableHeadItem>Timestamp</TableHeadItem>
							<TableHeadItem>Event name</TableHeadItem>
							<TableHeadItem>Pipelines</TableHeadItem>
						</tr>
					</thead>
					<tbody>
						{#each tableData as row (row.timestamp)}
							<tr
								class="hover:bg-lightgrey dark:hover:bg-almostblack {row === activeRow
									? 'border-l-2 border-purple bg-lightgrey dark:bg-almostblack'
									: ''}"
								on:click={() => setActiveRow(row)}
							>
								<TableDataCell>{row.timestamp}</TableDataCell>
								<TableDataCell>{row.eventName}</TableDataCell>
								<TableDataCell>{row.pipelines}</TableDataCell>
							</tr>
						{/each}
					</tbody>
				</table>
			</div>
		</div>

		<!--Selected event detail container-->
		<div
			class="w-1/2 bg-white border border-lightrey dark:border-none dark:bg-[#191919] overflow-scroll h-[450px] text-black dark:text-white rounded-lg"
		>
			<div class="p-8">
				<h2 class="text-2xl mb-8 text-grey dark:text-midgrey">{activeRow.eventName}</h2>
				<!--Pipelines container-->
				<div class="mt-4 mb-12">
					<Tag>Pipelines</Tag>

					<div
						class="mt-4 mb-12 border-l-2 border-midgrey dark:border-almostblack py-1 px-4 text-sm text-grey dark:text-midgrey"
					>
						{#each activeRow.pipelineNames as pipeline}
							<div class="mb-3 last:mb-0">
								<PlayOutline class="inline-block w-3 h-3 mr-2" strokeWidth="1" />
								{pipeline}
							</div>
						{/each}
					</div>
				</div>

				<Tag>Hops</Tag>
				<div
					class="mt-4 mb-12 border-l-2 border-midgrey dark:border-almostblack py-2 px-4 text-sm text-grey dark:text-midgrey"
				>
					<pre>{JSON.stringify(activeRow.hops, null, 2)}</pre>
				</div>

				<div class="mt-4 mb-12">
					<Tag>JSON</Tag>
					<div
						class="mt-4 mb-12 border-l-2 border-midgrey dark:border-almostblack py-2 px-4 text-sm text-grey dark:text-midgrey"
					>
						<pre>{JSON.stringify(activeRow.JSON, null, 2)}</pre>
					</div>
				</div>
			</div>
		</div>
	</div>
</div>
