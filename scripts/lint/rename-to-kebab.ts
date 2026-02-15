#!/usr/bin/env bun
import { readdir, readFile, writeFile, rename } from "fs/promises"
import { basename, dirname, extname, join, relative } from "path"
import { execSync } from "child_process"

const WEB_DIR = join(import.meta.dir, "../../web")
const SRC_DIR = join(WEB_DIR, "src")

function toKebabCase(name: string): string {
  return name
    .replace(/([a-z0-9])([A-Z])/g, "$1-$2")
    .replace(/([A-Z])([A-Z][a-z])/g, "$1-$2")
    .toLowerCase()
}

function isKebabCase(name: string): boolean {
  if (name.startsWith("$")) return true
  return /^[a-z0-9]+(-[a-z0-9]+)*$/.test(name)
}

async function findAllTsTsxFiles(dir: string): Promise<string[]> {
  const result: string[] = []
  const entries = await readdir(dir, { withFileTypes: true, recursive: true })
  for (const entry of entries) {
    if (!entry.isDirectory() && /\.(ts|tsx)$/.test(entry.name)) {
      result.push(join(entry.parentPath ?? entry.path, entry.name))
    }
  }
  return result
}

async function main() {
  console.log("=== Kebab-case Rename Script ===\n")

  // Get all git-tracked files for checking
  const trackedRaw = execSync("git ls-files --full-name src/", {
    cwd: WEB_DIR,
    encoding: "utf-8",
  })
  const trackedFiles = new Set(trackedRaw.trim().split("\n"))

  // Step 1: Find all files that need renaming
  const allFiles = await findAllTsTsxFiles(SRC_DIR)
  const renameMap = new Map<string, string>()
  const stemMap = new Map<string, string>()

  for (const filePath of allFiles) {
    const ext = extname(filePath)
    const stem = basename(filePath, ext)

    if (isKebabCase(stem) || stem === "index") continue

    const kebabStem = toKebabCase(stem)
    if (kebabStem === stem) continue

    const newPath = join(dirname(filePath), kebabStem + ext)
    renameMap.set(filePath, newPath)

    if (!stemMap.has(stem)) {
      stemMap.set(stem, kebabStem)
    }
  }

  // Check for conflicts: use git ls-files (case-sensitive) + untracked files
  const allTrackedAndUntracked = new Set([
    ...trackedRaw.trim().split("\n"),
    ...execSync("git ls-files --others --exclude-standard src/", {
      cwd: WEB_DIR,
      encoding: "utf-8",
    })
      .trim()
      .split("\n")
      .filter(Boolean),
  ])
  let hasConflict = false
  for (const [oldPath, newPath] of renameMap) {
    const relNew = relative(WEB_DIR, newPath)
    const relOld = relative(WEB_DIR, oldPath)
    if (allTrackedAndUntracked.has(relNew) && relNew !== relOld) {
      console.error(`CONFLICT: ${relOld} → ${relNew} (target exists!)`)
      hasConflict = true
    }
  }
  if (hasConflict) process.exit(1)

  console.log(`Files to rename: ${renameMap.size}\n`)

  // Step 2: Rename files
  console.log("--- Renaming files ---")
  for (const [oldPath, newPath] of renameMap) {
    const relOld = relative(WEB_DIR, oldPath)
    const relNew = relative(WEB_DIR, newPath)
    const oldBase = basename(oldPath).toLowerCase()
    const newBase = basename(newPath).toLowerCase()
    const isTracked = trackedFiles.has(relOld)

    try {
      if (isTracked) {
        if (oldBase === newBase) {
          // Pure case change — two-step for case-insensitive FS
          const tmpRel = relOld + ".tmp-rename"
          execSync(`git mv "${relOld}" "${tmpRel}"`, { cwd: WEB_DIR, stdio: "pipe" })
          execSync(`git mv "${tmpRel}" "${relNew}"`, { cwd: WEB_DIR, stdio: "pipe" })
        } else {
          execSync(`git mv "${relOld}" "${relNew}"`, { cwd: WEB_DIR, stdio: "pipe" })
        }
      } else {
        if (oldBase === newBase) {
          const tmpPath = oldPath + ".tmp-rename"
          await rename(oldPath, tmpPath)
          await rename(tmpPath, newPath)
        } else {
          await rename(oldPath, newPath)
        }
      }
      console.log(`  ${relOld} → ${basename(relNew)}`)
    } catch (err: unknown) {
      const msg = err instanceof Error ? err.message : String(err)
      console.error(`  FAIL ${relOld}: ${msg}`)
    }
  }

  // Step 3: Update import paths in ALL source files
  console.log("\n--- Updating import paths ---")
  const updatedFiles = await findAllTsTsxFiles(SRC_DIR)
  let filesUpdated = 0
  let totalReplacements = 0

  for (const filePath of updatedFiles) {
    let content = await readFile(filePath, "utf-8")
    let fileReplacements = 0

    content = content.replace(
      /(?:from|import)\s+['"]([^'"]+)['"]/g,
      (match, importPath: string) => {
        if (!importPath.startsWith(".") && !importPath.startsWith("@/")) return match

        const segments = importPath.split("/")
        const lastSegment = segments[segments.length - 1]

        if (stemMap.has(lastSegment)) {
          segments[segments.length - 1] = stemMap.get(lastSegment)!
          fileReplacements++
          return match.replace(importPath, segments.join("/"))
        }

        return match
      },
    )

    if (fileReplacements > 0) {
      await writeFile(filePath, content)
      filesUpdated++
      totalReplacements += fileReplacements
    }
  }

  console.log(`Files with import updates: ${filesUpdated}`)
  console.log(`Total import paths updated: ${totalReplacements}`)
  console.log("\n=== Done ===")
}

main().catch((err) => {
  console.error("Fatal error:", err)
  process.exit(1)
})
