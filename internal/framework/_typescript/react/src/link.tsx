import { type ComponentProps, useMemo } from "react";
import { makeFinalLinkProps, type RiverLinkPropsBase } from "river.now/client";

export function RiverLink(
	props: ComponentProps<"a"> &
		RiverLinkPropsBase<
			(e: React.MouseEvent<HTMLAnchorElement, MouseEvent>) => void | Promise<void>
		>,
) {
	const finalLinkProps = useMemo(() => makeFinalLinkProps(props), [props]);

	return (
		<a
			data-external={finalLinkProps.dataExternal}
			{...(props as any)}
			onPointerEnter={finalLinkProps.onPointerEnter}
			onFocus={finalLinkProps.onFocus}
			onTouchStart={finalLinkProps.onTouchStart}
			onPointerLeave={finalLinkProps.onPointerLeave}
			onBlur={finalLinkProps.onBlur}
			onTouchCancel={finalLinkProps.onTouchCancel}
			// biome-ignore lint: this onClick is very intentional
			onClick={finalLinkProps.onClick}
		>
			{props.children}
		</a>
	);
}
