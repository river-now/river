import { makeFinalLinkProps, type RiverLinkPropsBase } from "river.now/client";
import { createMemo, type JSX } from "solid-js";

export function RiverLink(
	props: JSX.AnchorHTMLAttributes<HTMLAnchorElement> &
		RiverLinkPropsBase<
			JSX.CustomEventHandlersCamelCase<HTMLAnchorElement>["onClick"]
		>,
) {
	const finalLinkProps = createMemo(() => makeFinalLinkProps(props));
	const { prefetch, scrollToTop, ...rest } = props;

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
