import { writable } from 'svelte/store';

export const activeItemStore = writable('New automation');
export const activeAutomationStore = writable({});
