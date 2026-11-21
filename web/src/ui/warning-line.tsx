import type { AlertTone } from "./count";
import { Callout, type CalloutAction } from "./callout";

type WarningLineProps = {
  signal: AlertTone;
  chip: string;
  subject: string;
  detail: string;
  since?: string;
  action: CalloutAction;
};

export const WarningLine = ({ signal, detail, ...rest }: WarningLineProps) => (
  <Callout tone={signal} {...rest}>
    {detail}
  </Callout>
);
