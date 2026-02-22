import { useState } from 'react'

import { useBrowseDirectory } from '@/hooks'

const getBreadcrumbs = (path: string) => {
  if (!path) {
    return []
  }

  const isWindows = /^[A-Za-z]:/.test(path)
  const parts = path.split(/[/\\]/).filter(Boolean)

  const breadcrumbs: { label: string; path: string }[] = []
  let accumulated = isWindows ? '' : '/'

  for (const part of parts) {
    if (isWindows) {
      accumulated = accumulated ? `${accumulated}\\${part}` : part
    } else {
      accumulated = `${accumulated}${accumulated === '/' ? '' : '/'}${part}`
    }

    // For Windows, use drive root path (part is already "C:" from split)
    const displayPath = isWindows && breadcrumbs.length === 0 ? `${part}\\` : accumulated

    breadcrumbs.push({ label: part, path: displayPath })
  }

  return breadcrumbs
}

type FolderBrowserOptions = {
  initialPath: string
  open: boolean
  onSelect: (path: string) => void
  onOpenChange: (open: boolean) => void
  fileExtensions?: string[]
}

export function useFolderBrowser(opts: FolderBrowserOptions) {
  const [currentPath, setCurrentPath] = useState(opts.initialPath)
  const [inputPath, setInputPath] = useState(opts.initialPath)
  const [selectedFile, setSelectedFile] = useState('')
  const query = useBrowseDirectory(currentPath || undefined, opts.open, opts.fileExtensions)
  const showFiles = !!opts.fileExtensions?.length

  const handleNavigate = (path: string) => {
    setCurrentPath(path)
    setInputPath(path)
    setSelectedFile('')
  }

  const handleFileSelect = (path: string) => {
    setSelectedFile(path)
    setInputPath(path)
  }

  const handleSelect = () => {
    opts.onSelect(selectedFile || currentPath || inputPath)
    opts.onOpenChange(false)
  }

  return {
    inputPath,
    setInputPath,
    selectedFile,
    ...query,
    breadcrumbs: getBreadcrumbs(query.data?.path ?? currentPath),
    selectedPath: selectedFile || currentPath || inputPath,
    showFiles,
    handleNavigate,
    handleFileSelect: showFiles ? handleFileSelect : undefined,
    handleInputSubmit: (e: React.SyntheticEvent) => {
      e.preventDefault()
      setCurrentPath(inputPath)
      setSelectedFile('')
    },
    handleSelect,
  }
}
