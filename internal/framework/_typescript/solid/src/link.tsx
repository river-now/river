import type {
	ExtractApp,
	PatternBasedProps,
	RiverAppBase,
	RiverLoaderPattern,
} from "river.now/client";
import {
	makeFinalLinkProps,
	resolvePath,
	type RiverAppConfig,
	type RiverLinkPropsBase,
} from "river.now/client";
import { createMemo, type JSX, mergeProps } from "solid-js";

export function RiverLink(
	props: JSX.AnchorHTMLAttributes<HTMLAnchorElement> &
		RiverLinkPropsBase<
			JSX.CustomEventHandlersCamelCase<HTMLAnchorElement>["onClick"]
		>,
) {
	const finalLinkProps = createMemo(() => makeFinalLinkProps(props));
	// oxlint-disable-next-line no-unused-vars
	const { prefetch, scrollToTop, replace, ...rest } = props;

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
	PatternBasedProps<App, Pattern>;

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
		const { pattern, params, splatValues, ...linkProps } = merged as any;

		const href = resolvePath({
			riverAppConfig,
			type: "loader",
			props: {
				pattern,
				...(params && { params }),
				...(splatValues && { splatValues }),
			},
		});

		return <RiverLink {...linkProps} href={href} />;
	};

	return TypedLink;
}
