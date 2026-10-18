import { Add01Icon, Cancel01Icon } from "@hugeicons/core-free-icons";

import { Button } from "../ui/button";
import { IconButton } from "../ui/icon-button";
import { Menu } from "../ui/menu";

export function PickMenu({
  label,
  options,
  onPick,
}: {
  label: string;
  options: readonly { readonly key: string; readonly name: string }[];
  onPick: (key: string) => void;
}) {
  const button = (
    <Button size="small" icon={Add01Icon} disabled={options.length === 0}>
      {label}
    </Button>
  );
  if (options.length === 0) {
    return button;
  }
  return (
    <Menu
      label={label}
      trigger={button}
      items={options.map((option) => ({
        label: option.name,
        onSelect: () => {
          onPick(option.key);
        },
      }))}
    />
  );
}

export function RemoveButton({
  name,
  disabled = false,
  onRemove,
}: {
  name: string;
  disabled?: boolean;
  onRemove: () => void;
}) {
  return (
    <IconButton
      icon={Cancel01Icon}
      label={`Remove ${name}`}
      disabled={disabled}
      onClick={onRemove}
    />
  );
}
