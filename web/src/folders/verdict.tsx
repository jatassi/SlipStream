import { Alert02Icon, Tick02Icon } from "@hugeicons/core-free-icons";
import { useQuery } from "@tanstack/react-query";

import { sentenceOf } from "../api/errors";
import { Icon } from "../ui/icon";
import { Spinner } from "../ui/spinner";

/** A purpose's judgment of one folder, in a sentence the browser shows as it is. */
export type FolderVerdict = { readonly usable: boolean; readonly sentence: string };

/** A purpose's rules for a folder on a Node, such as a library root's. */
export type FolderCheck<V extends FolderVerdict = FolderVerdict> = {
  /** What the verdict turns on besides the Node and path: the purpose, and the root a re-home moves. */
  readonly rules: readonly string[];
  readonly judge: (args: {
    readonly nodeId: string;
    readonly path: string;
    readonly signal: AbortSignal;
  }) => Promise<V>;
};

export type Judging =
  | { readonly kind: "idle" }
  | { readonly kind: "checking" }
  | { readonly kind: "judged"; readonly verdict: FolderVerdict };

/** One folder's verdict, cached once for every reader of the same rules, Node and path. */
export const verdictKey = (check: FolderCheck, nodeId: string, path: string) =>
  ["folder-verdict", check.rules, nodeId, path] as const;

/** The check's verdict on a folder; "" asks nothing. */
export function useVerdict(check: FolderCheck, nodeId: string, path: string): Judging {
  const query = useQuery({
    queryKey: verdictKey(check, nodeId, path),
    queryFn: ({ signal }) => check.judge({ nodeId, path, signal }),
    enabled: path !== "",
    staleTime: 0,
  });
  if (path === "") {
    return { kind: "idle" };
  }
  if (query.isError) {
    return { kind: "judged", verdict: { usable: false, sentence: sentenceOf(query.error) } };
  }
  return query.data === undefined ? { kind: "checking" } : { kind: "judged", verdict: query.data };
}

export const usable = (judging: Judging): boolean =>
  judging.kind === "judged" && judging.verdict.usable;

export function VerdictLine({
  id,
  judging,
  name,
}: {
  id?: string;
  judging: Judging;
  name: string;
}) {
  return (
    <output
      id={id}
      className="flex min-h-5 items-start gap-1.5 text-meta in-data-[density=touch]:text-body-touch"
    >
      {judging.kind === "checking" && (
        <>
          <Spinner className="mt-0.5 text-ink-3" />
          <span className="text-ink-3">Checking {name}…</span>
        </>
      )}
      {judging.kind === "judged" &&
        (judging.verdict.usable ? (
          <>
            <Icon icon={Tick02Icon} className="size-3.5 shrink-0 translate-y-0.5 text-ok" />
            <span className="text-ink-2">{judging.verdict.sentence}</span>
          </>
        ) : (
          <>
            <Icon icon={Alert02Icon} className="size-3.5 shrink-0 translate-y-0.5 text-fail-ink" />
            <span className="text-fail-ink">{judging.verdict.sentence}</span>
          </>
        ))}
    </output>
  );
}
