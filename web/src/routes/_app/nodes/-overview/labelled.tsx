import { Button } from "../../../../ui/button";

/** A fact with the one action that changes it, such as a name and Rename. */
export function Labelled({
  text,
  action,
}: {
  text: string;
  action: { label: string; onClick: () => void };
}) {
  return (
    <span className="flex flex-wrap items-center gap-x-3 gap-y-1">
      <span>{text}</span>
      <Button size="small" look="ghost" onClick={action.onClick}>
        {action.label}
      </Button>
    </span>
  );
}
