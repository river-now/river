import { ChildrenPermissive } from "../types.js";
import { cx } from "../utils/utils.js";

function UnorderedList({
  children,
  ...rest
}: { children: ChildrenPermissive } & JSX.IntrinsicElements["ul"]) {
  return (
    <ul {...rest} class={cx("flex-col-wrapper-bigger", rest.class)}>
      {children}
    </ul>
  );
}

function ListItem({
  children,
  ...rest
}: { children: ChildrenPermissive } & JSX.IntrinsicElements["li"]) {
  return (
    <li {...rest} class={cx("list-item", rest.class)}>
      {children}
    </li>
  );
}

export { UnorderedList, ListItem };
