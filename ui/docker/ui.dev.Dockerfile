FROM node:22.8.0-alpine

WORKDIR /app

COPY package.json package-lock.json* ./
RUN npm ci

ENV NEXT_TELEMETRY_DISABLED=1

COPY src ./src
COPY public ./public
COPY next.config.mjs .
COPY tsconfig.json .
COPY tailwind.config.ts .
COPY postcss.config.js .

CMD ["npm", "run", "dev"]
