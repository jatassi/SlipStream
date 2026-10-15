import { createLink } from "@tanstack/react-router";
import type { ComponentPropsWithRef } from "react";

// The brand leads to the first nav row without being a place of its own, so
// it never claims aria-current: the nav row it shares a path with does.
function BrandAnchor({ "aria-current": _current, children, ...props }: ComponentPropsWithRef<"a">) {
  return <a {...props}>{children}</a>;
}

export const BrandLink = createLink(BrandAnchor);
