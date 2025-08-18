// __TODO add back in any applicable comments from the Go version

export type Params = Record<string, string>;
type SegType = "splat" | "static" | "dynamic" | "index";

type Segment = {
	normalizedVal: string;
	segType: SegType;
};

export type RegisteredPattern = {
	originalPattern: string;
	normalizedPattern: string;
	normalizedSegments: Segment[];
	lastSegType: SegType;
	lastSegIsNonRootSplat: boolean;
	lastSegIsIndex: boolean;
	numberOfDynamicParamSegs: number;
};

type SegmentNode = {
	pattern: string;
	nodeType: number;
	children: Map<string, SegmentNode> | null;
	dynChildren: SegmentNode[];
	paramName: string;
	finalScore: number;
};

type Match = {
	registeredPattern: RegisteredPattern;
	params: Params;
	splatValues: string[];
};

type BestMatch = {
	registeredPattern: RegisteredPattern;
	params: Params;
	splatValues: string[];
	score: number;
};

type FindNestedMatchesResult = {
	params: Params;
	splatValues: string[];
	matches: Match[];
};

export type RegistrationOptions = {
	dynamicParamPrefixRune?: string;
	splatSegmentRune?: string;
	explicitIndexSegment?: string;
};

export type PatternRegistry = {
	staticPatterns: Map<string, RegisteredPattern>;
	dynamicPatterns: Map<string, RegisteredPattern>;
	rootNode: SegmentNode;
	config: {
		dynamicParamPrefixRune: string;
		splatSegmentRune: string;
		explicitIndexSegment: string;
		slashIndexSegment: string;
		usingExplicitIndexSegment: boolean;
	};
};

/////////////////////////////////////////////////////////////////////
/////// CONSTANTS
/////////////////////////////////////////////////////////////////////

const NODE_STATIC = 0;
const NODE_DYNAMIC = 1;
const NODE_SPLAT = 2;
const SCORE_STATIC_MATCH = 2;
const SCORE_DYNAMIC = 1;

const SEG_TYPES = {
	splat: "splat" as SegType,
	static: "static" as SegType,
	dynamic: "dynamic" as SegType,
	index: "index" as SegType,
};

/////////////////////////////////////////////////////////////////////
/////// SHARED UTILITIES
/////////////////////////////////////////////////////////////////////

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

function stripTrailingSlash(pattern: string): string {
	return pattern.length > 0 && pattern[pattern.length - 1] === "/"
		? pattern.substring(0, pattern.length - 1)
		: pattern;
}

/////////////////////////////////////////////////////////////////////
/////// REGISTRATION
/////////////////////////////////////////////////////////////////////

function createSegmentNode(): SegmentNode {
	return {
		pattern: "",
		nodeType: NODE_STATIC,
		children: null,
		dynChildren: [],
		paramName: "",
		finalScore: 0,
	};
}

function findOrCreateChild(node: SegmentNode, segment: string): SegmentNode {
	if (segment === "*" || (segment.length > 0 && segment[0] === ":")) {
		for (const child of node.dynChildren) {
			if (child.paramName === segment.substring(1)) return child;
		}
		return addDynamicChild(node, segment);
	}

	if (node.children === null) node.children = new Map<string, SegmentNode>();

	let child = node.children.get(segment);
	if (child) return child;

	child = createSegmentNode();
	child.nodeType = NODE_STATIC;
	node.children.set(segment, child);
	return child;
}

function addDynamicChild(node: SegmentNode, segment: string): SegmentNode {
	const child = createSegmentNode();
	if (segment === "*") {
		child.nodeType = NODE_SPLAT;
	} else {
		child.nodeType = NODE_DYNAMIC;
		child.paramName = segment.substring(1);
	}
	node.dynChildren.push(child);
	return child;
}

function getSegmentType(
	segment: string,
	dynamicPrefix: string,
	splatRune: string,
): SegType {
	if (segment === "") return SEG_TYPES.index;
	if (segment.length === 1 && segment === splatRune) return SEG_TYPES.splat;
	if (segment.length > 0 && segment[0] === dynamicPrefix)
		return SEG_TYPES.dynamic;
	return SEG_TYPES.static;
}

function isStatic(segments: Segment[]): boolean {
	for (const segment of segments) {
		if (
			segment.segType === SEG_TYPES.splat ||
			segment.segType === SEG_TYPES.dynamic
		) {
			return false;
		}
	}
	return true;
}

function normalizePattern(
	originalPattern: string,
	config: PatternRegistry["config"],
): RegisteredPattern {
	let normalizedPattern = originalPattern;

	if (config.usingExplicitIndexSegment) {
		if (normalizedPattern.endsWith("/")) {
			if (normalizedPattern !== "/") {
				throw new Error(`bad trailing slash: ${originalPattern}`);
			}
			normalizedPattern = normalizedPattern.replace(/\/+$/, "");
		}
		if (normalizedPattern.endsWith(config.slashIndexSegment)) {
			normalizedPattern = normalizedPattern.slice(
				0,
				-config.explicitIndexSegment.length,
			);
		}
	}

	const rawSegments = parseSegments(normalizedPattern);
	const segments: Segment[] = [];
	let numberOfDynamicParamSegs = 0;

	for (const seg of rawSegments) {
		let normalizedVal = seg;
		const segType = getSegmentType(
			seg,
			config.dynamicParamPrefixRune,
			config.splatSegmentRune,
		);

		if (segType === SEG_TYPES.dynamic) {
			numberOfDynamicParamSegs++;
			normalizedVal = ":" + seg.substring(1);
		}
		if (segType === SEG_TYPES.splat) {
			normalizedVal = "*";
		}

		segments.push({ normalizedVal, segType });
	}

	const segLen = segments.length;
	let lastType: SegType =
		segLen > 0 ? segments[segLen - 1]!.segType : SEG_TYPES.static;

	let finalNormalizedPattern = "/";
	for (let i = 0; i < segments.length; i++) {
		finalNormalizedPattern += segments[i]!.normalizedVal;
		if (i < segLen - 1) finalNormalizedPattern += "/";
	}

	if (finalNormalizedPattern.endsWith("/") && lastType !== SEG_TYPES.index) {
		finalNormalizedPattern = finalNormalizedPattern.replace(/\/+$/, "");
	}

	return {
		originalPattern,
		normalizedPattern: finalNormalizedPattern,
		normalizedSegments: segments,
		lastSegType: lastType,
		lastSegIsNonRootSplat: lastType === SEG_TYPES.splat && segLen > 1,
		lastSegIsIndex: lastType === SEG_TYPES.index,
		numberOfDynamicParamSegs,
	};
}

export function createPatternRegistry(
	opts?: RegistrationOptions,
): PatternRegistry {
	const config = {
		dynamicParamPrefixRune: opts?.dynamicParamPrefixRune ?? ":",
		splatSegmentRune: opts?.splatSegmentRune ?? "*",
		explicitIndexSegment: opts?.explicitIndexSegment ?? "",
		slashIndexSegment: "/" + (opts?.explicitIndexSegment ?? ""),
		usingExplicitIndexSegment: (opts?.explicitIndexSegment ?? "") !== "",
	};

	if (config.explicitIndexSegment.includes("/")) {
		throw new Error("explicit index segment cannot contain /");
	}

	return {
		staticPatterns: new Map(),
		dynamicPatterns: new Map(),
		rootNode: createSegmentNode(),
		config,
	};
}

export function registerPattern(
	registry: PatternRegistry,
	originalPattern: string,
): RegisteredPattern {
	const normalized = normalizePattern(originalPattern, registry.config);

	if (
		registry.staticPatterns.has(normalized.normalizedPattern) ||
		registry.dynamicPatterns.has(normalized.normalizedPattern)
	) {
		console.warn(`already registered: ${originalPattern}`);
	}

	if (isStatic(normalized.normalizedSegments)) {
		registry.staticPatterns.set(normalized.normalizedPattern, normalized);
		return normalized;
	}

	registry.dynamicPatterns.set(normalized.normalizedPattern, normalized);

	let current = registry.rootNode;
	let nodeScore = 0;

	for (let i = 0; i < normalized.normalizedSegments.length; i++) {
		const segment = normalized.normalizedSegments[i]!;
		const child = findOrCreateChild(current, segment.normalizedVal);

		if (segment.segType === SEG_TYPES.dynamic) {
			nodeScore += SCORE_DYNAMIC;
		} else if (segment.segType !== SEG_TYPES.splat) {
			nodeScore += SCORE_STATIC_MATCH;
		}

		if (i === normalized.normalizedSegments.length - 1) {
			child.finalScore = nodeScore;
			child.pattern = normalized.normalizedPattern;
		}

		current = child;
	}

	return normalized;
}

/////////////////////////////////////////////////////////////////////
/////// BEST MATCH FINDER
/////////////////////////////////////////////////////////////////////

type DfsBestState = {
	best: BestMatch | null;
	bestScore: number;
	foundMatch: boolean;
};

function dfsBest(
	registry: PatternRegistry,
	node: SegmentNode,
	segments: string[],
	depth: number,
	score: number,
	state: DfsBestState,
	checkTrailingSlash: boolean,
): void {
	const atNormalEnd = checkTrailingSlash && depth === segments.length - 1;

	if (node.pattern.length > 0) {
		const rp = registry.dynamicPatterns.get(node.pattern);
		if (rp) {
			if (
				depth === segments.length ||
				node.nodeType === NODE_SPLAT ||
				atNormalEnd
			) {
				if (!state.foundMatch || score > state.bestScore) {
					state.best = {
						registeredPattern: rp,
						params: {},
						splatValues: [],
						score,
					};
					state.bestScore = score;
					state.foundMatch = true;
				}
			}
		}
	}

	if (depth >= segments.length) return;

	if (node.children !== null) {
		const child = node.children.get(segments[depth]!);
		if (child) {
			dfsBest(
				registry,
				child,
				segments,
				depth + 1,
				score + SCORE_STATIC_MATCH,
				state,
				checkTrailingSlash,
			);

			if (
				state.foundMatch &&
				depth + 1 === segments.length &&
				child.pattern !== ""
			) {
				return;
			}
		}
	}

	for (const child of node.dynChildren) {
		switch (child.nodeType) {
			case NODE_DYNAMIC:
				if (segments[depth] !== "") {
					dfsBest(
						registry,
						child,
						segments,
						depth + 1,
						score + SCORE_DYNAMIC,
						state,
						checkTrailingSlash,
					);
				}
				break;

			case NODE_SPLAT:
				if (child.pattern.length > 0) {
					const rp = registry.dynamicPatterns.get(child.pattern);
					if (rp && !state.foundMatch) {
						state.best = {
							registeredPattern: rp,
							params: {},
							splatValues: [],
							score: 0,
						};
						state.foundMatch = true;
					}
				}
				break;
		}
	}
}

export function findBestMatch(
	registry: PatternRegistry,
	realPath: string,
): BestMatch | null {
	// Check static patterns first
	const rr = registry.staticPatterns.get(realPath);
	if (rr) {
		return {
			registeredPattern: rr,
			params: {},
			splatValues: [],
			score: 0,
		};
	}

	const segments = parseSegments(realPath);
	const hasTrailingSlash =
		realPath.length > 0 && realPath[realPath.length - 1] === "/";

	// Check static pattern without trailing slash
	if (hasTrailingSlash) {
		const pathWithoutTrailingSlash = realPath.substring(
			0,
			realPath.length - 1,
		);
		const rrWithoutSlash = registry.staticPatterns.get(
			pathWithoutTrailingSlash,
		);
		if (rrWithoutSlash) {
			return {
				registeredPattern: rrWithoutSlash,
				params: {},
				splatValues: [],
				score: 0,
			};
		}
	}

	// Search for dynamic patterns
	const state: DfsBestState = {
		best: null,
		bestScore: 0,
		foundMatch: false,
	};

	dfsBest(
		registry,
		registry.rootNode,
		segments,
		0,
		0,
		state,
		hasTrailingSlash,
	);

	if (!state.foundMatch || !state.best) {
		return null;
	}

	// Populate params
	if (state.best.registeredPattern.numberOfDynamicParamSegs > 0) {
		const params: Params = {};
		for (
			let i = 0;
			i < state.best.registeredPattern.normalizedSegments.length;
			i++
		) {
			const seg = state.best.registeredPattern.normalizedSegments[i]!;
			if (seg.segType === SEG_TYPES.dynamic) {
				params[seg.normalizedVal.substring(1)] = segments[i]!;
			}
		}
		state.best.params = params;
	}

	// Populate splat values
	if (
		state.best.registeredPattern.normalizedPattern === "/*" ||
		state.best.registeredPattern.lastSegIsNonRootSplat
	) {
		state.best.splatValues = segments.slice(
			state.best.registeredPattern.normalizedSegments.length - 1,
		);
	}

	return state.best;
}

/////////////////////////////////////////////////////////////////////
/////// NESTED MATCHES FINDER
/////////////////////////////////////////////////////////////////////

function dfsNestedMatches(
	registry: PatternRegistry,
	node: SegmentNode,
	segments: string[],
	depth: number,
	params: Params,
	matches: Map<string, Match>,
): void {
	if (node.pattern.length > 0) {
		const rp = registry.dynamicPatterns.get(node.pattern);
		if (rp && node.pattern !== "/*") {
			const paramsCopy = { ...params };
			let splatValues: string[] = [];

			if (node.nodeType === NODE_SPLAT && depth < segments.length) {
				splatValues = segments.slice(depth);
			}

			matches.set(node.pattern, {
				registeredPattern: rp,
				params: paramsCopy,
				splatValues,
			});

			if (depth === segments.length) {
				const indexPattern = node.pattern + "/";
				const rpIndex = registry.dynamicPatterns.get(indexPattern);
				if (rpIndex) {
					matches.set(indexPattern, {
						registeredPattern: rpIndex,
						params: paramsCopy,
						splatValues: [],
					});
				}
			}
		}
	}

	if (depth >= segments.length) return;

	const seg = segments[depth]!;

	if (node.children !== null) {
		const child = node.children.get(seg);
		if (child) {
			dfsNestedMatches(
				registry,
				child,
				segments,
				depth + 1,
				params,
				matches,
			);
		}
	}

	for (const child of node.dynChildren) {
		switch (child.nodeType) {
			case NODE_DYNAMIC: {
				const oldVal = params[child.paramName];
				const hadVal = oldVal !== undefined;
				params[child.paramName] = seg;

				dfsNestedMatches(
					registry,
					child,
					segments,
					depth + 1,
					params,
					matches,
				);

				if (hadVal) {
					params[child.paramName] = oldVal!;
				} else {
					delete params[child.paramName];
				}
				break;
			}
			case NODE_SPLAT:
				dfsNestedMatches(
					registry,
					child,
					segments,
					depth,
					params,
					matches,
				);
				break;
		}
	}
}

function flattenAndSortMatches(
	matches: Map<string, Match>,
	realPath: string,
	realSegmentLen: number,
): FindNestedMatchesResult | null {
	const results: Match[] = Array.from(matches.values());

	results.sort((i, j) => {
		if (i.registeredPattern.lastSegIsIndex) return 1;
		if (j.registeredPattern.lastSegIsIndex) return -1;
		return (
			i.registeredPattern.normalizedSegments.length -
			j.registeredPattern.normalizedSegments.length
		);
	});

	if (results.length === 0) return null;

	const isNotSlashRoute = realPath !== "" && realPath !== "/";
	if (
		isNotSlashRoute &&
		results.length === 1 &&
		results[0]!.registeredPattern.normalizedPattern === ""
	) {
		return null;
	}

	const lastMatch = results[results.length - 1]!;

	if (
		!lastMatch.registeredPattern.lastSegIsNonRootSplat &&
		lastMatch.registeredPattern.normalizedPattern !== "/*"
	) {
		const patternSegmentsLen =
			lastMatch.registeredPattern.normalizedSegments.length;

		if (patternSegmentsLen < realSegmentLen) return null;

		if (
			patternSegmentsLen === realSegmentLen &&
			lastMatch.registeredPattern.numberOfDynamicParamSegs > 0 &&
			Object.keys(lastMatch.params).length === 0
		) {
			return null;
		}
	}

	return {
		params: lastMatch.params,
		splatValues: lastMatch.splatValues,
		matches: results,
	};
}

export function findNestedMatches(
	registry: PatternRegistry,
	realPath: string,
): FindNestedMatchesResult | null {
	realPath = stripTrailingSlash(realPath);

	const realSegments = parseSegments(realPath);
	const matches: Map<string, Match> = new Map();

	// Check for empty pattern
	const emptyRR = registry.staticPatterns.get("");
	if (emptyRR) {
		matches.set(emptyRR.normalizedPattern, {
			registeredPattern: emptyRR,
			params: {},
			splatValues: [],
		});
	}

	const realSegmentsLen = realSegments.length;

	// Handle root path
	if (realPath === "" || realPath === "/") {
		const rr = registry.staticPatterns.get("/");
		if (rr) {
			matches.set(rr.normalizedPattern, {
				registeredPattern: rr,
				params: {},
				splatValues: [],
			});
		}
		return flattenAndSortMatches(matches, realPath, realSegmentsLen);
	}

	// Check static patterns progressively
	let pb = "";
	let foundFullStatic = false;

	for (let i = 0; i < realSegments.length; i++) {
		pb += "/" + realSegments[i];
		const rr = registry.staticPatterns.get(pb);
		if (rr) {
			matches.set(rr.normalizedPattern, {
				registeredPattern: rr,
				params: {},
				splatValues: [],
			});
			if (i === realSegmentsLen - 1) foundFullStatic = true;
		}
		if (i === realSegmentsLen - 1) {
			pb += "/";
			const rrWithSlash = registry.staticPatterns.get(pb);
			if (rrWithSlash) {
				matches.set(rrWithSlash.normalizedPattern, {
					registeredPattern: rrWithSlash,
					params: {},
					splatValues: [],
				});
			}
		}
	}

	// Check dynamic patterns if no full static match
	if (!foundFullStatic) {
		const rr = registry.dynamicPatterns.get("/*");
		if (rr) {
			matches.set("/*", {
				registeredPattern: rr,
				params: {},
				splatValues: realSegments,
			});
		}

		const params: Params = {};
		dfsNestedMatches(
			registry,
			registry.rootNode,
			realSegments,
			0,
			params,
			matches,
		);
	}

	// Clean up catch-all pattern if necessary
	const hasEmptyRR = emptyRR !== undefined;
	if (matches.has("/*")) {
		if (hasEmptyRR) {
			if (matches.size > 2) matches.delete("/*");
		} else if (matches.size > 1) {
			matches.delete("/*");
		}
	}

	if (matches.size < 2) {
		return flattenAndSortMatches(matches, realPath, realSegmentsLen);
	}

	// Find longest segment matches
	let longestSegmentLen = 0;
	const longestSegmentMatches: Map<string, Match> = new Map();

	for (const match of matches.values()) {
		if (
			match.registeredPattern.normalizedSegments.length >
			longestSegmentLen
		) {
			longestSegmentLen =
				match.registeredPattern.normalizedSegments.length;
		}
	}

	for (const match of matches.values()) {
		if (
			match.registeredPattern.normalizedSegments.length ===
			longestSegmentLen
		) {
			longestSegmentMatches.set(
				match.registeredPattern.lastSegType,
				match,
			);
		}
	}

	// Remove shorter matches
	for (const [pattern, match] of matches) {
		if (
			match.registeredPattern.normalizedSegments.length <
			longestSegmentLen
		) {
			if (
				match.registeredPattern.lastSegIsNonRootSplat ||
				match.registeredPattern.lastSegIsIndex
			) {
				matches.delete(pattern);
			}
		}
	}

	if (matches.size < 2) {
		return flattenAndSortMatches(matches, realPath, realSegmentsLen);
	}

	// Handle conflicts between longest matches
	if (longestSegmentMatches.size > 1) {
		const indexMatch = longestSegmentMatches.get(SEG_TYPES.index);
		if (indexMatch) {
			matches.delete(indexMatch.registeredPattern.normalizedPattern);
		}

		const dynamicExists = longestSegmentMatches.has(SEG_TYPES.dynamic);
		const splatExists = longestSegmentMatches.has(SEG_TYPES.splat);

		if (
			realSegmentsLen === longestSegmentLen &&
			dynamicExists &&
			splatExists
		) {
			const splatMatch = longestSegmentMatches.get(SEG_TYPES.splat)!;
			matches.delete(splatMatch.registeredPattern.normalizedPattern);
		}

		if (
			realSegmentsLen > longestSegmentLen &&
			splatExists &&
			dynamicExists
		) {
			const dynamicMatch = longestSegmentMatches.get(SEG_TYPES.dynamic)!;
			matches.delete(dynamicMatch.registeredPattern.normalizedPattern);
		}
	}

	return flattenAndSortMatches(matches, realPath, realSegmentsLen);
}
