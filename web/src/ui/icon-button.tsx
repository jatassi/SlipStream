import { Button as BaseButton } from "@base-ui/react/button";
import type { IconSvgElement } from "@hugeicons/react";
import type { ComponentProps } from "react";

import { Icon } from "./icon";

export function IconButton({
  icon,
  label,
  className = "",
  ...props
}: Omit<ComponentProps<typeof BaseButton>, "className" | "children" | "aria-label"> & {
  icon: IconSvgElement;
  label: string;
  className?: string;
}) {
  return (
    <BaseButton
      {...props}
      aria-label={label}
      className={`inline-flex size-(--icon-button-size) shrink-0 items-center justify-center rounded-capsule text-ink-2 transition-[background-color,color,scale] duration-(--duration-control) ease-standard data-disabled:opacity-40 not-data-disabled:hover:bg-fill not-data-disabled:hover:text-ink-1 not-data-disabled:active:scale-[.97] motion-reduce:transition-none motion-reduce:not-data-disabled:active:scale-100 ${className}`}
    >
      <Icon icon={icon} />
    </BaseButton>
  );
}
