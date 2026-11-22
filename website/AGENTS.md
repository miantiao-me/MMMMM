# Website

## Build and verify

- Run `bun install --frozen-lockfile` from the repository root. Run `bun run dev` and `bun run build` from `website/`. Vite prerenders the site into `dist/client/`.
- Run `bun run build` after website changes. For TypeScript changes also run `bun run tsc --noEmit`. The standalone check currently reports TS2591 for `process` in `vite.config.ts` because Node types are missing.
- Keep `src/routeTree.gen.ts`, `.tanstack/` and `dist/` out of Git. Vite regenerates the route tree from files in `src/routes/`.

## Content and release data

- Update both English and Simplified Chinese when changing user-facing copy. Shared components use `useT()` from `src/i18n.tsx`. Check the generated English page and its `/zh/` counterpart.
- `vite.config.ts` reads the latest release and plugin list from GitHub at build time. A 404 for the latest release renders the empty download state. Other failed GitHub requests fail the build.
