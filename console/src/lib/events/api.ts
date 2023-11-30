import TimeAgo from 'javascript-time-ago';
import en from 'javascript-time-ago/locale/en';

export interface Event {
	hops: {
		action: string;
		event: string;
		source: string;
	};

	timestamp: number;

	// eslint-disable-next-line @typescript-eslint/no-explicit-any
	[key: string]: any;
}

export interface EventLog {
	start_timestamp: string;
	end_timestamp: string;
	event_items: EventItem[];
}

export interface EventItem {
	event: Event;
	sequence_id: string;
	timestamp: string;
}

export interface EventTable {
	timestamp: string; // '2023-11-22T10:44:00.518137754Z';
	eventId: string;
	event: string;
	source: string;
	action: string;
	JSON: {
		// eslint-disable-next-line @typescript-eslint/no-explicit-any
		[key: string]: any;
	};
}

export function eventToTable(eventItem: EventItem): EventTable {
	return {
		timestamp: new Date(eventItem.timestamp).toLocaleString(),
		eventId: eventItem.sequence_id,
		event: eventItem.event?.hops?.event || '',
		source: eventItem.event?.hops?.source || '',
		action: eventItem.event?.hops?.action || '',
		JSON: eventItem.event
	};
}

TimeAgo.addLocale(en);
const timeAgo = new TimeAgo('en');

export function ago(startTimestamp: string): string {
	const then = new Date(startTimestamp);
	console.log(timeAgo.format(then));
	return timeAgo.format(then);
}
