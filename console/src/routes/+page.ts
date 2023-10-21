import type { PageLoad } from './$types';
import { PUBLIC_BACKEND_URL } from '$env/static/public';
import ky from 'ky';

export const load: PageLoad = async () => {
	let tasks: unknown;

	try {
		tasks = await ky.get(`${PUBLIC_BACKEND_URL}/tasks`).json();
	} catch (error) {
		// TODO: Would be better to let users know something went wrong,
		//       rather than just show empty
		tasks = [];
	}

	return { tasks };
};
