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

export function eventToTable(eventLog: EventLog): EventTable {
	return {
		timestamp: new Date(eventLog.timestamp).toLocaleString(),
		eventId: eventLog.sequence_id,
		event: eventLog.event.hops.event,
		source: eventLog.event.hops.source,
		action: eventLog.event.hops.action || '',
		JSON: eventLog.event
	};
}
