import { HugeiconsIcon, type IconSvgElement } from "@hugeicons/react";

export function Icon({ icon, className }: { icon: IconSvgElement; className?: string }) {
  return <HugeiconsIcon icon={icon} size={20} aria-hidden="true" className={className} />;
}
