<script lang="ts">
	import TableHeadItem from '$lib/tables/TableHeadItem.svelte';
	import TableDataCell from '$lib/tables/TableDataCell.svelte';
	import HopsNav from '$lib/nav/HopsNav.svelte';
	import type { PageData } from './$types';
	import Loading from '$lib/components/Loading.svelte';

	//Set default tab
	let tab = 'Events';

	export let data: PageData;

	// Event logs table data
	$: tableData = data.tableData;
	$: ago = data.ago;
	$: loading = data.tableData === undefined;

	// Dummy Pipeline data
	let pipelineData = [
		{
			name: 'Pipeline Name #1',
			metadata: 'Metadata',
			pastStatus: ['success', 'failure', 'success', 'success', 'failure']
		},
		{
			name: 'Pipeline Name #2',
			metadata: 'Metadata',
			pastStatus: ['failure', 'failure', 'failure', 'success', 'failure']
		},
		{
			name: 'Pipeline Name #3',
			metadata: 'Metadata',
			pastStatus: ['success', 'failure', 'failure', 'success', 'failure']
		}
	];

	let activeRow: any;
	$: activeRow = tableData?.[0];

	function setActiveRow(row: any) {
		activeRow = row;
	}
</script>

<svelte:head>
	<title>Advanced</title>
	<meta name="description" content="Trigger task flows" />
</svelte:head>

{#if loading}
	<Loading />
{:else}
	<div class="dark:bg-grain bg-cover bg-norepeat bg-fixed min-h-screen">
		<HopsNav />

		<!--Page title-->
		<div class="md:pl-20 md:pr-20 mx-8 md:mx-auto mt-12 pb-4">
			<h1 class="text-left text-5xl font-normal mb-2 dark:text-white">Advanced</h1>
			<h2 class="text-2xl mb-4 text-grey dark:text-white dark:text-opacity-50">
				Showing all events since {ago}
			</h2>

			<!--Tabs-->
			<!-- <div class="mt-10">
				<ul class="flex space-x-2"> -->
			<!-- <li>
						<button
							on:click={() => (tab = 'Pipelines')}
							class={tab === 'Pipelines'
								? 'bg-white text-black px-6 py-2 border border-black rounded-full font-semibold'
								: 'text-white border border-white rounded-full px-6 py-2 font-medium'}
							>Pipelines
						</button>
					</li> -->
			<!-- <li>
						<button
							on:click={() => (tab = 'Events')}
							class={tab === 'Events'
								? 'bg-white text-black px-6 py-2 border border-black rounded-full font-semibold'
								: 'text-white border border-white rounded-full px-6 py-2 font-medium'}
							>Event logs
						</button>
					</li> -->
			<!-- </ul>
			</div> -->
		</div>

		<!-- Container-->

		{#if tab === 'Pipelines'}
			<div class="pl-8 pr-8 md:pl-20 md:pr-20 pb-16 text-white">
				<!--Pipelines list container-->
				{#each pipelineData as pipeline}
					<a href="/console/advanced/pipeline_name">
						<div
							class="dark:bg-[#191919] rounded-lg px-8 py-8 mb-2 border border-[#191919] hover:border hover:border-purple hover:duration-300"
						>
							<!--Pipeline title & run indicators-->
							<div class="flex justify-between">
								<h2 class="text-2xl text-lightgrey">{pipeline.name}</h2>

								<div class="flex space-x-2">
									{#each pipeline.pastStatus as status}
										{#if status === 'success'}
											<img src="/images/success.svg" alt="success icon" />
										{:else if status === 'failure'}
											<img src="/images/failure.svg" alt="failure icon" />
										{/if}
									{/each}
								</div>
							</div>

							<!-- Pipeline metadata-->
							<p class="text-midgrey mt-1 font-normal text-base">{pipeline.metadata}</p>
						</div>
					</a>
				{/each}
			</div>
		{/if}

		<!--Tasks empty state -->
		{#if tab === 'Events' && tableData.length == 0}
			<div class="mx-8 md:mx-20 pb-20 md:space-x-4">
				<div
					class="bg-almostblack flex items-center justify-center text-center rounded-lg mt-4 py-60 md:!ml-0"
				>
					<p class="text-white text-base font-medium">No events since {ago}</p>
				</div>
			</div>
		{/if}

		{#if tab === 'Events' && tableData.length !== 0}
			<div class="flex space-x-4 pl-8 pr-8 md:pl-20 md:pr-20 text-white">
				<!--Events table container-->
				<div
					class="w-1/2 dark:bg-[#191919] dark:border-none border border-lightrey overflow-scroll h-[60vh] rounded-lg"
				>
					<div class="w-full">
						<table class="table-auto w-full">
							<thead class="text-left">
								<tr>
									<TableHeadItem>Timestamp</TableHeadItem>
									<TableHeadItem>ID</TableHeadItem>
									<TableHeadItem>Source</TableHeadItem>
									<TableHeadItem>Event</TableHeadItem>
									<TableHeadItem>Action</TableHeadItem>
								</tr>
							</thead>
							<tbody>
								{#each tableData as row, i (`${i}-${row.timestamp}`)}
									<tr
										class="hover:bg-lightgrey dark:hover:bg-almostblack {row === activeRow
											? 'border-l-2 border-purple bg-lightgrey dark:bg-almostblack'
											: ''}"
										on:click={() => setActiveRow(row)}
									>
										<TableDataCell>{row.timestamp}</TableDataCell>
										<TableDataCell>{row.eventId.slice(0, 7) + '...'}</TableDataCell>
										<TableDataCell>{row.source}</TableDataCell>
										<TableDataCell>{row.event}</TableDataCell>
										<TableDataCell>{row.action}</TableDataCell>
									</tr>
								{/each}
							</tbody>
						</table>
					</div>
				</div>

				<!--Selected event detail container-->
				{#if activeRow}
					<div
						class="w-1/2 bg-white border border-lightrey dark:border-purple dark:bg-[#191919] overflow-scroll h-[60vh] text-black dark:text-white rounded-lg"
					>
						<div class="p-8">
							<h2 class="text-xl mb-8 text-grey dark:text-midgrey">ID: {activeRow.eventId}</h2>
							<!--Pipelines container-->

							<div class="mt-4 mb-12">
								<div
									class="mt-4 mb-12 border-midgrey dark:border-almostblack py-1 text-sm text-grey dark:text-midgrey"
								>
									<pre>{JSON.stringify(activeRow.JSON, null, 2)}</pre>
								</div>
							</div>
						</div>
					</div>
				{/if}
			</div>
		{/if}
	</div>
{/if}
