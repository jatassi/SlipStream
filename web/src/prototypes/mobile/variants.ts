import type { ComponentType } from 'react'

import { NativeVariant } from './variants/native'

export type VariantEntry = {
  name: string
  axis: string
  component: ComponentType
}

export const VARIANTS: VariantEntry[] = [
  { name: 'Native', axis: 'Platform familiarity — tab bar, large titles, push navigation', component: NativeVariant },
]
