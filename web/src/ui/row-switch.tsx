import { mixedAttributes, type SwitchChecked, switchKnob, switchTrack } from "./switch";

/**
 * The switch for rows a page draws by the hundred. It looks like
 * SwitchControl and is a native button, because a Base UI control in every
 * row is what made long lists slow to scroll.
 */
export function RowSwitch({
  label,
  checked,
  onCheckedChange,
  disabled = false,
}: {
  label: string;
  checked: SwitchChecked;
  onCheckedChange: (checked: boolean) => void;
  disabled?: boolean;
}) {
  const on = checked === true ? { "data-checked": "" } : {};
  const mixed = mixedAttributes(checked);
  return (
    <button
      type="button"
      role="switch"
      aria-label={label}
      aria-checked={checked === true}
      disabled={disabled}
      onClick={() => {
        onCheckedChange(checked !== true);
      }}
      {...on}
      {...mixed}
      className={`${switchTrack} disabled:opacity-40`}
    >
      <span {...on} {...mixed} className={switchKnob} />
    </button>
  );
}
