type Fn = (...args: Array<any>) => any;

export function debounce<T extends Fn>(fn: T, delayInMs: number): T {
	let timeoutID: number | undefined;

	return ((...args: Array<any>) => {
		clearTimeout(timeoutID);
		timeoutID = window.setTimeout(() => {
			fn(...args);
		}, delayInMs);
	}) as T;
}
