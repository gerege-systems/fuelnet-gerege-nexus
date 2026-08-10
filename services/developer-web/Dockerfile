# The publishing console for developer.gerege.mn.
#
# Two stages for the same reason the platform frontend has two: the build needs
# the whole toolchain and the runtime needs a server and a .next directory.
FROM node:20-alpine AS builder
WORKDIR /app

COPY package.json package-lock.json ./
RUN npm ci

COPY . .

# The registry address, the issuer and the client id are read at run time: every
# call to either happens on the server, so nothing about them belongs in the
# browser bundle. The one exception is the storefront link in the header.
ARG NEXT_PUBLIC_STOREFRONT_URL=https://appstore.gerege.mn
ENV NEXT_PUBLIC_STOREFRONT_URL=$NEXT_PUBLIC_STOREFRONT_URL
RUN npm run build

FROM node:20-alpine AS runner
WORKDIR /app
ENV NODE_ENV=production

COPY --from=builder /app/package.json /app/package-lock.json ./
RUN npm ci --omit=dev
COPY --from=builder /app/.next ./.next
# No public/ directory: the console ships no static assets — its only images
# are the letter marks it draws itself. Copying a directory that is not there
# fails the build, and an empty one cannot be committed to git to make it exist.

USER node
EXPOSE 3000
CMD ["npm", "start"]
