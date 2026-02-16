# Frontend Agent Documentation

### Making Changes
When making changes to this file, ALWAYS make the same update to CLAUDE.md in the same directory.

## Design System — Media Type Theming

Movies use orange (`movie-*`), TV shows use blue (`tv-*`). CSS variables defined in `src/index.css` with shades 50-950 (OKLCH color space).

**Tailwind usage:** `text-movie-500`, `bg-tv-400`, `border-movie-600`, etc.

**Conventions:**
- Movie content -> `movie-*` classes; TV content -> `tv-*` classes
- Mixed content -> `media-gradient` utilities (`bg-media-gradient`, `text-media-gradient`)
- Backgrounds: use `/10` or `/15` opacity (e.g., `bg-movie-500/10`)
- Text on dark backgrounds: 400 shades; borders/accents: 500 shades
- Glow effects for interactivity: `glow-movie`, `glow-tv`, `hover:glow-movie`, `glow-media`
- Live palette preview at `/dev/colors` in developer mode

## Base UI (NOT Radix)

This project uses **Base UI** (`@base-ui/react`) for shadcn/ui components.

**Trigger composition — use `render` prop, NOT Radix-style `asChild`:**
```tsx
// WRONG
<TooltipTrigger asChild><Button>Click</Button></TooltipTrigger>

// CORRECT
<TooltipTrigger render={<Button />}>Click</TooltipTrigger>
```
Applies to: `TooltipTrigger`, `DialogTrigger`, `PopoverTrigger`, `DropdownMenuTrigger`, etc.

**SelectValue gotcha:** `SelectValue` renders the raw `value`, not the display label. When value differs from label, render the label manually in the trigger:
```tsx
<SelectTrigger>
  {OPTIONS.find((o) => o.value === selected)?.label}
</SelectTrigger>
```

## React Patterns

### State Synchronization
Do NOT use `useEffect` to sync state from props (triggers `react-hooks/set-state-in-effect` lint error). Use render-time state adjustment:
```tsx
const [formData, setFormData] = useState(null)
const [prevData, setPrevData] = useState(data)

if (data !== prevData) {
  setPrevData(data)
  if (data) setFormData(data)
}
```

### Hook Extraction
Separate logic from presentation. Extract state, queries, mutations, handlers into `use-<feature>.ts` in the same directory. Component should be a thin JSX shell (<50 lines).

### Error Handling
- Mutations: `onError` callback for user-facing toast
- Fire-and-forget: `void` prefix (e.g., `void queryClient.invalidateQueries(...)`)
- Never use `.catch(() => {})` — it swallows errors
- Never make `onSuccess` async

### Conditional Rendering
Priority: early returns > component map > single ternary. **Never nest ternaries.**

### Null Handling
Use `??` by default. Use `||` only when falsy coalescing is intentional (0, `""`, NaN should fallback) — add a comment explaining why.
