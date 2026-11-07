import type { WorksContentType } from "../../api/gen/model";
import { Details } from "../../ui/details";
import { Dialog } from "../../ui/dialog";
import { placedOf } from "../fields-by-tier";
import { KeyValues, Ratings } from "../page/about";
import { aboutText } from "../page/fields-format";
import type { ItemView } from "./model";

/** An Item's own facts: its still, overview, what its type places in About, its ratings and its details. */
export function ItemDetails({
  type,
  view,
  onClose,
}: {
  type: WorksContentType;
  view: ItemView | undefined;
  onClose: () => void;
}) {
  return (
    <Dialog
      title={view === undefined ? "" : `${view.code} ${view.item.label}`}
      size={680}
      open={view !== undefined}
      onOpenChange={(open) => {
        if (!open) {
          onClose();
        }
      }}
    >
      {view !== undefined && <ItemFacts type={type} view={view} />}
    </Dialog>
  );
}

function ItemFacts({ type, view: { item } }: { type: WorksContentType; view: ItemView }) {
  const at = (place: Parameters<typeof placedOf>[3]) => placedOf(type, "item", item.fields, place);
  const stills = new Set(type.artwork.filter(({ tier }) => tier === "item").map(({ key }) => key));
  const still = item.artwork.find(({ kind }) => stills.has(kind));
  const overview = at("overview").at(0);
  const details = at("details");
  const record = `${type.source} lists this ${type.item.one.toLowerCase()} as ${item.key}.`;
  return (
    <div className="flex flex-col gap-4">
      {still !== undefined && (
        <img
          src={`${still.url}?size=medium`}
          alt=""
          className="aspect-video w-full rounded-item object-cover"
        />
      )}
      {overview !== undefined && <p className="text-ink-2">{aboutText(overview.value)}</p>}
      <KeyValues holder={item} items={at("about")} />
      {item.ratings.length > 0 && <Ratings ratings={item.ratings} />}
      {details.length === 0 ? (
        <p className="text-ink-2">{record}</p>
      ) : (
        <Details summary={record}>
          <KeyValues holder={item} items={details} />
        </Details>
      )}
    </div>
  );
}
