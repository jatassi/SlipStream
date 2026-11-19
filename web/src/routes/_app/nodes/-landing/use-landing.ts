import {
  useGetRoleBoardSuspense,
  useGetTopologySuspense,
  useListNodesSuspense,
} from "../../../../api/gen/query/nodes/nodes";
import { useRestarting } from "../../../../nodes/state";
import { useNow } from "../../../../tasks/use-now";
import { type Landing, landingOf } from "./view";

/** Topology read as the landing shows it, kept current to the minute. */
export function useLanding(): Landing {
  const topology = useGetTopologySuspense().data;
  const board = useGetRoleBoardSuspense().data;
  const restarting = useRestarting(useListNodesSuspense().data.nodes);
  const now = useNow(60_000);
  return landingOf(topology, board, restarting, { now, zone: undefined });
}
