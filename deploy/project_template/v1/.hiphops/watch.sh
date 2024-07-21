deno run --allow-all --no-code-cache --unstable-worker-options --watch=$(deno eval -p 'Deno.args.join(",")' ./hiphops/flows/**/*.{ts,js}),mod.ts mod.ts
