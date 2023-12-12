<script lang="ts">
	import AutomationDetailView from '$lib/desktop_app/AutomationDetailView.svelte';
	import SidebarNav from '$lib/desktop_app/SidebarNav.svelte';
	import { activeItemStore } from '$lib/store';
</script>

<svelte:head>
	<title>Hiphops</title>
	<meta name="description" content="Desktop app" />
</svelte:head>

<!--Page container-->
<div class="flex">
	<!--Sidebar-->
	<SidebarNav />

	<AutomationDetailView />
</div>
