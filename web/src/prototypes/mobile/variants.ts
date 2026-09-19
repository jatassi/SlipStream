import type { ComponentType } from 'react'

import { CinematicVariant } from './variants/cinematic'
import { ConsoleVariant } from './variants/console'
import { NativeVariant } from './variants/native'

export type VariantEntry = {
  name: string
  axis: string
  component: ComponentType
}

export const VARIANTS: VariantEntry[] = [
  { name: 'Native', axis: 'Platform familiarity — tab bar, large titles, push navigation', component: NativeVariant },
  { name: 'Cinematic', axis: 'Immersion — edge-to-edge art, glass dock, gesture sheets', component: CinematicVariant },
  { name: 'Console', axis: 'Density & speed — command bar, dense rows, zero ornamental motion', component: ConsoleVariant },
]
