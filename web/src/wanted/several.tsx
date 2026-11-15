import { useNavigate } from "@tanstack/react-router";
import { type ReactNode, useState } from "react";

import { sentenceOf } from "../api/errors";
import type { GrabsSeveralBody, GrabsSeveralQueued } from "../api/gen/model";
import { searchSeveralWorks } from "../api/gen/query/grabs/grabs";
import { ConfirmDialog } from "../ui/confirm-dialog";
import { toast } from "../ui/toast";
import { worksWords } from "./wanted-words";

/** Above this many Works a search of several asks first, because it can take days to drain. */
const askAbove = 100;

/** What a search of several Works did, as its toast. */
export function severalWords({ queued, leftGone }: GrabsSeveralQueued): string {
  if (queued === 0) {
    return "Nothing to search. Every listed Work is already queued or has only gone files.";
  }
  const done = `Queued ${worksWords(queued)} for search.`;
  return leftGone === 0
    ? done
    : `${done} ${worksWords(leftGone)} with gone files weren't queued. Search them from their Work pages.`;
}

/** Searches several Works, toasting what it queued; a toast that leaves out gone files offers Wanted filtered to them. */
function useSeveralRun(contentType: string | undefined) {
  const navigate = useNavigate();
  const [busy, setBusy] = useState(false);
  const showGone = () => {
    const search = { reason: "gone" };
    void (contentType === undefined
      ? navigate({ to: "/library/wanted", search })
      : navigate({ to: "/library/wanted/$contentType", params: { contentType }, search }));
  };
  const run = async (body: GrabsSeveralBody) => {
    setBusy(true);
    try {
      const answer = await searchSeveralWorks(body);
      toast(
        severalWords(answer),
        answer.queued > 0 && answer.leftGone > 0
          ? { action: { label: "Show", onClick: showGone } }
          : {},
      );
    } finally {
      setBusy(false);
    }
  };
  return { busy, run };
}

/** Searches several Works in the bulk lane, first asking when there are more than 100. */
export function useSearchSeveral(contentType: string | undefined): {
  busy: boolean;
  ask: (body: GrabsSeveralBody, count: number) => void;
  dialog: ReactNode;
} {
  const { busy, run } = useSeveralRun(contentType);
  const [asking, setAsking] = useState<{ body: GrabsSeveralBody; count: number }>();
  const ask = (body: GrabsSeveralBody, count: number) => {
    if (count > askAbove) {
      setAsking({ body, count });
      return;
    }
    run(body).catch((error: unknown) => {
      toast(sentenceOf(error), { kind: "fail" });
    });
  };
  const dialog = (
    <ConfirmDialog
      title={`Search ${worksWords(asking?.count ?? 0)}?`}
      confirmLabel="Search"
      open={asking !== undefined}
      onOpenChange={(open) => {
        if (!open) {
          setAsking(undefined);
        }
      }}
      onConfirm={() => (asking === undefined ? undefined : run(asking.body))}
    >
      <p>They wait for each Indexer&apos;s limit and can take several days.</p>
    </ConfirmDialog>
  );
  return { busy, ask, dialog };
}
