import type { RiverAppConfig } from "./api_client_helpers.ts";

export type RiverAppBase = {
	routes: readonly any[];
	appConfig: RiverAppConfig;
	rootData: any;
};

export type ExtractApp<C extends RiverAppConfig> = C["__phantom"];

type RiverLoader<App extends RiverAppBase> = Extract<
	App["routes"][number],
	{ _type: "loader" }
>;

type RiverQuery<App extends RiverAppBase> = Extract<
	App["routes"][number],
	{ _type: "query" }
>;

type RiverMutation<App extends RiverAppBase> = Extract<
	App["routes"][number],
	{ _type: "mutation" }
>;

export type RiverLoaderPattern<App extends RiverAppBase> =
	RiverLoader<App>["pattern"];

export type RiverQueryPattern<App extends RiverAppBase> =
	RiverQuery<App>["pattern"];

export type RiverMutationPattern<App extends RiverAppBase> =
	RiverMutation<App>["pattern"];

export type RiverLoaderOutput<
	App extends RiverAppBase,
	P extends RiverLoaderPattern<App>,
> = Extract<RiverLoader<App>, { pattern: P }>["phantomOutputType"];

export type RiverQueryInput<
	App extends RiverAppBase,
	P extends RiverQueryPattern<App>,
> = Extract<RiverQuery<App>, { pattern: P }>["phantomInputType"];

export type RiverQueryOutput<
	App extends RiverAppBase,
	P extends RiverQueryPattern<App>,
> = Extract<RiverQuery<App>, { pattern: P }>["phantomOutputType"];

export type RiverMutationInput<
	App extends RiverAppBase,
	P extends RiverMutationPattern<App>,
> = Extract<RiverMutation<App>, { pattern: P }>["phantomInputType"];

export type RiverMutationOutput<
	App extends RiverAppBase,
	P extends RiverMutationPattern<App>,
> = Extract<RiverMutation<App>, { pattern: P }>["phantomOutputType"];

export type RiverMutationMethod<
	App extends RiverAppBase,
	P extends RiverMutationPattern<App>,
> =
	Extract<RiverMutation<App>, { pattern: P }> extends { method: infer M }
		? M extends string
			? M
			: "POST"
		: "POST";

export type GetParams<App extends RiverAppBase, P extends string> =
	Extract<App["routes"][number], { pattern: P }> extends {
		params: ReadonlyArray<infer Params>;
	}
		? Params extends string
			? Params
			: never
		: never;

export type RiverRouteParams<
	App extends RiverAppBase,
	P extends RiverLoaderPattern<App>,
> = GetParams<App, P>;

export type HasParams<App extends RiverAppBase, P extends string> =
	GetParams<App, P> extends never ? false : true;

export type IsSplat<App extends RiverAppBase, P extends string> =
	Extract<App["routes"][number], { pattern: P }> extends { isSplat: true }
		? true
		: false;

export type IsEmptyInput<T> = [T] extends [null | undefined | never]
	? true
	: false;

export type PatternBasedProps<App extends RiverAppBase, P extends string> = {
	pattern: P;
} & (HasParams<App, P> extends true
	? IsSplat<App, P> extends true
		? {
				params: { [K in GetParams<App, P>]: string };
				splatValues: Array<string>;
			}
		: {
				params: { [K in GetParams<App, P>]: string };
			}
	: IsSplat<App, P> extends true
		? {
				splatValues: Array<string>;
			}
		: {});

export type WithOptionalInput<TInput> =
	IsEmptyInput<TInput> extends true ? { input?: TInput } : { input: TInput };

export type RiverRoutePropsGeneric<
	JSXElement,
	App extends RiverAppBase,
	P extends RiverLoaderPattern<App>,
> = {
	idx: number;
	Outlet: (props: Record<string, any>) => JSXElement;
	__phantom_pattern: P;
} & Record<string, any>;
