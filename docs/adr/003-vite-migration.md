# ADR-003: Migrate Frontend Build Tooling from CRA to Vite

## Status
Accepted

## Context

The frontend currently uses **Create React App** (CRA) via `react-scripts` 5.0.1. CRA was deprecated by the React core team in early 2023 and has received no meaningful upstream updates since then. The bundler it wraps (Webpack 5) is slow by modern standards, and CRA's pinned transitive dependency tree has open security advisories with no upstream fix path.

The React documentation now recommends Vite as the preferred tool for new SPA projects.

Current project specifics relevant to this decision:
- Five source files (`App.tsx`, `Dashboard.tsx`, `Settings.tsx`, `Trends.tsx`, `Archives.tsx`) reference `process.env.REACT_APP_API_URL`
- `docker-compose.yml` injects `REACT_APP_API_URL` and `WDS_SOCKET_PORT` (CRA-specific WebSocket env var)
- No frontend tests exist today (CRA bundles Jest; this is not currently utilized)
- The app is served inside Docker; container runs on port 3000, mapped to host 3001 in docker-compose

## Decision

Migrate to **Vite** with `@vitejs/plugin-react` as the build tool and dev server. TypeScript is upgraded from `^4.9` to `^5.0` to enable `"moduleResolution": "bundler"`.

## Consequences

### Files Changed

| File | Change |
|---|---|
| `docs/adr/003-vite-migration.md` | This file |
| `frontend/package.json` | Remove `react-scripts`; add `vite`, `@vitejs/plugin-react` as devDependencies; update `scripts`; remove CRA-specific `eslintConfig` and `browserslist` fields |
| `frontend/vite.config.ts` | New — React plugin, dev server bound to `0.0.0.0:3000` for Docker |
| `frontend/tsconfig.json` | Update `target` → `ESNext`, `moduleResolution` → `bundler`, add `"types": ["vite/client"]` |
| `frontend/index.html` | Moved from `public/` to project root; `<script type="module" src="/src/index.tsx">` added |
| `frontend/Dockerfile` | Update `CMD` to `npm run dev` |
| `docker-compose.yml` | Rename `REACT_APP_API_URL` → `VITE_API_URL`; remove `WDS_SOCKET_PORT` |
| `frontend/src/*.tsx` (×5) | `process.env.REACT_APP_API_URL` → `import.meta.env.VITE_API_URL` |
| `frontend/src/vite-env.d.ts` | New — `/// <reference types="vite/client" />` for `ImportMeta.env` type support |
| `frontend/package.json` (`@types/node`) | Bumped from `^16.18.0` → `^20.0.0` to satisfy Vite 5 peer dep requirement |

### Pros

1. **Speed**: Cold dev server start drops from ~10–15 s to under 1 s; HMR is near-instant (module-level, not full-page reload)
2. **Actively maintained**: Vite has a regular release cadence and is the current React-team recommendation; CRA is archived
3. **No security baggage**: CRA's pinned Webpack/Babel/PostCSS graph carries CVEs with no upstream fix; Vite's dependency surface is far smaller
4. **Explicit config**: `vite.config.ts` is a plain TypeScript file — easy to read and extend, versus CRA's hidden Webpack config (eject-or-nothing)
5. **Smaller install**: Removes the heavy `react-scripts` transitive dependency tree; `npm install` and Docker build are faster
6. **Better DX**: Native ESM in dev means no bundling step during development; source maps are accurate

### Cons

1. **Migration effort**: Eight files require changes across config, source, and Docker layers
2. **Env var rename**: All five source files change `process.env.REACT_APP_API_URL` → `import.meta.env.VITE_API_URL`, and `docker-compose.yml` must also be updated — easy to miss one
3. **`index.html` moves**: Vite expects `index.html` at the project root; the `<script type="module">` entry must be wired up manually
4. **No built-in test runner**: Vite does not bundle Jest. Frontend tests (when added) will require a separate decision — **Vitest** (recommended) or standalone Jest with `ts-jest`. Deferred since no frontend tests exist today.
5. **`@ant-design/charts` compatibility**: Uses dynamic imports and Canvas; smoke-test after migration to confirm chart rendering is unaffected
6. **Peer dependency conflict**: Vite 5 requires `@types/node@^18.0.0 || >=20.0.0`; the existing `^16.18.0` pin caused `npm install` to fail. Required a manual bump and lock file regeneration.

## Alternatives Considered

### Stay on CRA
Rejected. CRA is deprecated with no upstream security fixes and no customization path short of ejecting.

### Eject CRA
Rejected. Exposes ~700 lines of Webpack config that become a permanent maintenance burden without solving the underlying problem.

### Migrate to Next.js
Rejected. Next.js is a full framework (SSR, routing, RSC, etc.), not a build-tool replacement. The frontend is a plain SPA backed by a separate Go API; the additional complexity is unwarranted.

## Out of Scope

- Test runner selection (deferred — no frontend tests exist today)
- Per-environment `.env` file strategy (current single default is sufficient)
- CSS/PostCSS configuration beyond Vite defaults
