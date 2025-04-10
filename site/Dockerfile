# frontend-deps
FROM node:22-alpine as frontend-deps
WORKDIR /app
RUN npm i -g pnpm
COPY package.json pnpm-lock.yaml ./
RUN pnpm install --frozen-lockfile
COPY . .

# backend-builder
FROM golang:1.24 as backend-builder
WORKDIR /app
RUN curl -fsSL https://deb.nodesource.com/setup_22.x | bash -
RUN apt-get -y update
RUN apt-get -y install nodejs
RUN npm i -g pnpm
COPY go.mod go.sum ./
RUN go mod download
COPY . .
COPY --from=frontend-deps /app/node_modules ./node_modules
RUN npx @tailwindcss/cli -i ./css/tailwind-input.css -o ./css/tailwind-output.css
RUN go run ./go/cmd/build --no-binary
RUN CGO_ENABLED=0 GOOS=linux go build -mod=readonly -v -o /app/main ./go/cmd/app

# server
FROM node:22-alpine
WORKDIR /app
RUN apk --no-cache add ca-certificates
COPY --from=backend-builder /app/main /app
RUN ls -la
RUN adduser -D myuser
USER myuser
ENTRYPOINT ["/app/main"]
