import { ArrowRight01Icon } from "@hugeicons/core-free-icons";

import { Icon } from "./icon";

export function Chevron() {
  return (
    <Icon
      icon={ArrowRight01Icon}
      className="size-4 shrink-0 transition-[rotate] duration-(--duration-chevron) ease-standard group-data-panel-open:rotate-90 motion-reduce:transition-none"
    />
  );
}
