# Build context is the repository root (pnpm workspace). Serves the production build with `next start`.
# Keep in sync with .node-version and CI.
FROM node:26.13.0-alpine AS build
RUN npm install --global pnpm@12.16.1
ARG NEXT_PUBLIC_API_BASE_URL
ARG NEXT_PUBLIC_SITE_URL
ENV NEXT_PUBLIC_API_BASE_URL=$NEXT_PUBLIC_API_BASE_URL NEXT_PUBLIC_SITE_URL=$NEXT_PUBLIC_SITE_URL
WORKDIR /repo
COPY . .
RUN pnpm install --frozen-lockfile \
 && pnpm --filter @dealo/web build

FROM node:26.13.0-alpine AS run
RUN npm install --global pnpm@12.16.1
ENV NODE_ENV=production
WORKDIR /repo
COPY --from=build /repo /repo
USER node
EXPOSE 3000
CMD ["pnpm", "--filter", "@dealo/web", "start", "--port", "3000"]
