import type { TrackersRoute, TrackersTarget } from "../api/gen/model";
import { type ClientPick, builtInKey } from "../grabs/send-to-sheet";
import { Button } from "../ui/button";
import { Section } from "../ui/section";
import { Select } from "../ui/select";
import { Refusal } from "../ui/setting-parts";
import { type ClientNames, routeWords } from "./route-words";
import { useRouteSave } from "./route-save";

const keyOf = ({ nodeId, clientId }: TrackersTarget) => clientId ?? builtInKey(nodeId);

const byRootKey = "by-root";

const metaLook = "text-meta text-ink-3 in-data-[density=touch]:text-body-touch";

/** Every client that can take the Grabs, the home's built-in one marked as the default. */
function RoutePicker({
  route,
  picks,
  busy,
  onPick,
}: {
  route: TrackersRoute;
  picks: readonly ClientPick[];
  busy: boolean;
  onPick: (pick: ClientPick) => void;
}) {
  const fallback = route.default === null ? undefined : keyOf(route.default);
  return (
    <Select
      label="Grabs go to"
      options={[
        ...(route.to === null
          ? [{ value: byRootKey, label: "Where the Work's library root is" }]
          : []),
        ...picks.map(({ key, label }) => ({
          value: key,
          label: key === fallback ? `${label} (default)` : label,
        })),
      ]}
      value={route.to === null ? byRootKey : keyOf(route.to)}
      disabled={busy}
      onValueChange={(key) => {
        const picked = picks.find((pick) => pick.key === key);
        if (picked !== undefined) {
          onPick(picked);
        }
      }}
    />
  );
}

/**
 * One home Node's route: where the Grabs its Indexers find go now, the
 * picker, and Use the default while the Admin picked one.
 */
export function RouteRow({
  tracker,
  route,
  picks,
  names,
  announces,
}: {
  tracker: { readonly id: string; readonly name: string };
  route: TrackersRoute;
  picks: readonly ClientPick[];
  names: ClientNames;
  announces: string | undefined;
}) {
  const home = names.node(route.homeNodeId);
  const save = useRouteSave(tracker.id, route.homeNodeId);
  const from = `${tracker.name}'s Grabs from Indexers on ${home}`;
  return (
    <Section title={`Indexers on ${home}`}>
      <div className="flex max-w-120 flex-col gap-2">
        <RoutePicker
          route={route}
          picks={picks}
          busy={save.busy}
          onPick={(picked) => void save.pick(picked.body, `${from} go to ${picked.name}`)}
        />
        <p className={metaLook}>{routeWords(names, route)}</p>
        {announces !== undefined && <p className={metaLook}>{announces}</p>}
        {save.refusal !== undefined && <Refusal error={save.refusal} />}
        {route.stored !== null && (
          <div>
            <Button
              look="ghost"
              busy={save.busy}
              onClick={() => void save.toDefault(`${from} go to the default`)}
            >
              Use the default
            </Button>
          </div>
        )}
      </div>
    </Section>
  );
}
