<script lang="ts">
	import { tableData } from './dummyData';
	import PipelineNav from '$lib/components/PipelineNav.svelte';
	import TableHeadItem from '$lib/tables/TableHeadItem.svelte';
	import TableDataCell from '$lib/tables/TableDataCell.svelte';
	import { slide } from 'svelte/transition';

	let activeRow = tableData[0];

	let activeDetail = '';

	function setActiveRow(row: any) {
		activeRow = row;
	}

	//Toggle Step detail accordions
	function toggleDetail(id: string) {
		if (activeDetail === id) {
			activeDetail = '';
		} else {
			activeDetail = id;
			console.log(id);
		}
	}
</script>

<svelte:head>
	<title>Pipeline name | Pipelines</title>
	<meta name="description" content="" />
</svelte:head>

<div class="dark:bg-grain bg-cover bg-norepeat min-h-screen pb-16">
	<PipelineNav />

	<!--Page title-->
	<div class="mx-8 md:mx-20 mt-16 pb-4">
		<h1 class="text-left text-5xl font-normal mb-4 dark:text-white">Pipeline name</h1>
		<h2 class="text-base mb-4 text-grey dark:text-white dark:text-opacity-50">Pipeline metadata</h2>
	</div>

	<div class="flex space-x-4 pl-8 pr-8 md:pl-20 md:pr-20 text-white h-[60vh]">
		<!--Events table container-->
		<div
			class="w-1/2 dark:bg-[#191919] dark:border-none border border-lightrey overflow-scroll rounded-lg"
		>
			<div class="w-full">
				<table class="table-auto w-full">
					<thead class="text-left">
						<tr>
							<TableHeadItem>Timestamp</TableHeadItem>
							<TableHeadItem>ID</TableHeadItem>
							<TableHeadItem>Data</TableHeadItem>
							<TableHeadItem>Status</TableHeadItem>
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
								<TableDataCell>{row.eventId}</TableDataCell>
								<TableDataCell>{row.data}</TableDataCell>
								<TableDataCell>{row.status}</TableDataCell>
							</tr>
						{/each}
					</tbody>
				</table>
			</div>
		</div>

		<!--Selected pipeline detail container-->
		<div
			class="w-1/2 bg-white border border-lightrey dark:border-purple dark:bg-[#191919] overflow-scroll text-black dark:text-white rounded-lg h-[60vh]"
		>
			<div class="px-8 pt-4">
				{#each activeRow.pipelineSteps as step}
					<div>
						<div class="flex pt-4">
							<!--Step status indicators-->
							<div class="pr-4">
								{#if step.status === 'Completed'}
									<img src="/images/success.svg" alt="success icon" />
								{:else if step.status === 'Failed'}
									<img src="/images/failure.svg" alt="failure icon" />
								{:else if step.status === 'In progress'}
									<img
										src="/images/inprogress.svg"
										alt="loading icon"
										class="animate-[spin_2s_linear_infinite] hidden dark:block"
									/>
								{:else if step.status === 'DNR'}
									<img src="/images/dnr.svg" alt="Did not run icon" />
								{/if}
							</div>
							<!--Step Title & metadata-->
							<div class="border-b border-almostblack w-full pb-4 flex justify-between">
								<div class="w-full">
									<h4 class="font-medium">{step.name}</h4>
									<p class="text-sm text-midgrey">{step.status} {step.executionTime}</p>

									<!--Detail container-->
									{#if activeDetail === step.name}
										<div
											class="bg-almostblack p-4 mt-2 rounded"
											transition:slide={{ duration: 400 }}
										>
											<pre>{JSON.stringify(step.JSON, null, 2)}</pre>
										</div>
									{/if}
								</div>
								<div>
									<button on:click={() => toggleDetail(step.name)}>
										<img
											src="/images/down-arrow.svg"
											alt="Down arrow"
											class={activeDetail === step.name ? 'rotate-180' : ''}
										/>
									</button>
								</div>
							</div>
						</div>
					</div>
				{/each}
			</div>
		</div>
	</div>
</div>
