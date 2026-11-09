import { Collapsible } from "@base-ui/react/collapsible";
import type { ReactNode } from "react";

import { Chevron } from "./chevron";

export function Details({
  summary,
  action,
  open,
  onOpenChange,
  fields = false,
  children,
}: {
  /** The plain-language sentence it leads with; left out where the row it sits in already says it. */
  summary?: ReactNode;
  action?: ReactNode;
  open?: boolean;
  onOpenChange?: (open: boolean) => void;
  /** Holds form fields: the panel takes the form's width and keeps them mounted while closed, so each keeps its value and error. */
  fields?: boolean;
  children: ReactNode;
}) {
  return (
    <Collapsible.Root
      {...(open !== undefined && { open })}
      {...(onOpenChange && { onOpenChange })}
      className="flex flex-col items-start gap-1 text-body text-ink-1 in-data-[density=touch]:text-body-touch"
    >
      {summary !== undefined && <p>{summary}</p>}
      {action}
      <Collapsible.Trigger className="group -mx-2 inline-flex h-(--control-height) items-center gap-1 rounded-item px-2 font-medium text-ink-2 hover:text-ink-1">
        <Chevron />
        Details
      </Collapsible.Trigger>
      <Collapsible.Panel
        keepMounted={fields}
        className={`text-ink-2 ${fields ? "self-stretch" : ""}`}
      >
        {children}
      </Collapsible.Panel>
    </Collapsible.Root>
  );
}
