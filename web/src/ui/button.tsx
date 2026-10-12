import { Button as BaseButton } from "@base-ui/react/button";
import type { ComponentProps } from "react";

const look = {
  primary: "bg-solid text-on-solid",
  default: "bg-fill text-ink-1 ring-1 ring-hairline ring-inset",
} as const;

export function Button({
  variant = "default",
  className,
  ...props
}: ComponentProps<typeof BaseButton> & { variant?: keyof typeof look }) {
  return (
    <BaseButton
      {...props}
      className={`inline-flex items-center gap-2 rounded-full px-4 py-2 font-semibold ${look[variant]} ${typeof className === "string" ? className : ""}`}
    />
  );
}
