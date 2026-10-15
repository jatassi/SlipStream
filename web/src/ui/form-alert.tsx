import { Alert02Icon } from "@hugeicons/core-free-icons";

import { useFormContext } from "./form-context";
import { Icon } from "./icon";

export function FormAlert() {
  const form = useFormContext();
  return (
    <form.Subscribe selector={(state): unknown => state.errorMap.onSubmit}>
      {(error) =>
        typeof error === "string" ? (
          <p role="alert" className="flex items-center gap-1.5 text-fail-ink">
            <Icon icon={Alert02Icon} className="size-4 shrink-0" />
            {error}
          </p>
        ) : null
      }
    </form.Subscribe>
  );
}
