import { h, type JSX } from "preact";
import { useMemo } from "preact/hooks";
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

export function RiverLink(
	props: JSX.HTMLAttributes<HTMLAnchorElement> &
		RiverLinkPropsBase<
			(
				e: JSX.TargetedMouseEvent<HTMLAnchorElement>,
			) => void | Promise<void>
		>,
) {
	const finalLinkProps = useMemo(() => makeFinalLinkProps(props), [props]);
	// oxlint-disable-next-line no-unused-vars
	const { prefetch, scrollToTop, replace, ...rest } = props;

	return h(
		"a",
		{
			"data-external": finalLinkProps.dataExternal,
			...(rest as any),
			onPointerEnter: finalLinkProps.onPointerEnter,
			onFocus: finalLinkProps.onFocus,
			onPointerLeave: finalLinkProps.onPointerLeave,
			onBlur: finalLinkProps.onBlur,
			onTouchCancel: finalLinkProps.onTouchCancel,
			onClick: finalLinkProps.onClick,
		},
		props.children,
	);
}

type TypedRiverLinkProps<
	App extends RiverAppBase,
	Pattern extends RiverLoaderPattern<App> = RiverLoaderPattern<App>,
> = Omit<JSX.HTMLAttributes<HTMLAnchorElement>, "href" | "pattern"> &
	RiverLinkPropsBase<
		(e: JSX.TargetedMouseEvent<HTMLAnchorElement>) => void | Promise<void>
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
		const { pattern, params, splatValues, ...linkProps } = props as any;

		const href = resolvePath({
			riverAppConfig,
			type: "loader",
			props: {
				pattern,
				...(params && { params }),
				...(splatValues && { splatValues }),
			},
		});

		const finalProps = { ...defaultProps, ...linkProps, href };
		return <RiverLink {...finalProps} />;
	};

	TypedLink.displayName = `TypedLink(${Object.keys(defaultProps || {}).join(", ")})`;
	return TypedLink;
}
