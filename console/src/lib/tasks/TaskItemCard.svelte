<script lang="ts">
	import type { TaskSummary } from './api';

	export let command: TaskSummary;
</script>

<a href="/console/{command.name}">
	<div
		class="px-6 py-8 bg-grey bg-opacity-5 rounded-xl dark:bg-white dark:bg-opacity-10 border border-white dark:border-black hover:border dark:hover:border hover:border-purple dark:hover:border-purple"
	>
		<h3 class="text-4xl mb-4">{command.emoji || '▶️'}</h3>
		<p class="text-black font-medium dark:text-white">{command.display_name}</p>
	</div>
</a>
