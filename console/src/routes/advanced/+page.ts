import { PUBLIC_BACKEND_URL } from '$env/static/public';
import ky from 'ky';

import { eventToTable, type EventLog, ago } from '$lib/events/api';
import type { PageLoad } from './$types';

export const load: PageLoad = async () => {
	const eventLog: EventLog = await ky.get(`${PUBLIC_BACKEND_URL}/events?sourceonly=true`).json();
	const events = eventLog.event_items;

	const tableData = events.map((eventItem) => eventToTable(eventItem));

	return {
		ago: ago(eventLog.start_timestamp),
		tableData
	};
};
