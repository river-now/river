import type {
	ExtractApp,
	PermissivePatternBasedProps,
	RiverAppBase,
	RiverLoaderPattern,
} from "river.now/client";
import {
	__makeFinalLinkProps,
	__resolvePath,
	type RiverAppConfig,
	type RiverLinkPropsBase,
} from "river.now/client";
import { createMemo, mergeProps, splitProps, type JSX } from "solid-js";

export function RiverLink(
	props: JSX.AnchorHTMLAttributes<HTMLAnchorElement> &
		RiverLinkPropsBase<
			JSX.CustomEventHandlersCamelCase<HTMLAnchorElement>["onClick"]
		>,
) {
	const finalLinkProps = createMemo(() => __makeFinalLinkProps(props));
	const [, rest] = splitProps(props, [
		"prefetch",
		"scrollToTop",
		"replace",
		"state",
	]);

	return (
		<a
			data-external={finalLinkProps().dataExternal}
			{...rest}
			onPointerEnter={finalLinkProps().onPointerEnter}
			onFocus={finalLinkProps().onFocus}
			onPointerLeave={finalLinkProps().onPointerLeave}
			onBlur={finalLinkProps().onBlur}
			onTouchCancel={finalLinkProps().onTouchCancel}
			onClick={finalLinkProps().onClick}
		>
			{props.children}
		</a>
	);
}

type TypedRiverLinkProps<
	App extends RiverAppBase,
	Pattern extends RiverLoaderPattern<App> = RiverLoaderPattern<App>,
> = Omit<JSX.AnchorHTMLAttributes<HTMLAnchorElement>, "href" | "pattern"> &
	RiverLinkPropsBase<
		JSX.CustomEventHandlersCamelCase<HTMLAnchorElement>["onClick"]
	> &
	PermissivePatternBasedProps<App, Pattern> & {
		search?: string;
		hash?: string;
	};

export function makeTypedLink<C extends RiverAppConfig>(
	riverAppConfig: C,
	defaultProps?: Partial<
		Omit<
			TypedRiverLinkProps<ExtractApp<C>>,
			"pattern" | "params" | "splatValues"
		>
	>,
) {
	type App = ExtractApp<C>;

	const TypedLink = <Pattern extends RiverLoaderPattern<App>>(
		props: TypedRiverLinkProps<App, Pattern>,
	) => {
		const merged = mergeProps(defaultProps || {}, props);

		const [local, linkProps] = splitProps(merged as any, [
			"pattern",
			"params",
			"splatValues",
			"search",
			"hash",
			"state",
		]);

		const href = createMemo(() => {
			const basePath = __resolvePath({
				riverAppConfig,
				type: "loader",
				props: {
					pattern: local.pattern,
					...(local.params && { params: local.params }),
					...(local.splatValues && {
						splatValues: local.splatValues,
					}),
				},
			});
			const url = new URL(basePath, window.location.origin);
			if (local.search !== undefined) url.search = local.search;
			if (local.hash !== undefined) url.hash = local.hash;
			return url.href;
		});

		return <RiverLink {...linkProps} href={href()} state={local.state} />;
	};

	return TypedLink;
}
