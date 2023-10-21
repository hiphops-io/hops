export interface Task {
	display_name: string;
	name: string;
	summary: string;
	description: string;
	emoji?: string;
	params: Param[];
}

export interface TaskSummary {
	display_name: string;
	name: string;
	emoji: string;
}

export interface BaseParam {
	name: string;
	display_name: string;
	help?: string;
	flag?: string;
	shortflag?: string;
	required: boolean;
}

export interface StrParam extends BaseParam {
	type: 'string';
	default?: string;
}

export interface TextParam extends BaseParam {
	type: 'text';
	default?: string;
}

export interface BoolParam extends BaseParam {
	type: 'bool';
	default?: boolean;
}

export interface NumberParam extends BaseParam {
	type: 'number';
	default?: number;
}

export type Param = StrParam | TextParam | BoolParam | NumberParam;

export interface TaskRunResponse {
	errors: { [key: string]: string[] };
	message: string;
	sequence_id: string;
}
