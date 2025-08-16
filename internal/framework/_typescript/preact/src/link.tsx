import { h, type JSX } from "preact";
import { useMemo } from "preact/hooks";
import type { RiverUntypedFunction } from "river.now/client";
import {
	makeFinalLinkProps,
	resolvePath,
	type APIConfig,
	type RiverLinkPropsBase,
	type SharedBase,
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
	F extends RiverUntypedFunction,
	Pattern extends F["pattern"] = F["pattern"],
> = Omit<JSX.HTMLAttributes<HTMLAnchorElement>, "href"> &
	RiverLinkPropsBase<
		(e: JSX.TargetedMouseEvent<HTMLAnchorElement>) => void | Promise<void>
	> &
	Omit<SharedBase<Pattern, F>, "options">;

export function makeTypedLink<F extends RiverUntypedFunction>(
	apiConfig: APIConfig,
	defaultProps?: Partial<
		Omit<TypedRiverLinkProps<F>, "pattern" | "params" | "splatValues">
	>,
) {
	const TypedLink = <Pattern extends F["pattern"]>(
		props: TypedRiverLinkProps<F, Pattern>,
	) => {
		const { pattern, params, splatValues, options, ...linkProps } =
			props as any;

		const pathProps: SharedBase<Pattern, F> = {
			pattern,
			options,
			...(params && { params }),
			...(splatValues && { splatValues }),
		};

		const href = resolvePath({
			apiConfig,
			type: "loader",
			props: pathProps as any,
		});

		const finalProps = { ...defaultProps, ...linkProps, href };
		return <RiverLink {...(finalProps as any)} />;
	};

	TypedLink.displayName = `TypedLink(${Object.keys(defaultProps || {}).join(", ")})`;
	return TypedLink;
}
