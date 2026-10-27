import { Link } from "@tanstack/react-router";

import { isProblem, sentenceOf } from "../api/errors";
import type { WorksContentType } from "../api/gen/model";
import { useSetContentTypeEnabled } from "../api/gen/query/works/works";
import { AlertLine } from "../ui/alert-line";
import { Switch } from "../ui/switch";

function Refusal({ type, error }: { type: WorksContentType; error: unknown }) {
  if (!isProblem(error, "content_type_in_use")) {
    return <AlertLine text={sentenceOf(error)} />;
  }
  return (
    <p role="alert" className="text-fail-ink">
      The Library still holds {type.work.many}. Delete them in the{" "}
      <Link
        to="/library/$contentType"
        params={{ contentType: type.id }}
        className="text-ink-1 underline"
      >
        {type.name} Library
      </Link>
      , then turn {type.name} off.
    </p>
  );
}

function TypeSwitch({ type }: { type: WorksContentType }) {
  const toggle = useSetContentTypeEnabled();
  const checked = toggle.isIdle || toggle.isError ? type.enabled : toggle.variables.data.enabled;
  return (
    <div className="flex flex-col gap-2">
      <Switch
        label={type.name}
        checked={checked}
        onCheckedChange={(enabled) => {
          toggle.mutate({ contentType: type.id, data: { enabled } });
        }}
      />
      {toggle.isError && <Refusal type={type} error={toggle.error} />}
    </div>
  );
}

/**
 * A Content Type's on and off switch. Turning off a type that holds Works is
 * refused with a link to its Library.
 */
export function ContentTypeSwitch({ type }: { type: WorksContentType }) {
  // The live list changing type.enabled starts the switch afresh, so it never
  // shows a stale choice.
  return <TypeSwitch key={String(type.enabled)} type={type} />;
}
