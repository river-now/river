// matcher.ts
type Params = Record<string, string>;
type Pattern = string;
type SegType = "splat" | "static" | "dynamic" | "index";
type PatternsMap = Map<Pattern, RegisteredPattern>;
type MatchesMap = Map<Pattern, Match>;

// Constants matching Go version
const nodeStatic = 0;
const nodeDynamic = 1;
const nodeSplat = 2;
const scoreStaticMatch = 2;
const scoreDynamic = 1;

class RegisteredPattern {
	originalPattern: string;
	normalizedPattern: string;
	normalizedSegments: segment[];
	lastSegType: SegType;
	lastSegIsNonRootSplat: boolean;
	lastSegIsIndex: boolean;
	numberOfDynamicParamSegs: number;

	constructor(
		originalPattern: string,
		normalizedPattern: string,
		normalizedSegments: segment[],
		lastSegType: SegType,
		lastSegIsNonRootSplat: boolean,
		lastSegIsIndex: boolean,
		numberOfDynamicParamSegs: number,
	) {
		this.originalPattern = originalPattern;
		this.normalizedPattern = normalizedPattern;
		this.normalizedSegments = normalizedSegments;
		this.lastSegType = lastSegType;
		this.lastSegIsNonRootSplat = lastSegIsNonRootSplat;
		this.lastSegIsIndex = lastSegIsIndex;
		this.numberOfDynamicParamSegs = numberOfDynamicParamSegs;
	}

	NormalizedPattern(): string {
		return this.normalizedPattern;
	}

	NormalizedSegments(): segment[] {
		return this.normalizedSegments;
	}

	OriginalPattern(): string {
		return this.originalPattern;
	}
}

class segment {
	normalizedVal: string;
	segType: SegType;

	constructor(normalizedVal: string, segType: SegType) {
		this.normalizedVal = normalizedVal;
		this.segType = segType;
	}
}

class segmentNode {
	pattern: string = "";
	nodeType: number = nodeStatic;
	children: Map<string, segmentNode> | null = null;
	dynChildren: segmentNode[] = [];
	paramName: string = "";
	finalScore: number = 0;

	// findOrCreateChild finds or creates a child node for a segment
	findOrCreateChild(segment: string): segmentNode {
		if (segment === "*" || (segment.length > 0 && segment[0] === ":")) {
			for (const child of this.dynChildren) {
				if (child.paramName === segment.substring(1)) {
					return child;
				}
			}
			return this.addDynamicChild(segment);
		}

		if (this.children === null) {
			this.children = new Map<string, segmentNode>();
		}
		let child = this.children.get(segment);
		if (child) {
			return child;
		}
		child = new segmentNode();
		child.nodeType = nodeStatic;
		this.children.set(segment, child);
		return child;
	}

	// addDynamicChild creates a new dynamic or splat child node
	addDynamicChild(segment: string): segmentNode {
		const child = new segmentNode();
		if (segment === "*") {
			child.nodeType = nodeSplat;
		} else {
			child.nodeType = nodeDynamic;
			child.paramName = segment.substring(1);
		}
		this.dynChildren.push(child);
		return child;
	}
}

class Match {
	RegisteredPattern: RegisteredPattern;
	params: Params;
	splatValues: string[];

	constructor(
		RegisteredPattern: RegisteredPattern,
		params: Params = {},
		splatValues: string[] = [],
	) {
		this.RegisteredPattern = RegisteredPattern;
		this.params = params;
		this.splatValues = splatValues;
	}

	get originalPattern(): string {
		return this.RegisteredPattern.originalPattern;
	}
	get normalizedPattern(): string {
		return this.RegisteredPattern.normalizedPattern;
	}
	get normalizedSegments(): segment[] {
		return this.RegisteredPattern.normalizedSegments;
	}
	get lastSegType(): SegType {
		return this.RegisteredPattern.lastSegType;
	}
	get lastSegIsNonRootSplat(): boolean {
		return this.RegisteredPattern.lastSegIsNonRootSplat;
	}
	get lastSegIsIndex(): boolean {
		return this.RegisteredPattern.lastSegIsIndex;
	}
	get numberOfDynamicParamSegs(): number {
		return this.RegisteredPattern.numberOfDynamicParamSegs;
	}
}

class BestMatch {
	RegisteredPattern: RegisteredPattern | null = null;
	Params: Params = {};
	SplatValues: string[] = [];
	score: number = 0;

	ensureRegisteredPattern(): RegisteredPattern {
		if (!this.RegisteredPattern) {
			throw new Error("No registered pattern.");
		}
		return this.RegisteredPattern;
	}
	get originalPattern(): string {
		return this.ensureRegisteredPattern().originalPattern;
	}
	get normalizedPattern(): string {
		return this.ensureRegisteredPattern().normalizedPattern;
	}
	get normalizedSegments(): segment[] {
		return this.ensureRegisteredPattern().normalizedSegments || [];
	}
	get lastSegType(): SegType {
		return this.ensureRegisteredPattern().lastSegType;
	}
	get lastSegIsNonRootSplat(): boolean {
		return this.ensureRegisteredPattern().lastSegIsNonRootSplat || false;
	}
	get lastSegIsIndex(): boolean {
		return this.ensureRegisteredPattern().lastSegIsIndex || false;
	}
	get numberOfDynamicParamSegs(): number {
		return this.ensureRegisteredPattern().numberOfDynamicParamSegs || 0;
	}
}

interface Options {
	DynamicParamPrefixRune?: string; // Optional. Defaults to ':'.
	SplatSegmentRune?: string; // Optional. Defaults to '*'.
	// Optional. Defaults to empty string (effectively a trailing slash in the pattern).
	// Could also be something like "_index" if preferred by the user.
	ExplicitIndexSegment?: string;
	Quiet?: boolean; // Optional. Defaults to false. Set to true if you want to quash warnings.
}

interface FindNestedMatchesResults {
	Params: Params;
	SplatValues: string[];
	Matches: Match[];
}

class Matcher {
	private staticPatterns: PatternsMap;
	private dynamicPatterns: PatternsMap;
	private rootNode: segmentNode;
	private explicitIndexSegment: string;
	private dynamicParamPrefixRune: string;
	private splatSegmentRune: string;
	private slashIndexSegment: string;
	private usingExplicitIndexSegment: boolean;
	private quiet: boolean;

	constructor(opts?: Options) {
		this.staticPatterns = new Map<Pattern, RegisteredPattern>();
		this.dynamicPatterns = new Map<Pattern, RegisteredPattern>();
		this.rootNode = new segmentNode();

		const mungedOpts = mungeOptsToDefaults(opts);
		this.explicitIndexSegment = mungedOpts.ExplicitIndexSegment!;
		this.dynamicParamPrefixRune = mungedOpts.DynamicParamPrefixRune!;
		this.splatSegmentRune = mungedOpts.SplatSegmentRune!;
		this.quiet = mungedOpts.Quiet!;
		this.slashIndexSegment = "/" + this.explicitIndexSegment;
		this.usingExplicitIndexSegment = this.explicitIndexSegment !== "";
	}

	GetExplicitIndexSegment(): string {
		return this.explicitIndexSegment;
	}

	GetDynamicParamPrefixRune(): string {
		return this.dynamicParamPrefixRune;
	}

	GetSplatSegmentRune(): string {
		return this.splatSegmentRune;
	}

	NormalizePattern(originalPattern: string): RegisteredPattern {
		let normalizedPattern = originalPattern;

		// if using an index sig
		if (this.usingExplicitIndexSegment) {
			// ignore trailing slashes
			if (normalizedPattern.endsWith("/")) {
				if (normalizedPattern !== "/") {
					throw new Error(
						`Error with pattern '${originalPattern}'. With the exception of any absolute root pattern ('/'), trailing slashes are not permitted when using an explicit index segment. If you intend to make this an index route, add your explicit index segment. Otherwise, remove the trailing slash.`,
					);
				}
				normalizedPattern = normalizedPattern.replace(/\/+$/, "");
			}
			// if is an idx route, clear the sig, but leave the trailing slash
			if (normalizedPattern.endsWith(this.slashIndexSegment)) {
				normalizedPattern = normalizedPattern.slice(
					0,
					-this.explicitIndexSegment.length,
				);
			}

			// Now patterns with a trailing slash are index routes, and those without a trailing
			// slash are non-index routes. This means that the normalized pattern for the "true"
			// root would be an empty string, whereas the normalized pattern for the index route
			// would be a single slash.
		}

		const rawSegments = ParseSegments(normalizedPattern);
		const segments: segment[] = [];

		let numberOfDynamicParamSegs = 0;

		for (const seg of rawSegments) {
			let normalizedVal = seg;

			const segType = this.getSegmentTypeAssumeNormalized(seg);
			if (segType === "dynamic") {
				numberOfDynamicParamSegs++;
				normalizedVal = ":" + seg.substring(1);
			}
			if (segType === "splat") {
				normalizedVal = "*";
			}

			segments.push(new segment(normalizedVal, segType));
		}

		const segLen = segments.length;
		let lastType: SegType = "static";
		if (segLen > 0) {
			const lastSegment = segments[segLen - 1];
			if (lastSegment) {
				lastType = lastSegment.segType;
			}
		}

		let finalNormalizedPattern = "/";
		for (let i = 0; i < segments.length; i++) {
			const segment = segments[i];
			if (segment) {
				finalNormalizedPattern += segment.normalizedVal;
				if (i < segLen - 1) {
					finalNormalizedPattern += "/";
				}
			}
		}

		if (finalNormalizedPattern.endsWith("/") && lastType !== "index") {
			finalNormalizedPattern = finalNormalizedPattern.replace(/\/+$/, "");
		}

		return new RegisteredPattern(
			originalPattern,
			finalNormalizedPattern,
			segments,
			lastType,
			lastType === "splat" && segLen > 1,
			lastType === "index",
			numberOfDynamicParamSegs,
		);
	}

	RegisterPattern(originalPattern: string): RegisteredPattern {
		const _normalized = this.NormalizePattern(originalPattern);

		if (this.staticPatterns.has(_normalized.normalizedPattern)) {
			if (!this.quiet) {
				console.warn(
					getAppropriateWarningMsg(
						originalPattern,
						this.usingExplicitIndexSegment,
					),
				);
			}
		}
		if (this.dynamicPatterns.has(_normalized.normalizedPattern)) {
			if (!this.quiet) {
				console.warn(
					getAppropriateWarningMsg(
						originalPattern,
						this.usingExplicitIndexSegment,
					),
				);
			}
		}

		if (getIsStatic(_normalized.normalizedSegments)) {
			this.staticPatterns.set(_normalized.normalizedPattern, _normalized);
			return _normalized;
		}

		this.dynamicPatterns.set(_normalized.normalizedPattern, _normalized);

		let current = this.rootNode;
		let nodeScore = 0;

		for (let i = 0; i < _normalized.normalizedSegments.length; i++) {
			const segment = _normalized.normalizedSegments[i];
			if (!segment) continue;

			const child = current.findOrCreateChild(segment.normalizedVal);

			if (segment.segType === "dynamic") {
				nodeScore += scoreDynamic;
			} else if (segment.segType !== "splat") {
				nodeScore += scoreStaticMatch;
			}

			if (i === _normalized.normalizedSegments.length - 1) {
				child.finalScore = nodeScore;
				child.pattern = _normalized.normalizedPattern;
			}

			current = child;
		}

		return _normalized;
	}

	FindBestMatch(realPath: string): [BestMatch | null, boolean] {
		let rr = this.staticPatterns.get(realPath);
		if (rr) {
			const best = new BestMatch();
			best.RegisteredPattern = rr;
			return [best, true];
		}

		const segments = ParseSegments(realPath);
		const hasTrailingSlash =
			realPath.length > 0 && realPath[realPath.length - 1] === "/";

		if (hasTrailingSlash) {
			const pathWithoutTrailingSlash = realPath.substring(
				0,
				realPath.length - 1,
			);
			rr = this.staticPatterns.get(pathWithoutTrailingSlash);
			if (rr) {
				const best = new BestMatch();
				best.RegisteredPattern = rr;
				return [best, true];
			}
		}

		const best = new BestMatch();
		const state = { bestScore: 0, foundMatch: false };

		this.dfsBest(
			this.rootNode,
			segments,
			0,
			0,
			best,
			state,
			hasTrailingSlash,
		);

		if (!state.foundMatch) {
			return [null, false];
		}

		if (best.numberOfDynamicParamSegs > 0) {
			const params: Params = {};
			for (let i = 0; i < best.normalizedSegments.length; i++) {
				const seg = best.normalizedSegments[i];
				const segmentValue = segments[i];
				if (
					seg &&
					seg.segType === "dynamic" &&
					segmentValue !== undefined
				) {
					params[seg.normalizedVal.substring(1)] = segmentValue;
				}
			}
			best.Params = params;
		}

		if (best.normalizedPattern === "/*" || best.lastSegIsNonRootSplat) {
			best.SplatValues = segments.slice(
				best.normalizedSegments.length - 1,
			);
		}

		return [best, true];
	}

	private dfsBest(
		node: segmentNode,
		segments: string[],
		depth: number,
		score: number,
		best: BestMatch,
		state: { bestScore: number; foundMatch: boolean },
		checkTrailingSlash: boolean,
	): void {
		const atNormalEnd = checkTrailingSlash && depth === segments.length - 1;

		if (node.pattern.length > 0) {
			const rp = this.dynamicPatterns.get(node.pattern);
			if (rp) {
				if (
					depth === segments.length ||
					node.nodeType === nodeSplat ||
					atNormalEnd
				) {
					if (!state.foundMatch || score > state.bestScore) {
						best.RegisteredPattern = rp;
						best.score = score;
						state.bestScore = score;
						state.foundMatch = true;
					}
				}
			}
		}

		if (depth >= segments.length) {
			return;
		}

		if (node.children !== null) {
			const segmentAtDepth = segments[depth];
			if (segmentAtDepth !== undefined) {
				const child = node.children.get(segmentAtDepth);
				if (child) {
					this.dfsBest(
						child,
						segments,
						depth + 1,
						score + scoreStaticMatch,
						best,
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
		}

		for (const child of node.dynChildren) {
			switch (child.nodeType) {
				case nodeDynamic:
					// Don't match empty segments to dynamic parameters
					if (
						segments[depth] !== undefined &&
						segments[depth] !== ""
					) {
						this.dfsBest(
							child,
							segments,
							depth + 1,
							score + scoreDynamic,
							best,
							state,
							checkTrailingSlash,
						);
					}
					break;
				case nodeSplat:
					if (child.pattern.length > 0) {
						const rp = this.dynamicPatterns.get(child.pattern);
						if (rp && !state.foundMatch) {
							best.RegisteredPattern = rp;
							state.foundMatch = true;
						}
					}
					break;
			}
		}
	}

	FindNestedMatches(
		realPath: string,
	): [FindNestedMatchesResults | null, boolean] {
		const realSegments = ParseSegments(realPath);
		const matches: MatchesMap = new Map();

		const emptyRR = this.staticPatterns.get("");
		const hasEmptyRR = emptyRR !== undefined;

		if (hasEmptyRR) {
			matches.set(emptyRR.normalizedPattern, new Match(emptyRR));
		}

		if (realPath === "" || realPath === "/") {
			const rr = this.staticPatterns.get("/");
			if (rr) {
				matches.set(rr.normalizedPattern, new Match(rr));
			}
			return flattenAndSortMatches(matches, realPath);
		}

		let pb = "";
		let foundFullStatic = false;
		for (let i = 0; i < realSegments.length; i++) {
			const segment = realSegments[i];
			if (segment !== undefined) {
				pb += "/" + segment;
				const rr = this.staticPatterns.get(pb);
				if (rr) {
					matches.set(rr.normalizedPattern, new Match(rr));
					if (i === realSegments.length - 1) {
						foundFullStatic = true;
					}
				}
				if (i === realSegments.length - 1) {
					pb += "/";
					const rr = this.staticPatterns.get(pb);
					if (rr) {
						matches.set(rr.normalizedPattern, new Match(rr));
					}
				}
			}
		}

		if (!foundFullStatic) {
			// For the catch-all pattern (e.g., "/*"), handle it specially
			const rr = this.dynamicPatterns.get("/*");
			if (rr) {
				matches.set("/*", new Match(rr, {}, realSegments));
			}

			// DFS for the rest of the matches
			const params: Params = {};
			this.dfsNestedMatches(
				this.rootNode,
				realSegments,
				0,
				params,
				matches,
			);
		}

		// if there are multiple matches and a catch-all, remove the catch-all
		// UNLESS the sole other match is an empty str pattern
		if (matches.has("/*")) {
			if (hasEmptyRR) {
				if (matches.size > 2) {
					matches.delete("/*");
				}
			} else if (matches.size > 1) {
				matches.delete("/*");
			}
		}

		if (matches.size < 2) {
			return flattenAndSortMatches(matches, realPath);
		}

		let longestSegmentLen = 0;
		const longestSegmentMatches: MatchesMap = new Map();
		for (const match of matches.values()) {
			if (match.normalizedSegments.length > longestSegmentLen) {
				longestSegmentLen = match.normalizedSegments.length;
			}
		}
		for (const match of matches.values()) {
			if (match.normalizedSegments.length === longestSegmentLen) {
				longestSegmentMatches.set(match.lastSegType, match);
			}
		}

		// if there is any splat or index with a segment length shorter than longest segment length, remove it
		for (const [pattern, match] of matches) {
			if (match.normalizedSegments.length < longestSegmentLen) {
				if (match.lastSegIsNonRootSplat || match.lastSegIsIndex) {
					matches.delete(pattern);
				}
			}
		}

		if (matches.size < 2) {
			return flattenAndSortMatches(matches, realPath);
		}

		// if the longest segment length items are (1) dynamic, (2) splat, or (3) index, remove them as follows:
		// - if the realSegmentLen equals the longest segment length, prioritize dynamic, then splat, and always remove index
		// - if the realSegmentLen is greater than the longest segment length, prioritize splat, and always remove dynamic and index
		if (longestSegmentMatches.size > 1) {
			const indexMatch = longestSegmentMatches.get("index");
			if (indexMatch) {
				matches.delete(indexMatch.normalizedPattern);
			}

			const dynamicExists = longestSegmentMatches.has("dynamic");
			const splatExists = longestSegmentMatches.has("splat");

			if (
				realSegments.length === longestSegmentLen &&
				dynamicExists &&
				splatExists
			) {
				const splatMatch = longestSegmentMatches.get("splat")!;
				matches.delete(splatMatch.normalizedPattern);
			}
			if (
				realSegments.length > longestSegmentLen &&
				splatExists &&
				dynamicExists
			) {
				const dynamicMatch = longestSegmentMatches.get("dynamic")!;
				matches.delete(dynamicMatch.normalizedPattern);
			}
		}

		return flattenAndSortMatches(matches, realPath);
	}

	private dfsNestedMatches(
		node: segmentNode,
		segments: string[],
		depth: number,
		params: Params,
		matches: MatchesMap,
	): void {
		if (node.pattern.length > 0) {
			const rp = this.dynamicPatterns.get(node.pattern);
			if (rp) {
				// Don't process the ultimate catch-all here
				if (node.pattern !== "/*") {
					// Copy params
					const paramsCopy = { ...params };

					let splatValues: string[] = [];
					if (
						node.nodeType === nodeSplat &&
						depth < segments.length
					) {
						// For splat nodes, collect all remaining segments
						splatValues = segments.slice(depth);
					}

					const match = new Match(rp, paramsCopy, splatValues);
					matches.set(node.pattern, match);

					// Check for index segment if we're at the exact depth
					if (depth === segments.length) {
						const indexPattern = node.pattern + "/";
						const rpIndex = this.dynamicPatterns.get(indexPattern);
						if (rpIndex) {
							matches.set(
								indexPattern,
								new Match(rpIndex, paramsCopy),
							);
						}
					}
				}
			}
		}

		// If we've consumed all segments, stop
		if (depth >= segments.length) {
			return;
		}

		const seg = segments[depth];
		if (seg === undefined) {
			return;
		}

		// Try static children
		if (node.children !== null) {
			const child = node.children.get(seg);
			if (child) {
				this.dfsNestedMatches(
					child,
					segments,
					depth + 1,
					params,
					matches,
				);
			}
		}

		// Try dynamic/splat children
		for (const child of node.dynChildren) {
			switch (child.nodeType) {
				case nodeDynamic: {
					// Backtracking pattern for dynamic
					const oldVal = params[child.paramName];
					const hadVal = oldVal !== undefined;
					params[child.paramName] = seg;

					this.dfsNestedMatches(
						child,
						segments,
						depth + 1,
						params,
						matches,
					);

					if (hadVal && oldVal !== undefined) {
						params[child.paramName] = oldVal;
					} else {
						delete params[child.paramName];
					}
					break;
				}
				case nodeSplat:
					// For splat nodes, we collect remaining segments and don't increment depth
					this.dfsNestedMatches(
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

	private getSegmentTypeAssumeNormalized(segment: string): SegType {
		if (segment === "") {
			return "index";
		}
		if (segment.length === 1 && segment === this.splatSegmentRune) {
			return "splat";
		}
		if (segment.length > 0 && segment[0] === this.dynamicParamPrefixRune) {
			return "dynamic";
		}
		return "static";
	}
}

// Helper functions
function HasLeadingSlash(pattern: string): boolean {
	return pattern.length > 0 && pattern[0] === "/";
}

function HasTrailingSlash(pattern: string): boolean {
	return pattern.length > 0 && pattern[pattern.length - 1] === "/";
}

function EnsureLeadingSlash(pattern: string): string {
	if (!HasLeadingSlash(pattern)) {
		return "/" + pattern;
	}
	return pattern;
}

function EnsureTrailingSlash(pattern: string): string {
	if (!HasTrailingSlash(pattern)) {
		return pattern + "/";
	}
	return pattern;
}

function StripLeadingSlash(pattern: string): string {
	if (HasLeadingSlash(pattern)) {
		return pattern.substring(1);
	}
	return pattern;
}

function StripTrailingSlash(pattern: string): string {
	if (HasTrailingSlash(pattern)) {
		return pattern.substring(0, pattern.length - 1);
	}
	return pattern;
}

function JoinPatterns(rp: RegisteredPattern, pattern: string): string {
	const base = rp.normalizedPattern;
	let result = base;

	const patternHasLeadingSlash = HasLeadingSlash(pattern);

	if (HasTrailingSlash(base) && patternHasLeadingSlash) {
		pattern = pattern.substring(1);
	} else if (!patternHasLeadingSlash) {
		result += "/";
	}

	result += pattern;

	return result;
}

function getAppropriateWarningMsg(
	pattern: string,
	usingExplicitIndexSegment: boolean,
): string {
	const base = `Pattern '${pattern}' is already registered.`;
	if (usingExplicitIndexSegment) {
		return (
			base +
			" When you use an explicit index segment, trailing slashes are ignored, which may be the reason for your effectively duplicated patterns."
		);
	}
	return base;
}

function getIsStatic(segments: segment[]): boolean {
	if (segments.length > 0) {
		for (const segment of segments) {
			if (segment.segType === "splat" || segment.segType === "dynamic") {
				return false;
			}
		}
	}
	return true;
}

function mungeOptsToDefaults(opts?: Options): Required<Options> {
	const copy: Required<Options> = {
		DynamicParamPrefixRune: opts?.DynamicParamPrefixRune ?? ":",
		SplatSegmentRune: opts?.SplatSegmentRune ?? "*",
		ExplicitIndexSegment: opts?.ExplicitIndexSegment ?? "",
		Quiet: opts?.Quiet ?? false,
	};

	if (copy.ExplicitIndexSegment.includes("/")) {
		throw new Error("explicit index segment cannot contain a slash");
	}

	return copy;
}

function ParseSegments(path: string): string[] {
	// Fast path for common cases
	if (path === "") {
		return [];
	}
	if (path === "/") {
		return [""];
	}

	// Skip leading slash
	let startIdx = 0;
	if (path[0] === "/") {
		startIdx = 1;
	}

	// Maximum potential segments
	let maxSegments = 0;
	for (let i = startIdx; i < path.length; i++) {
		if (path[i] === "/") {
			maxSegments++;
		}
	}

	// Add one more for the final segment
	if (path.length > 0) {
		maxSegments++;
	}

	if (maxSegments === 0) {
		return [];
	}

	const segs: string[] = [];
	let start = startIdx;

	for (let i = startIdx; i < path.length; i++) {
		if (path[i] === "/") {
			if (i > start) {
				segs.push(path.substring(start, i));
			}
			start = i + 1;
		}
	}

	// Add final segment
	if (start < path.length) {
		segs.push(path.substring(start));
	}

	if (path.length > 0 && path[path.length - 1] === "/") {
		// Add empty string for trailing slash
		segs.push("");
	}

	return segs;
}

function flattenAndSortMatches(
	matches: MatchesMap,
	realPath: string,
): [FindNestedMatchesResults | null, boolean] {
	const results: Match[] = Array.from(matches.values());

	results.sort((i, j) => {
		// if any match is an index, it should be last
		if (i.lastSegIsIndex) {
			return 1;
		}
		if (j.lastSegIsIndex) {
			return -1;
		}

		// else sort by segment length
		return i.normalizedSegments.length - j.normalizedSegments.length;
	});

	if (results.length === 0) {
		return [null, false];
	}

	// if not slash route and solely matched "", then invalid
	const isNotSlashRoute = realPath !== "" && realPath !== "/";
	const firstResult = results[0];
	if (
		isNotSlashRoute &&
		results.length === 1 &&
		firstResult &&
		firstResult.normalizedPattern === ""
	) {
		return [null, false];
	}

	const lastMatch = results[results.length - 1];
	if (!lastMatch) {
		return [null, false];
	}

	return [
		{
			Params: lastMatch.params,
			SplatValues: lastMatch.splatValues,
			Matches: results,
		},
		true,
	];
}

// Exports
export {
	BestMatch,
	EnsureLeadingSlash,
	EnsureTrailingSlash,
	HasLeadingSlash,
	HasTrailingSlash,
	JoinPatterns,
	Match,
	Matcher,
	ParseSegments,
	RegisteredPattern,
	StripLeadingSlash,
	StripTrailingSlash,
	type FindNestedMatchesResults,
	type Options,
	type Params,
};
