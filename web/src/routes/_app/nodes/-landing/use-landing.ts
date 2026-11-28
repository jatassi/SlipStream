import {
  useGetRoleBoardSuspense,
  useGetTopologySuspense,
  useListNodesSuspense,
} from "../../../../api/gen/query/nodes/nodes";
import { useRunningConversions } from "../../../../live/conversions";
import { useRestarting } from "../../../../nodes/state";
import { useClock } from "../../../../ui/clock";
import { type Landing, landingOf } from "./view";

/** Topology read as the landing shows it, kept current to the minute. */
export function useLanding(): Landing {
  const topology = useGetTopologySuspense().data;
  const board = useGetRoleBoardSuspense().data;
  const restarting = useRestarting(useListNodesSuspense().data.nodes);
  return landingOf(topology, board, restarting, useRunningConversions(), useClock());
}
