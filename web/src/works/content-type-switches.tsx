import { useQueryClient } from "@tanstack/react-query";
import { useState } from "react";

import { isProblem, sentenceOf } from "../api/errors";
import type { WorksContentType } from "../api/gen/model";
import { setContentTypeEnabled, useListContentTypesSuspense } from "../api/gen/query/works/works";
import { adoptContentType } from "../live/adopt";
import { AlertLine } from "../ui/alert-line";
import { Switch } from "../ui/switch";

function TypeSwitch({ type, last }: { type: WorksContentType; last: boolean }) {
  const queryClient = useQueryClient();
  const [busy, setBusy] = useState(false);
  const [refusal, setRefusal] = useState<string>();
  const turn = async (enabled: boolean) => {
    setBusy(true);
    setRefusal(undefined);
    try {
      adoptContentType(queryClient, await setContentTypeEnabled(type.id, { enabled }));
    } catch (error) {
      setRefusal(
        isProblem(error, "content_type_last")
          ? "At least one kind of media stays on. Turn on another before turning this one off."
          : sentenceOf(error),
      );
    } finally {
      setBusy(false);
    }
  };
  return (
    <li className="flex flex-col gap-1">
      <Switch
        label={type.name}
        checked={type.enabled}
        disabled={busy || (type.enabled && last)}
        onCheckedChange={(enabled) => void turn(enabled)}
      />
      {refusal !== undefined && <AlertLine text={refusal} />}
    </li>
  );
}

/** A switch per Content Type. The last one on stays on, so SlipStream always manages something. */
export function ContentTypeSwitches() {
  const { contentTypes } = useListContentTypesSuspense().data;
  const on = contentTypes.filter((type) => type.enabled).length;
  return (
    <div className="flex flex-col gap-2">
      <ul aria-label="Kinds of media" className="flex flex-col gap-1">
        {contentTypes.map((type) => (
          <TypeSwitch key={type.id} type={type} last={on === 1} />
        ))}
      </ul>
      {on === 1 && (
        <p className="text-meta text-ink-3 in-data-[density=touch]:text-body-touch">
          At least one kind of media stays on.
        </p>
      )}
    </div>
  );
}
