<script>
	import { page } from '$app/stores';
</script>

<div class="flex flex-col justify-center items-center self-center h-[70vh]">
	<h1 class="text-3xl font-extralight">Something went wrong | {$page?.status}</h1>
	<br />
	<pre class="text-sm font-mono text-gray-500">{$page?.error?.message}</pre>
</div>
