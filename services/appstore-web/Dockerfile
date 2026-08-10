# The public storefront for appstore.gerege.mn.
#
# Two stages for the same reason the platform frontend has two: the build needs
# the whole toolchain and the runtime needs a server and a .next directory.
FROM node:20-alpine AS builder
WORKDIR /app

COPY package.json package-lock.json ./
RUN npm ci

COPY . .

# The registry address is read at run time, not baked in: the storefront calls
# it from the server, so it can be a loopback address that differs per host.
# Only NEXT_PUBLIC_* values need to be known here, and there is one — the
# developer console's address, which appears as a link in the footer.
ARG NEXT_PUBLIC_DEVELOPER_URL=https://developer.gerege.mn
ENV NEXT_PUBLIC_DEVELOPER_URL=$NEXT_PUBLIC_DEVELOPER_URL
RUN npm run build

FROM node:20-alpine AS runner
WORKDIR /app
ENV NODE_ENV=production

COPY --from=builder /app/package.json /app/package-lock.json ./
RUN npm ci --omit=dev
COPY --from=builder /app/.next ./.next
COPY --from=builder /app/public ./public

USER node
EXPOSE 3000
CMD ["npm", "start"]
