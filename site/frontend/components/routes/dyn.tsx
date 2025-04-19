import { useCurrentAppData } from "../app_utils.ts";

export function Dyn() {
	return <div>{useCurrentAppData().params.dyn}</div>;
}
