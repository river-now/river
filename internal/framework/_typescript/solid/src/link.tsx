import type { RiverUntypedFunction } from "river.now/client";
import {
	type APIConfig,
	makeFinalLinkProps,
	resolvePath,
	type RiverLinkPropsBase,
	type SharedBase,
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
	F extends RiverUntypedFunction,
	Pattern extends F["pattern"] = F["pattern"],
> = Omit<JSX.AnchorHTMLAttributes<HTMLAnchorElement>, "href"> &
	RiverLinkPropsBase<
		JSX.CustomEventHandlersCamelCase<HTMLAnchorElement>["onClick"]
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
		const merged = mergeProps(defaultProps || {}, props);
		const { pattern, params, splatValues, options, ...linkProps } =
			merged as any;

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

		return <RiverLink {...linkProps} href={href} />;
	};

	return TypedLink;
}
