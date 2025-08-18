export function parseSegments(path: string): string[] {
	if (path === "") return [];
	if (path === "/") return [""];

	let startIdx = path[0] === "/" ? 1 : 0;
	let maxSegments = 0;

	for (let i = startIdx; i < path.length; i++) {
		if (path[i] === "/") maxSegments++;
	}
	if (path.length > 0) maxSegments++;
	if (maxSegments === 0) return [];

	const segs: string[] = [];
	let start = startIdx;

	for (let i = startIdx; i < path.length; i++) {
		if (path[i] === "/") {
			if (i > start) segs.push(path.substring(start, i));
			start = i + 1;
		}
	}

	if (start < path.length) segs.push(path.substring(start));
	if (path.length > 0 && path[path.length - 1] === "/") segs.push("");

	return segs;
}
