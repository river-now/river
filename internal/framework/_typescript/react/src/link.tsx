import { memo, type ComponentProps, type JSX } from "react";
import type {
	ExtractApp,
	PermissivePatternBasedProps,
	RiverAppBase,
	RiverLoaderPattern,
} from "river.now/client";
import {
	makeFinalLinkProps,
	resolvePath,
	type RiverAppConfig,
	type RiverLinkPropsBase,
} from "river.now/client";

export const RiverLink = memo(function RiverLink(
	props: ComponentProps<"a"> &
		RiverLinkPropsBase<
			(
				e: React.MouseEvent<HTMLAnchorElement, MouseEvent>,
			) => void | Promise<void>
		>,
) {
	const finalLinkProps = makeFinalLinkProps(props);
	// oxlint-disable-next-line no-unused-vars
	const { prefetch, scrollToTop, replace, state, ...rest } = props;

	return (
		<a
			data-external={finalLinkProps.dataExternal}
			{...(rest as any)}
			onPointerEnter={finalLinkProps.onPointerEnter}
			onFocus={finalLinkProps.onFocus}
			onPointerLeave={finalLinkProps.onPointerLeave}
			onBlur={finalLinkProps.onBlur}
			onTouchCancel={finalLinkProps.onTouchCancel}
			onClick={finalLinkProps.onClick}
		>
			{props.children}
		</a>
	);
});

type TypedRiverLinkProps<
	App extends RiverAppBase,
	Pattern extends RiverLoaderPattern<App> = RiverLoaderPattern<App>,
> = Omit<ComponentProps<"a">, "href" | "pattern"> &
	RiverLinkPropsBase<
		(
			e: React.MouseEvent<HTMLAnchorElement, MouseEvent>,
		) => void | Promise<void>
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
		const {
			pattern,
			params,
			splatValues,
			search,
			hash,
			state,
			...linkProps
		} = props as any;

		const href = resolvePath({
			riverAppConfig,
			type: "loader",
			props: {
				pattern,
				...(params && { params }),
				...(splatValues && { splatValues }),
			},
		});

		const url = new URL(href, window.location.origin);
		if (search !== undefined) url.search = search;
		if (hash !== undefined) url.hash = hash;

		const finalProps = {
			...defaultProps,
			...linkProps,
			href: url.href,
			state,
		};

		return <RiverLink {...finalProps} />;
	};

	const MemoizedTypedLink = memo(TypedLink) as <
		Pattern extends RiverLoaderPattern<App>,
	>(
		props: TypedRiverLinkProps<App, Pattern>,
	) => JSX.Element;

	(MemoizedTypedLink as any).displayName =
		`TypedLink(${Object.keys(defaultProps || {}).join(", ")})`;

	return MemoizedTypedLink;
}
