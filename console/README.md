# Hops Console

The UI for the hops console.

Created with:
- Typescript
- svelte/sveltekit
- Tailwind CSS


## Installation

Ensure you have `pnpm` installed and then install the dependencies

```bash
# pnpm can be installed in a few ways, here we use npm. Homebrew, custom installer etc also available
npm install -g pnpm

cd console

pnpm install
```

## Developing

You'll need to start the backend so the UI can fetch data, first create a hops config:

```bash
mkdir -p ~/.hops && cp dsl/testdata/valid.hops ~/.hops/main.hops
```

Then start hops

```bash
go run main.go start -d --address=0.0.0.0:8916
```

Once you've installed dependencies with `pnpm install`, start a development server:

```bash
# All pnpm commands are run from within the console dir
cd console

pnpm run dev

# or start the server and open the app in a new browser tab
pnpm run dev -- --open
```

## Building

To create a production version of your app:

```bash
pnpm run build
```

You can preview the production build with `npm run preview`.


## Misc Info

**Icons**

We use the [Phosphor](https://icones.js.org/collection/ph) icon set on [Icônes](https://icones.js.org/). Default should be the duotone variation.

Select `svelte` for an icon, then copy into the icons folder. We also add the same `script` block to each icon, in addition to adding an attribute `class` to allow custom styling. See existing icons for examples.
