import { useState } from 'react'

import { Plus, X } from 'lucide-react'

import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'

function ExtensionBadge({ ext, onRemove }: { ext: string; onRemove: () => void }) {
  return (
    <Badge variant="secondary" className="gap-1">
      {ext}
      <button type="button" onClick={onRemove} className="hover:text-destructive ml-1">
        <X className="size-3" />
      </button>
    </Badge>
  )
}

export function ExtensionManager({ extensions, onChange }: { extensions: string[]; onChange: (extensions: string[]) => void }) {
  const [newExt, setNewExt] = useState('')
  const addExtension = () => {
    if (!newExt) {
      return
    }
    let ext = newExt.trim().toLowerCase()
    if (!ext.startsWith('.')) {
      ext = `.${ext}`
    }
    if (!extensions.includes(ext)) {
      onChange([...extensions, ext])
    }
    setNewExt('')
  }
  const handleKeyDown = (e: React.KeyboardEvent) => {
    if (e.key === 'Enter') {
      e.preventDefault()
      addExtension()
    }
  }
  return (
    <div className="space-y-3">
      <div className="flex flex-wrap gap-2">
        {extensions.map((ext) => (
          <ExtensionBadge key={ext} ext={ext} onRemove={() => onChange(extensions.filter((e) => e !== ext))} />
        ))}
      </div>
      <div className="flex gap-2">
        <Input
          value={newExt}
          onChange={(e) => setNewExt(e.target.value)}
          placeholder=".mkv"
          aria-label="New video extension"
          className="h-11 w-24 text-base"
          onKeyDown={handleKeyDown}
        />
        <Button type="button" variant="outline" className="h-11" onClick={addExtension}>
          <Plus className="mr-1 size-4" />
          Add
        </Button>
      </div>
    </div>
  )
}
