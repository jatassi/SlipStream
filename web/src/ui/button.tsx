import { Button as BaseButton } from "@base-ui/react/button";
import type { IconSvgElement } from "@hugeicons/react";
import type { ComponentProps, ReactNode } from "react";

import { Icon } from "./icon";
import { Spinner } from "./spinner";

export type ButtonLook = "default" | "primary" | "ghost" | "fail" | "warn" | "on-fail";
export type ButtonSize = "default" | "small";

const looks: { readonly [L in ButtonLook]: string } = {
  default:
    "bg-fill text-ink-1 ring-1 ring-hairline-strong ring-inset not-data-disabled:hover:bg-fill-hover",
  primary: "bg-solid text-on-solid",
  ghost:
    "bg-transparent text-ink-2 not-data-disabled:hover:bg-fill not-data-disabled:hover:text-ink-1",
  fail: "bg-fail text-fail-deep",
  warn: "bg-warn text-warn-deep",
  "on-fail":
    "bg-[rgba(255,220,216,.1)] text-fail-text ring-1 ring-[rgba(255,220,216,.16)] ring-inset",
};

const smallHeight = "h-7 px-3 in-data-[density=touch]:h-(--control-height)";

function height(look: ButtonLook, size: ButtonSize): string {
  if (size === "small") {
    return smallHeight;
  }
  return look === "primary" ? "h-(--primary-height) px-3.5" : "h-(--control-height) px-3.5";
}

export function Button({
  look = "default",
  size = "default",
  icon,
  busy = false,
  disabled = false,
  className = "",
  children,
  ...props
}: Omit<ComponentProps<typeof BaseButton>, "className" | "children" | "focusableWhenDisabled"> & {
  look?: ButtonLook;
  size?: ButtonSize;
  icon?: IconSvgElement | undefined;
  busy?: boolean;
  className?: string;
  children: ReactNode;
}) {
  return (
    <BaseButton
      {...props}
      disabled={disabled || busy}
      focusableWhenDisabled={busy}
      aria-busy={busy || undefined}
      className={`inline-flex shrink-0 items-center justify-center gap-2 rounded-capsule text-button font-semibold whitespace-nowrap transition-[background-color,color,scale] duration-(--duration-control) ease-standard select-none in-data-[density=touch]:px-4.5 in-data-[density=touch]:text-body-touch in-data-[density=touch]:font-semibold not-data-disabled:active:scale-[.97] motion-reduce:transition-none motion-reduce:not-data-disabled:active:scale-100 ${looks[look]} ${height(look, size)} ${disabled ? "opacity-40" : ""} ${className}`}
    >
      {busy ? <Spinner /> : icon === undefined ? null : <Icon icon={icon} className="size-4" />}
      {children}
    </BaseButton>
  );
}
