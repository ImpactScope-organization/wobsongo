# WobSongo frontend

The WobSongo infodemic dashboard: SvelteKit (Svelte 5), Tailwind CSS v4,
shadcn-svelte and LayerChart, rendered on the server.

**Read [`tr.md`](./tr.md) first.** It is the spec for architecture, testing,
forms, accessibility and auth.

## Setup

Tools come from devbox at the repo root (Node, pnpm). From `frontend/`:

```sh
pnpm install
pnpm exec playwright install chromium   # once; the Storybook tests run in it
cp .env.example .env                    # API_HOST=stub://local
```

`API_HOST` picks the data source, and the server won't start without it:
`stub://local` runs on the built-in stub data (an in-memory store, reset on
restart), an `http(s)://` URL on the backend. With stubs, sign in as
`a.ouedraogo@wobsongo.org` / `wobsongo-demo-2026`, or sign up. The backend
repositories are placeholders that answer "not available yet" until the
OpenAPI client lands (`TODO(openapi)` in `src/lib/server/repos/api`).

## Commands

| Command          | What it does                                                   |
| ---------------- | -------------------------------------------------------------- |
| `pnpm dev`       | Dev server                                                     |
| `pnpm storybook` | Component workshop on port 6006                                |
| `pnpm test`      | Unit tests (server render) and Storybook play tests (Chromium) |
| `pnpm check`     | Svelte and TypeScript checks                                   |
| `pnpm lint`      | Prettier and ESLint                                            |
| `pnpm format`    | Rewrite files with Prettier                                    |
| `pnpm build`     | Production build (Node adapter); `pnpm preview` to run it      |

`make fecheck` at the repo root runs check, lint and test, like CI.

## Running the build

`node build` serves the app. SvelteKit refuses form posts whose `Origin` doesn't
match the site, and adapter-node assumes `https`. Behind a TLS proxy, set
`PROTOCOL_HEADER=x-forwarded-proto` (and `HOST_HEADER=x-forwarded-host` if the
proxy rewrites the host). Over plain `http` locally, use `pnpm preview` instead.

## Layout

- `src/lib/domain`: types and business rules, framework-free
- `src/lib/components`: presentational components by view, each with a
  `*.test.ts` (rendered with `svelte/server`) and `*.stories.svelte`
- `src/lib/components/ui`: shadcn-svelte components, added with its CLI
- `src/lib/fixtures`: sample data for stories, tests and stubs
- `src/lib/server`: server-only code. `services/` holds the business logic and
  the repository interfaces it needs; `repos/stub` and `repos/api` implement
  them; `backend.ts` picks one from `API_HOST`; `auth/` handles the session
  cookies; `hooks.ts` sets `locals` and guards the `(app)` routes
- `src/routes`: pages, loaders and form actions
