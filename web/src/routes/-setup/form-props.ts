import type { SetupCode } from "../../session/setup-code";

export type ChoiceId = "start" | "join";

export type FormProps = {
  readonly code: SetupCode;
  readonly back: () => void;
  /** Opens another choice's form, as Start does for a Join that is waiting. */
  readonly open: (id: ChoiceId) => void;
  /** The Node refused the code because it restarted since the check. */
  readonly restarted: () => void;
};
