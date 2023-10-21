import { base } from '$app/paths';

export const abs = (path: string): string => {
	const abspath = base + path;
	return abspath.replace(/\/$/, '');
};

export const isActive = (path: string, currentPathname: string): boolean => {
	return currentPathname.replace(/\/$/, '') === path;
};
