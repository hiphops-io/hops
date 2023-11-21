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

export interface EventTable {
	timestamp: string; // '2023-10-10 11:00 AM';
	eventId: string;
	eventName: string;
	source: string;
	action: string;
	JSON: {
		// eslint-disable-next-line @typescript-eslint/no-explicit-any
		[key: string]: any;
	};
}

export function eventToTable(event: Event): EventTable {
	return {
		timestamp: new Date(event.timestamp).toLocaleString(),
		eventId: event.hops.event,
		eventName: event.hops.event,
		source: event.hops.source,
		action: event.hops.action,
		JSON: event
	};
}
