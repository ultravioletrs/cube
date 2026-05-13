# Veda — UI

The frontend for Veda, a document intelligence platform for natural language search and Q&A over enterprise documents.

## Prerequisites

- [Node.js](https://nodejs.org/) v20 or later
- [npm](https://www.npmjs.com/) v10 or later (comes with Node.js)

## Getting started

Install dependencies:

```bash
npm install
```

Start the development server:

```bash
npm run dev
```

The app runs at `http://localhost:5173` by default.

## Available scripts

| Script | Description |
|--------|-------------|
| `npm run dev` | Start development server with HMR |
| `npm run build` | Type-check and build for production (output: `dist/`) |
| `npm run preview` | Serve the production build locally |
| `npm run lint` | Run ESLint |

## Tech stack

- [React 19](https://react.dev/) + [TypeScript](https://www.typescriptlang.org/)
- [Vite](https://vite.dev/) — build tool and dev server
- [Tailwind CSS v4](https://tailwindcss.com/) — utility-first styling
- [shadcn/ui](https://ui.shadcn.com/) + [Base UI](https://base-ui.com/) — component libraries
- [React Router v7](https://reactrouter.com/) — client-side routing
- [TanStack Query v5](https://tanstack.com/query) — server state management

## Path aliases

`@/` resolves to `src/`, so imports like `@/components/Button` work from anywhere in the project.
