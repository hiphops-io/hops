<script lang="ts">
	import HopsNav from '$lib/nav/HopsNav.svelte';

	let tableData = [
		{
			timestamp: '2023-10-10 10:00 AM',
			eventName: 'Event 1',
			pipelines: 3,
			pipelineNames: ['Pipeline 1', 'Pipeline 2', 'Pipeline 3'],
			JSON: 'This is some JSON from Event 1'
		},
		{
			timestamp: '2023-10-10 02:30 PM',
			eventName: 'Event 2',
			pipelines: 5,
			pipelineNames: ['Pipeline 1', 'Pipeline 2', 'Pipeline 3', 'Pipeline 4', 'Pipeline 5'],
			JSON: 'This is some JSON from Event 2'
		},
		{
			timestamp: '2023-10-11 09:15 AM',
			eventName: 'Event 3',
			pipelines: 2,
			pipelineNames: ['Pipeline 1', 'Pipeline 2'],
			JSON: 'This is some JSON from Event 3'
		},
		{
			timestamp: '2023-10-11 11:00 AM',
			eventName: 'Event 4',
			pipelines: 1,
			pipelineNames: ['Pipeline 1'],
			JSON: 'This is some JSON from Event 4'
		},
		{
			timestamp: '2023-10-11 11:04 AM',
			eventName: 'Event 5',
			pipelines: 1,
			pipelineNames: ['Pipeline 1'],
			JSON: 'This is some JSON from Event 5'
		},
		{
			timestamp: '2023-10-11 11:05 AM',
			eventName: 'Event 6',
			pipelines: 1,
			pipelineNames: ['Pipeline 1'],
			JSON: 'This is some JSON from Event 6'
		},
		{
			timestamp: '2023-10-11 11:06 AM',
			eventName: 'Event 7',
			pipelines: 1,
			pipelineNames: ['Pipeline 1'],
			JSON: 'This is some JSON from Event 7'
		},
		{
			timestamp: '2023-10-11 11:07 AM',
			eventName: 'Event 8',
			pipelines: 1,
			pipelineNames: ['Pipeline 1'],
			JSON: 'This is some JSON from Event 8'
		},
		{
			timestamp: '2023-10-11 11:08 AM',
			eventName: 'Event 9',
			pipelines: 1,
			pipelineNames: ['Pipeline 1'],
			JSON: 'This is some JSON from Event 9'
		},
		{
			timestamp: '2023-10-11 11:09 AM',
			eventName: 'Event 10',
			pipelines: 1,
			pipelineNames: ['Pipeline 1'],
			JSON: 'This is some JSON from Event 10'
		},
		{
			timestamp: '2023-10-11 11:10 AM',
			eventName: 'Event 11',
			pipelines: 1,
			pipelineNames: ['Pipeline 1'],
			JSON: 'This is some JSON from Event 11'
		},
		{
			timestamp: '2023-10-11 11:11 AM',
			eventName: 'Event 12',
			pipelines: 1,
			pipelineNames: ['Pipeline 1'],
			JSON: 'This is some JSON from Event 12'
		}
	];

	let activeRow = tableData[0];

	function setActiveRow(row: any) {
		activeRow = row;
	}
</script>

<svelte:head>
	<title>Events</title>
	<meta name="description" content="Trigger task flows" />
</svelte:head>

<div class="dark:bg-grain bg-cover bg-norepeat h-screen">
	<HopsNav />

	<!--Page title-->
	<div class="md:pl-20 md:pr-20 mx-8 md:mx-auto mt-16 pb-4">
		<h1 class="text-left text-6xl font-normal mb-4 dark:text-white">Events</h1>

		<h2 class="text-2xl mb-6 text-grey dark:text-white dark:text-opacity-50">Description TBC</h2>
	</div>

	<!-- Events container-->
	<div class="flex space-x-4 pl-8 pr-8 md:pl-20 md:pr-20 text-white">
		<!--Events table container-->
		<div class="w-1/2 bg-[#191919] rounded-lg overflow-scroll h-[450px]">
			<div class="w-full">
				<table class="table-auto w-full">
					<thead class="text-left">
						<tr>
							<th class=" border-r border-b border-almostblack p-2">Timestamp</th>
							<th class=" border-r border-b border-almostblack p-2">Event name</th>
							<th class="border-b border-almostblack p-2">Pipelines</th>
						</tr>
					</thead>
					<tbody>
						{#each tableData as row (row.timestamp)}
							<tr
								class="hover:bg-almostblack {row === activeRow
									? 'border-l-4 border-purple bg-almostblack'
									: ''}"
								on:click={() => setActiveRow(row)}
							>
								<td class=" border-r border-b border-almostblack p-2">{row.timestamp}</td>
								<td class=" border-r border-b border-almostblack p-2">{row.eventName}</td>
								<td class="border-b border-almostblack p-2">{row.pipelines}</td>
							</tr>
						{/each}
					</tbody>
				</table>
			</div>
		</div>

		<!--Selected event detail container-->
		<div class="w-1/2 bg-[#191919] rounded-lg border border-purple overflow-scroll h-[450px]">
			<div class="p-8">
				<p
					class="border rounded-full px-4 py-1 border-grey w-fit mb-2 text-sm font-medium text-lightgrey"
				>
					Tag
				</p>
				<p class="mb-8">{activeRow.eventName}</p>
				<p
					class="border rounded-full px-4 py-1 border-grey w-fit mb-2 text-sm font-medium text-lightgrey"
				>
					Pipelines
				</p>
				<p class="mb-8">{activeRow.pipelineNames}</p>
				<p
					class="border rounded-full px-4 py-1 border-grey w-fit mb-2 text-sm font-medium text-lightgrey"
				>
					JSON
				</p>
				<p class="mb-8">{activeRow.JSON}</p>
			</div>
		</div>
	</div>
</div>
