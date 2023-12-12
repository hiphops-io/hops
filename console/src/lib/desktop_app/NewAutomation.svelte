<script lang="ts">
	import { claim_space } from 'svelte/internal';

	// Automations dummy data
	export let dummyTemplateData: any;
</script>

<div class="h-auto w-full bg-almostblack m-4 px-16 py-8 rounded-md">
	<div class="space-y-12">
		<div>
			<p class="text-white text-4xl">New automation</p>
		</div>

		<div>
			<p class="text-nines text-xl font-medium mb-4">Choose a template</p>

			<div class="grid grid-cols-3 gap-4 pb-16">
				{#each dummyTemplateData as template}
					<div class="p-6 border border-white border-opacity-10 rounded-lg w-full space-y-4">
						<p class="text-3xl">{template.emoji}</p>
						<p class="font-semibold text-white">{template.name}</p>
					</div>
				{/each}
			</div>

			<div>
				<p class="text-nines text-xl font-medium mb-2">Create your own</p>
				<button
					><p class="text-white text-opacity-80 text-base font-medium">
						Learn how to create a custom automation ->
					</p></button
				>
			</div>
		</div>
	</div>
</div>
