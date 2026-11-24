import { useGetRemoteAccessSuspense } from "../api/gen/query/reachability/reachability";
import { useAddNode } from "../routes/_app/nodes/-add-node/add-node";
import { Page } from "../shell/page";
import { NothingWanted } from "./empty";
import { Wanted } from "./wanted";

/**
 * Remote access: what is set up for use away from home and whether players
 * outside can connect to it, one recommendation first and each route under
 * it. A pushed page of Nodes, with no nav row of its own.
 */
export function RemoteAccessPage() {
  const page = useGetRemoteAccessSuspense().data;
  const adding = useAddNode();
  return (
    <Page>
      {page.nothingWanted ? (
        <NothingWanted suggestions={page.suggestions} />
      ) : (
        <Wanted page={page} pair={adding.start} />
      )}
      {adding.dialog}
    </Page>
  );
}
