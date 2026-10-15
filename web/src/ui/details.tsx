import { Collapsible } from "@base-ui/react/collapsible";
import { ArrowRight01Icon } from "@hugeicons/core-free-icons";
import type { ReactNode } from "react";

import { Icon } from "./icon";

export function Details({ summary, children }: { summary: ReactNode; children: ReactNode }) {
  return (
    <Collapsible.Root className="flex flex-col items-start gap-1 text-body text-ink-1 in-data-[density=touch]:text-body-touch">
      <p>{summary}</p>
      <Collapsible.Trigger className="group -mx-2 inline-flex h-(--control-height) items-center gap-1 rounded-item px-2 font-medium text-ink-2 hover:text-ink-1">
        <Icon
          icon={ArrowRight01Icon}
          className="size-4 transition-[rotate] duration-(--duration-chevron) ease-standard group-data-panel-open:rotate-90 motion-reduce:transition-none"
        />
        Details
      </Collapsible.Trigger>
      <Collapsible.Panel className="text-ink-2">{children}</Collapsible.Panel>
    </Collapsible.Root>
  );
}
