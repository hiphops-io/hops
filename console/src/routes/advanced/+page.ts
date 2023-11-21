import { PUBLIC_BACKEND_URL } from '$env/static/public';
import ky from 'ky';

import { eventToTable, type Event } from '$lib/events/api';
import type { PageLoad } from './$types';

export const load: PageLoad = async () => {
	const events: Event[] = await ky.get(`${PUBLIC_BACKEND_URL}/events`).json();

	const tableData = events.map((event) => eventToTable(event));

	return { tableData };
};
