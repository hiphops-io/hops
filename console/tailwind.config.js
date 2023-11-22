/** @type {import('tailwindcss').Config}*/
const config = {
	content: [
		'./src/**/*.{html,js,svelte,ts}',
		'./node_modules/flowbite-svelte/**/*.{html,js,svelte,ts}'
	],

	plugins: [require('flowbite/plugin')],

	darkMode: 'class',

	theme: {
		extend: {
			colors: {
				purple: '#CD72FC',
				lightpurple: '#F9EFFF',
				grey: '#666',
				lightgrey: '#EEE',
				midgrey: '#DDD',
				black: '#000',
				white: '#fff',
				error: '#FF4D4D'
				// flowbite-svelte
			}
		}
	}
};

module.exports = config;
