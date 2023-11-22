import { PUBLIC_BACKEND_URL } from '$env/static/public';
import ky from 'ky';

import { eventToTable, type EventLog } from '$lib/events/api';
import type { PageLoad } from './$types';

export const load: PageLoad = async () => {
	const events: EventLog[] = await ky.get(`${PUBLIC_BACKEND_URL}/events`).json();

	const tableData = events.map((eventLog) => eventToTable(eventLog));

	return { tableData };
};
