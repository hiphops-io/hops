<script>
	import { page } from '$app/stores';
	import ErrorPage from '../lib/errors/ErrorPage.svelte';
	import NotFound from '../lib/errors/NotFound.svelte';
</script>

{#if $page?.status == 404}
	<NotFound />
{:else}
	<ErrorPage />
{/if}
