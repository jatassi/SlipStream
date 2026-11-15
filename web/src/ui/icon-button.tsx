import { Button as BaseButton } from "@base-ui/react/button";
import type { IconSvgElement } from "@hugeicons/react";
import type { ComponentProps } from "react";

import { Icon } from "./icon";
import { Spinner } from "./spinner";

export function IconButton({
  icon,
  label,
  busy = false,
  disabled = false,
  className = "",
  ...props
}: Omit<
  ComponentProps<typeof BaseButton>,
  "className" | "children" | "aria-label" | "focusableWhenDisabled"
> & {
  icon: IconSvgElement;
  label: string;
  /** A spinner takes the icon's place, and the button keeps its focus but can't be pressed. */
  busy?: boolean;
  className?: string;
}) {
  return (
    <BaseButton
      {...props}
      disabled={disabled || busy}
      focusableWhenDisabled={busy}
      aria-busy={busy || undefined}
      aria-label={label}
      className={`inline-flex size-(--icon-button-size) shrink-0 items-center justify-center rounded-capsule text-ink-2 transition-[background-color,color,scale] duration-(--duration-control) ease-standard data-disabled:opacity-40 not-data-disabled:hover:bg-fill not-data-disabled:hover:text-ink-1 not-data-disabled:active:scale-[.97] motion-reduce:transition-none motion-reduce:not-data-disabled:active:scale-100 ${busy ? "data-disabled:opacity-100" : ""} ${className}`}
    >
      {busy ? <Spinner /> : <Icon icon={icon} />}
    </BaseButton>
  );
}
