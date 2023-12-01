<script lang="ts">
	export let text: string;
</script>

<div class="text-lightgrey text-base flex items-start space-x-2">
	<img src="/images/info-icon.svg" alt="info icon" />
	<p>
		{@html text}
	</p>
</div>
