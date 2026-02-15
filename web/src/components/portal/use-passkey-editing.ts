import { useEffect, useRef, useState } from 'react'

import { useUpdatePasskeyName } from '@/hooks/portal'

export function usePasskeyEditing() {
  const [editingId, setEditingId] = useState<string | null>(null)
  const [editingName, setEditingName] = useState('')
  const editInputRef = useRef<HTMLInputElement>(null)

  const updateName = useUpdatePasskeyName()

  useEffect(() => {
    if (editingId !== null) {
      editInputRef.current?.focus()
    }
  }, [editingId])

  const handleStartEdit = (id: string, currentName: string) => {
    setEditingId(id)
    setEditingName(currentName)
  }

  const handleSaveEdit = async () => {
    if (!editingId || !editingName.trim()) {
      return
    }
    await updateName.mutateAsync({ id: editingId, name: editingName })
    setEditingId(null)
    setEditingName('')
  }

  const handleCancelEdit = () => {
    setEditingId(null)
    setEditingName('')
  }

  return {
    editingId,
    editingName,
    setEditingName,
    editInputRef,
    updatePending: updateName.isPending,
    handleStartEdit,
    handleSaveEdit,
    handleCancelEdit,
  }
}
