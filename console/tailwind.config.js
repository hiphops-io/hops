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
				black: '#191919',
				almostblack: '#282828',
				white: '#fff',
				error: '#FF4D4D',
				// flowbite-svelte

				//Desktop colors
				nines: '#999',
				twodee: '#2D2D2D'
			},
			backgroundImage: {
				grain: "url('/images/light-grain.png')"
			},
			boxShadow: {
				purpleglow: '0px 0px 24px 4px rgba(205, 114, 252)'
			}
		}
	}
};

module.exports = config;
