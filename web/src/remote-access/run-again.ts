import { sentenceOf } from "../api/errors";
import type { ReachabilityExposureView } from "../api/gen/model";
import { runRemoteAccess } from "../api/gen/query/reachability/reachability";
import { toast } from "../ui/toast";

/** Checks one route again now, saying so in a toast. */
export async function runAgain(exposure: ReachabilityExposureView) {
  try {
    await runRemoteAccess({ exposure: exposure.id });
    toast(`Checking ${exposure.subject} again`);
  } catch (error) {
    toast(sentenceOf(error), { kind: "fail" });
  }
}
