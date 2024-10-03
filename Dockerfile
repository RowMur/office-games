FROM golang:1.23-alpine AS go-builder
WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY *.go ./
COPY internal/ ./internal/
RUN go build

FROM node:22-alpine3.19 AS asset-builder
WORKDIR /app
COPY package.json ./
COPY package-lock.json ./
RUN npm install

COPY tailwind.config.js ./
COPY internal/assets/input.css ./internal/assets/
COPY internal/views/*.templ ./internal/views/
RUN npm run build-styles

FROM golang:1.23-alpine AS go-runner

COPY --from=asset-builder /app/internal/assets/ ./internal/assets/
COPY --from=go-builder /app/office-games ./
CMD ["./office-games"]
EXPOSE 8080