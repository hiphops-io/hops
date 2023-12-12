<script lang="ts">
	let showCode = false;

	function toggleShowCode() {
		showCode = !showCode;
	}
</script>

<!--Code window & detail/output container-->
<div class="w-full h-auto space-y-2 px-4">
	<!--Automation actions-->
	<div class="w-full flex justify-between items-center py-2">
		<div>
			<p class="text-nines font-semibold text-sm">activeFilename</p>
		</div>

		<div class="flex space-x-6 items-center">
			<button on:click={toggleShowCode} class="text-nines font-semibold flex text-sm space-x-2">
				<img src="images/desktop_app/code-icon.svg" alt="Code icon" />
				<p>{showCode ? 'Hide code' : 'Show code'}</p>
			</button>
			<button
				class="bg-purple font-semibold flex px-2 py-1 text-sm rounded-sm space-x-2 items-center"
			>
				<img src="images/desktop_app/publish-icon.svg" alt="Publish icon" />
				<p>Publish</p>
			</button>
		</div>
	</div>

	<div class="flex !m-0">
		<!--Code window-->
		<div class="bg-almostblack h-[660px] rounded-lg flex {showCode ? 'w-[50%] mr-2' : 'hidden'}">
			<p class="text-error m-auto">Hops code</p>
		</div>

		<!--Detail/output window-->
		<div class="bg-almostblack h-[660px] rounded-lg flex {showCode ? ' w-[50%]' : 'w-full'}">
			<p class="text-error m-auto">Detail/output panel</p>
		</div>

		<!--END code window & detail/output container-->
	</div>

	<!--Instruction container-->
	<div class="bg-almostblack w-full h-[160px] rounded-lg" />
</div>
