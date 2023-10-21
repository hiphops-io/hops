<script lang="ts">
	let classProp: string = '';
	export { classProp as class };
</script>

<div
	class={`
    dark:bg-slate-800 bg-white 
    rounded-2xl dark:border dark:border-slate-700
    shadow px-5 py-2
    ${classProp}
  `}
>
	<slot />
</div>
