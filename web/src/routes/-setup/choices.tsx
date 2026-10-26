import type { ComponentType } from "react";

import type { SystemStatus } from "../../api/gen/model";
import { Button } from "../../ui/button";
import type { ChoiceId, FormProps } from "./form-props";
import { JoinForm } from "./join-form";
import { StartForm } from "./start-form";

export type Choice = {
  readonly id: ChoiceId;
  readonly label: string;
  readonly about: string;
  /** Exactly one row is primary for any status. A removed Node leads with Join. */
  readonly primary: (status: SystemStatus) => boolean;
  readonly Form: ComponentType<FormProps>;
};

/** What this Node can become. S49 adds "restore". */
export const choices: { readonly [C in ChoiceId]: Choice & { readonly id: C } } = {
  start: {
    id: "start",
    label: "Start a new SlipStream",
    about: "This Node becomes the Controller, and you create the first Admin.",
    primary: (status) => status.removed === undefined,
    Form: StartForm,
  },
  join: {
    id: "join",
    label: "Join an existing SlipStream",
    about:
      "This Node becomes a Worker of a SlipStream you already run, with a Join Code from Add a Node.",
    primary: (status) => status.removed !== undefined,
    Form: JoinForm,
  },
};

export function ChoiceList({
  status,
  choose,
}: {
  status: SystemStatus;
  choose: (id: ChoiceId) => void;
}) {
  const ordered = Object.values(choices).toSorted(
    (a, b) => Number(b.primary(status)) - Number(a.primary(status)),
  );
  return (
    <ul aria-label="What this Node becomes" className="flex flex-col gap-4">
      {ordered.map((choice) => (
        <li key={choice.id} className="flex flex-col items-start gap-2">
          <Button
            look={choice.primary(status) ? "primary" : "default"}
            onClick={() => {
              choose(choice.id);
            }}
          >
            {choice.label}
          </Button>
          <p className="text-meta text-ink-3">{choice.about}</p>
        </li>
      ))}
    </ul>
  );
}
