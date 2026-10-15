import { Input } from "@base-ui/react/input";
import { Search01Icon } from "@hugeicons/core-free-icons";

import { Icon } from "./icon";

export function SearchField({
  label,
  value,
  onValueChange,
  placeholder,
  className = "",
}: {
  label: string;
  value: string;
  onValueChange: (value: string) => void;
  placeholder?: string;
  className?: string;
}) {
  return (
    <span
      className={`relative flex h-(--control-height) items-center rounded-capsule bg-fill text-ink-3 ring-1 ring-hairline ring-inset ${className}`}
    >
      <Icon icon={Search01Icon} className="pointer-events-none absolute left-3 size-4" />
      <Input
        type="search"
        aria-label={label}
        placeholder={placeholder}
        value={value}
        onValueChange={(next) => {
          onValueChange(next);
        }}
        className="h-full w-full min-w-0 rounded-capsule bg-transparent pr-3.5 pl-9 text-body text-ink-1 placeholder:text-ink-3 in-data-[density=touch]:text-input-touch"
      />
    </span>
  );
}
