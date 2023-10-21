// See https://kit.svelte.dev/docs/types#app
// for information about these interfaces
declare global {
	namespace App {
		// interface Error {}
		// interface Locals {}
		// interface PageData {}
		// interface Platform {}
		interface Locals {
			colorTheme: import('$lib/types').ColorTheme;
		}
		// interface Platform {}
		interface Session {
			colorTheme: import('$lib/types').ColorTheme;
		}
	}
}

export {};
