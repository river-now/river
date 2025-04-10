import { makeFinalLinkProps, type RiverLinkPropsBase } from "@sjc5/river/client";
import { type ComponentProps, useMemo } from "react";

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
			onPointerLeave={finalLinkProps.onPointerLeave}
			onBlur={finalLinkProps.onBlur}
			// biome-ignore lint: this onClick is very intentional
			onClick={finalLinkProps.onClick}
		>
			{props.children}
		</a>
	);
}
