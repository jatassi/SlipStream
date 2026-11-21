import type {
  TargetsProfile,
  TargetsQualityList,
  TargetsSize,
  TargetsTarget,
} from "../../api/gen/model";
import { draftOf, rulesOfDraft } from "../../targets/draft";
import { useQualityNames } from "../../targets/quality-names";
import { upgradeSentence } from "../../targets/upgrade-sentence";
import { Button } from "../../ui/button";
import { Details } from "../../ui/details";
import { Fact, Facts } from "../../ui/facts";

type Name = (key: string) => string;

const sizeWords = ({ min, max }: TargetsSize): string =>
  max === null
    ? `${String(min)} MiB per minute or more`
    : `${String(min)} to ${String(max)} MiB per minute`;

function Machinery({
  profile,
  target,
  name,
}: {
  profile: TargetsProfile;
  target: TargetsTarget;
  name: Name;
}) {
  const qualities = target.tiers.flatMap((tier) => tier ?? []);
  return (
    <div className="mt-2">
      <Facts look="well">
        <Fact label="Profile">{profile.name}</Fact>
        <Fact label="Qualities">{qualities.map((key) => name(key)).join(", ")}</Fact>
        <Fact label="Cutoff">{name(target.cutoff)}</Fact>
        <Fact label="Sizes">
          <ul>
            {target.sizes.map((size) => (
              <li key={size.quality}>
                {name(size.quality)}, {sizeWords(size)}
              </li>
            ))}
          </ul>
        </Fact>
      </Facts>
    </div>
  );
}

export function TargetLine({
  profile,
  target,
  qualities,
  named,
  onEdit,
}: {
  profile: TargetsProfile;
  target: TargetsTarget;
  qualities: TargetsQualityList;
  named: boolean;
  onEdit: () => void;
}) {
  const name = useQualityNames();
  const { sentence } = upgradeSentence(
    rulesOfDraft(draftOf(target, qualities, [])),
    qualities.qualities.map((quality) => quality.key),
    name,
  );
  return (
    <Details
      summary={named ? `${target.name}: ${sentence}` : sentence}
      action={
        <Button size="small" onClick={onEdit}>
          Edit {target.name}
        </Button>
      }
    >
      <Machinery profile={profile} target={target} name={name} />
    </Details>
  );
}
