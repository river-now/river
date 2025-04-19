/////////////////////////////////////////////////////////////////////
// GENERAL UTILS
/////////////////////////////////////////////////////////////////////

export function isAbortError(error: unknown) {
	return error instanceof Error && error.name === "AbortError";
}

export function LogInfo(message?: any, ...optionalParams: Array<any>) {
	console.log("River:", message, ...optionalParams);
}

export function LogError(message?: any, ...optionalParams: Array<any>) {
	console.error("River:", message, ...optionalParams);
}

export function Panic(msg?: string): never {
	LogError("Panic");
	throw new Error(msg ?? "panic");
}
