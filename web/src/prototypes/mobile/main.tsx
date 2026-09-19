import { StrictMode } from 'react'
import { createRoot } from 'react-dom/client'

import { Harness } from './harness/harness'

import './styles/prototype.css'

const rootElement = document.getElementById('root')
if (rootElement) {
  createRoot(rootElement).render(
    <StrictMode>
      <Harness />
    </StrictMode>,
  )
}
