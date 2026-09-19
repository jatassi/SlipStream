export type NavInput = 'keyboard' | 'pointer'

let input: NavInput = 'pointer'
let bound = false

function onKeyDown(event: KeyboardEvent) {
  if (
    event.metaKey ||
    event.altKey ||
    event.ctrlKey ||
    event.key === 'Tab' ||
    event.key === 'Enter' ||
    event.key === ' ' ||
    event.key === 'Escape' ||
    event.key.startsWith('Arrow')
  ) {
    input = 'keyboard'
  }
}

function onPointerDown() {
  input = 'pointer'
}

function bind() {
  if (bound) {
    return
  }
  bound = true
  globalThis.addEventListener('keydown', onKeyDown, true)
  globalThis.addEventListener('pointerdown', onPointerDown, true)
}

export function lastNavInput(): NavInput {
  bind()
  return input
}
