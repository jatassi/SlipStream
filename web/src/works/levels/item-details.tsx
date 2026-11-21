import type { WorksContentType } from "../../api/gen/model";
import { Artwork } from "../../ui/artwork";
import { Details } from "../../ui/details";
import { Dialog } from "../../ui/dialog";
import { placedOf } from "../fields-by-tier";
import { FieldFacts, Ratings } from "../page/about";
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

function ItemFacts({ type, view }: { type: WorksContentType; view: ItemView }) {
  const { item } = view;
  const at = (place: Parameters<typeof placedOf>[3]) => placedOf(type, "item", item.fields, place);
  const stills = new Set(type.artwork.filter(({ tier }) => tier === "item").map(({ key }) => key));
  const still = item.artwork.find(({ kind }) => stills.has(kind));
  const overview = at("overview").at(0);
  const details = at("details");
  const record = `${type.source} ID: ${item.key}`;
  return (
    <div className="flex flex-col gap-4">
      {still !== undefined && (
        <Artwork use="still" size="wide" url={still.url} label={`${view.code} ${item.label}`} />
      )}
      {overview !== undefined && <p className="text-ink-2">{aboutText(overview.value)}</p>}
      <FieldFacts holder={item} items={at("about")} look="inline" />
      {item.ratings.length > 0 && <Ratings ratings={item.ratings} />}
      {details.length === 0 ? (
        <p className="text-ink-2">{record}</p>
      ) : (
        <Details summary={record}>
          <FieldFacts holder={item} items={details} look="well" />
        </Details>
      )}
    </div>
  );
}
