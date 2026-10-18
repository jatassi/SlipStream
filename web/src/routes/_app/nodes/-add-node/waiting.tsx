import { Spinner } from "../../../../ui/spinner";

export function Waiting({ text }: { text: string }) {
  return (
    <output className="flex items-center gap-2 text-ink-2">
      <Spinner />
      {text}
    </output>
  );
}
