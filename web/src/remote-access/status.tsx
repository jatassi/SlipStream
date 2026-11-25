import type { ReachabilityRemoteAccess, ReachabilityStatusPart } from "../api/gen/model";
import { type StatusPart, StatusSentence } from "../ui/status-sentence";
import { LastRun } from "./last-run";
import { recommendationAnchor } from "./recommendation";
import { routeAnchor } from "./route";

export const routesAnchor = "routes";

/** The status sentence, its warning and failure parts linking to the card or their route. */
function statusParts(page: ReachabilityRemoteAccess): StatusPart[] {
  const hrefOf = ({ exposureId }: ReachabilityStatusPart) =>
    exposureId === undefined
      ? `#${routesAnchor}`
      : exposureId === page.card?.exposureId
        ? `#${recommendationAnchor}`
        : `#${routeAnchor(exposureId)}`;
  return page.status.parts.map((part): StatusPart => {
    if (part.tone === "fail") {
      return { kind: "failure", text: part.text, href: hrefOf(part) };
    }
    if (part.tone === "warn") {
      return { kind: "warning", text: part.text, href: hrefOf(part) };
    }
    return { kind: "plain", text: part.text };
  });
}

/** The status sentence, then when the routes were last checked and from where, with Run again. */
export function StatusHead({ page }: { page: ReachabilityRemoteAccess }) {
  return (
    <div className="mb-4 flex flex-col gap-3">
      <StatusSentence parts={statusParts(page)} />
      <LastRun run={page.lastRun} />
    </div>
  );
}
