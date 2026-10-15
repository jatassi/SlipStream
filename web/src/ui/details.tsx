import { Collapsible } from "@base-ui/react/collapsible";
import type { ReactNode } from "react";

import { Chevron } from "./chevron";

export function Details({
  summary,
  action,
  children,
}: {
  summary: ReactNode;
  action?: ReactNode;
  children: ReactNode;
}) {
  return (
    <Collapsible.Root className="flex flex-col items-start gap-1 text-body text-ink-1 in-data-[density=touch]:text-body-touch">
      <p>{summary}</p>
      {action}
      <Collapsible.Trigger className="group -mx-2 inline-flex h-(--control-height) items-center gap-1 rounded-item px-2 font-medium text-ink-2 hover:text-ink-1">
        <Chevron />
        Details
      </Collapsible.Trigger>
      <Collapsible.Panel className="text-ink-2">{children}</Collapsible.Panel>
    </Collapsible.Root>
  );
}
