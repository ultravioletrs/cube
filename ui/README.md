# Cube AI UI

The React frontend for Cube AI.

## Run the complete application

The recommended way to run Cube AI, including the UI and all required backend
services, is from the repository root:

```bash
make up
```

Open `https://localhost` and accept the local self-signed TLS certificate if
your browser displays a warning.

Sign in with the default local administrator:

```text
Username: admin
Password: m2N2Lfno
```

No commands from this directory are required when using the Docker-hosted UI.

## Prerequisites

For local frontend development:

- A running Cube AI backend stack (`make up` from the repository root)
- [Node.js](https://nodejs.org/) v20 or later
- [npm](https://www.npmjs.com/) v10 or later (comes with Node.js)

## Local frontend development

Run the following commands from the `ui/` directory:

Install dependencies:

```bash
npm install
```

Start the development server:

```bash
npm run dev
```

The app runs at `http://localhost:5173` by default.

The Vite development server proxies API requests to the local Cube AI backend.
Keep the Docker services started by `make up` running while developing the UI.

To make the development server available outside localhost, run:

```bash
npm run dev -- --host 0.0.0.0
```

## Available scripts

| Script | Description |
|--------|-------------|
| `npm run dev` | Start development server with HMR |
| `npm run build` | Type-check and build for production (output: `dist/`) |
| `npm run preview` | Serve the production build locally |
| `npm run lint` | Run ESLint |

## Chat record selection

Chat answers are grounded in indexed records (RAG). The records panel on the
right of the chat controls which records the model can retrieve from:

- **All records** (default) — every indexed record is active. No selection
  needed; the panel header shows `N active · all`. The chat sends no
  `record_ids`, so the backend searches all records via its index (scales to
  large corpora without enumerating the list).
- **Customized** — click **Customize** to limit the chat to a chosen allowlist.
  The search box queries records by name on the server (paginated, so it works
  past the client load cap). Toggle individual records, or **Select all** /
  **Clear** to (de)select every record matching the current search. **Reset**
  returns to all records. A selection is capped (1000 records); "all" is the way
  to use more.
- **Scoped** — opening chat from a Source or Record pins it to that scope:
  - A **source** scope limits the pool to that source's records, and can still
    be narrowed further with the customize filter.
  - A **single record** scope is locked and cannot be customized.

## Tech stack

- [React 19](https://react.dev/) + [TypeScript](https://www.typescriptlang.org/)
- [Vite](https://vite.dev/) — build tool and dev server
- [Tailwind CSS v4](https://tailwindcss.com/) — utility-first styling
- [shadcn/ui](https://ui.shadcn.com/) + [Base UI](https://base-ui.com/) — component libraries
- [React Router v7](https://reactrouter.com/) — client-side routing
- [TanStack Query v5](https://tanstack.com/query) — server state management

## Path aliases

`@/` resolves to `src/`, so imports like `@/components/Button` work from anywhere in the project.
