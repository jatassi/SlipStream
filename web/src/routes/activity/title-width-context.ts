import { createContext } from 'react'

type TitleWidthContextType = {
  registerWidth: (id: string, width: number) => void
  unregisterWidth: (id: string) => void
  maxWidth: number
}

export const TitleWidthContext = createContext<TitleWidthContextType>({
  registerWidth: () => undefined,
  unregisterWidth: () => undefined,
  maxWidth: 0,
})
