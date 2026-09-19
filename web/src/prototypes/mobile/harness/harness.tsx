import { VARIANTS } from '../variants'
import { PhoneFrame } from './phone-frame'
import { Picker } from './picker'
import { usePicker } from './use-picker'

const NAMES = VARIANTS.map((v) => v.name)

export function Harness() {
  const { active, mountKey, select, replay } = usePicker(VARIANTS.length)
  const Variant = VARIANTS[active].component

  return (
    <>
      <Picker names={NAMES} active={active} onSelect={select} onReplay={replay} />
      <PhoneFrame>
        <div key={mountKey} className="m-root h-full">
          <Variant />
        </div>
      </PhoneFrame>
    </>
  )
}
