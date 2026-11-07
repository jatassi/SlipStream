import { useState } from "react";

import type { WorksContentType } from "../../api/gen/model";
import { AttemptActions, useAttempt } from "../../targets/attempt";
import { Dialog } from "../../ui/dialog";
import { act } from "../actions/act";
import type { Subject } from "../actions/actions";
import { namedOf } from "../actions/outcomes";
import type { MonitoringChoice } from "./choice";
import { MonitoringFields } from "./monitoring-fields";
import { useMonitoringPreview } from "./use-monitoring";

type FormProps = {
  readonly type: WorksContentType;
  readonly subjects: readonly Subject[];
  /** Where Include starts: the Work's own rule, or off for many Works. */
  readonly includeAsides: boolean;
  readonly onDone: () => void;
};

function MonitoringForm({ type, subjects, includeAsides, onDone }: FormProps) {
  const [choice, setChoice] = useState<MonitoringChoice>({
    preset: "all",
    includeAsides,
    monitorNew: undefined,
  });
  const preview = useMonitoringPreview(
    subjects.map(({ id }) => id),
    choice,
  );
  const applying = useAttempt(async () => {
    await act(subjects, { kind: "monitoring", ...choice });
    onDone();
  });
  return (
    <div className="flex flex-col gap-4">
      <MonitoringFields
        type={type}
        works={subjects.length}
        preview={preview}
        choice={choice}
        onChange={setChoice}
      />
      <AttemptActions attempt={applying} onCancel={onDone}>
        Apply
      </AttemptActions>
    </div>
  );
}

/**
 * A preset for one Work or many of one many-Items type, each preset with
 * what it monitors and the line saying how many Items start and stop
 * being monitored, before Apply writes it.
 */
export function MonitoringDialog({
  open,
  onOpenChange,
  ...form
}: Omit<FormProps, "onDone"> & { open: boolean; onOpenChange: (open: boolean) => void }) {
  return (
    <Dialog
      title={`Set monitoring for ${namedOf(form.subjects).words}`}
      open={open}
      onOpenChange={onOpenChange}
    >
      <MonitoringForm
        {...form}
        onDone={() => {
          onOpenChange(false);
        }}
      />
    </Dialog>
  );
}
