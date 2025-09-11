export function logInfo(message?: any, ...optionalParams: Array<any>) {
	console.log("River:", message, ...optionalParams);
}

export function logError(message?: any, ...optionalParams: Array<any>) {
	console.error("River:", message, ...optionalParams);
}
