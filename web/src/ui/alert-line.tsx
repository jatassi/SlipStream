import { SignalLine } from "./signal-line";

export const AlertLine = ({ text }: { text: string }) => (
  <SignalLine tone="fail" announce>
    {text}
  </SignalLine>
);
