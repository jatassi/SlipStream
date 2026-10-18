import { Collapsible } from "@base-ui/react/collapsible";
import { type ReactNode, useId } from "react";

import { Chevron } from "./chevron";

/**
 * A card whose title opens and closes its body. The header keeps aside, such
 * as a count and a switch, in reach while the body is closed.
 */
export function DisclosureCard({
  title,
  open,
  onOpenChange,
  aside,
  children,
}: {
  title: string;
  open: boolean;
  onOpenChange: (open: boolean) => void;
  aside?: ReactNode;
  children: ReactNode;
}) {
  const titleId = useId();
  return (
    <Collapsible.Root
      open={open}
      onOpenChange={onOpenChange}
      render={<section aria-labelledby={titleId} />}
      className="overflow-hidden rounded-card bg-fill ring-1 ring-hairline ring-inset"
    >
      <div className="flex min-h-11 items-center gap-3 px-4">
        <h3 id={titleId} className="flex-1 font-semibold text-ink-1">
          <Collapsible.Trigger className="group -mx-2 inline-flex h-(--control-height) items-center gap-1.5 rounded-item px-2">
            <Chevron />
            {title}
          </Collapsible.Trigger>
        </h3>
        {aside}
      </div>
      <Collapsible.Panel className="border-t border-hairline">{children}</Collapsible.Panel>
    </Collapsible.Root>
  );
}
