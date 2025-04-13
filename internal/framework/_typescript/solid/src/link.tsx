import { makeFinalLinkProps, type RiverLinkPropsBase } from "river.now/client";
import { createMemo, type JSX } from "solid-js";

export function RiverLink(
	props: JSX.AnchorHTMLAttributes<HTMLAnchorElement> &
		RiverLinkPropsBase<JSX.CustomEventHandlersCamelCase<HTMLAnchorElement>["onClick"]>,
) {
	const finalLinkProps = createMemo(() => makeFinalLinkProps(props));

	return (
		<a
			data-external={finalLinkProps().dataExternal}
			{...props}
			onPointerEnter={finalLinkProps().onPointerEnter}
			onFocus={finalLinkProps().onFocus}
			onTouchStart={finalLinkProps().onTouchStart}
			onPointerLeave={finalLinkProps().onPointerLeave}
			onBlur={finalLinkProps().onBlur}
			onTouchCancel={finalLinkProps().onTouchCancel}
			// biome-ignore lint: this onClick is very intentional
			onClick={finalLinkProps().onClick}
		>
			{props.children}
		</a>
	);
}
