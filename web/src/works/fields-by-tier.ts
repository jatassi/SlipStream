import type {
  WorksContentType,
  WorksFieldSpec,
  WorksFieldValue,
  WorksItem,
} from "../api/gen/model";

export type Placed = {
  readonly spec: WorksFieldSpec;
  readonly value: WorksFieldValue;
};

type Tier = WorksFieldSpec["tier"];
type Place = WorksFieldSpec["in"][number];

/** A Work's, level's or Item's values of the fields its type shows in a place at that tier, in declared order. */
export function placedOf(
  type: WorksContentType,
  tier: Tier,
  fields: readonly WorksFieldValue[],
  place: Place,
): readonly Placed[] {
  return type.fields
    .filter((spec) => spec.tier === tier && spec.in.includes(place))
    .flatMap((spec) => {
      const value = fields.find((field) => field.key === spec.key);
      return value === undefined ? [] : [{ spec, value }];
    });
}

/** The short identifier an Item row leads with, such as S01E03, or its position when its type names none. */
export function codeOf(type: WorksContentType, item: WorksItem): string {
  const value = placedOf(type, "item", item.fields, "code").at(0)?.value;
  return value?.kind === "text" ? value.text : String(item.position);
}
