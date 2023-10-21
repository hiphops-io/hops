import { error } from '@sveltejs/kit';
import type { PageLoad } from './$types';
import { PUBLIC_BACKEND_URL } from '$env/static/public';
import ky from 'ky';

import type { Task } from '$lib/tasks/api';

// TODO: Create an endpoint for fetching a single task entry
//       on the backend and use that instead.
export const load: PageLoad = async ({ params }) => {
	const tasks: Task[] = await ky.get(`${PUBLIC_BACKEND_URL}/tasks`).json();

	const task = tasks.find((task) => {
		return task.name === params.task_name;
	});

	if (task === undefined) {
		throw error(404, 'Task not found');
	}

	return { task: task };
};
