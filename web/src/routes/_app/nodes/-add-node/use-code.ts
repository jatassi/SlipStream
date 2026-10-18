import { useQuery } from "@tanstack/react-query";

import { isApiError, sentenceOf } from "../../../../api/errors";
import type { NodesCodeState, NodesJoinCode } from "../../../../api/gen/model";
import {
  getGetJoinCodeQueryKey,
  getJoinCode,
  getListJoinCodesQueryKey,
  listJoinCodes,
  useListNodesSuspense,
} from "../../../../api/gen/query/nodes/nodes";
import { useNow } from "../../../../tasks/use-now";
import { type AddNodeState, type CodeAnswer, addNodeState } from "./state";

/**
 * How often the dialog asks after a pasted Node address. The Controller
 * works out "can't reach" when asked, so no event says it.
 */
export const dialPoll = 2000;

const dialingOf = (state: NodesCodeState | undefined): boolean =>
  state?.dialing !== undefined && !state.dialing.unreachable;

// A failed refetch keeps the last answer, so a 404 after a revoke wins over it.
function answerOf({
  data,
  error,
}: {
  data: NodesCodeState | undefined;
  error: unknown;
}): CodeAnswer {
  if (isApiError(error) && error.code === "join_code_not_found") {
    return { kind: "gone" };
  }
  if (data !== undefined) {
    return { kind: "known", state: data };
  }
  return error === null ? { kind: "loading" } : { kind: "failed", sentence: sentenceOf(error) };
}

/** The Join Code the dialog follows, where it stands, and the codes still open beside it. */
export function useCode(id: string, minted: NodesJoinCode | undefined) {
  const now = useNow(1000);
  const { nodes } = useListNodesSuspense().data;
  const code = useQuery({
    queryKey: getGetJoinCodeQueryKey(id),
    queryFn: ({ signal }) => getJoinCode(id, { signal }),
    retry: false,
    refetchInterval: (query) => (dialingOf(query.state.data) ? dialPoll : false),
  });
  const open = useQuery({
    queryKey: getListJoinCodesQueryKey(),
    queryFn: ({ signal }) => listJoinCodes({ signal }),
  });
  const codes = (open.data?.codes ?? []).filter(
    ({ expiresAt }) => Date.parse(expiresAt) > now.getTime(),
  );
  const expiresAt =
    minted?.id === id
      ? minted.expiresAt
      : open.data?.codes.find((candidate) => candidate.id === id)?.expiresAt;
  const state: AddNodeState = addNodeState({
    id,
    answer: answerOf(code),
    minted,
    expiresAt,
    nodes,
    now,
  });
  return { state, codes, nodes, now };
}
