type Fn = (...args: Array<any>) => any;

export function debounce<T extends Fn>(
	fn: T,
	delayInMs: number,
): (...args: Parameters<T>) => Promise<Awaited<ReturnType<T>>> {
	let timeoutID: number;

	return (...args: Parameters<T>) =>
		new Promise<Awaited<ReturnType<T>>>((resolve) => {
			clearTimeout(timeoutID);
			timeoutID = window.setTimeout(() => {
				resolve(fn(...args));
			}, delayInMs);
		});
}
