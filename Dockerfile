FROM golang:1.23-alpine
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY *.go ./
COPY internal/ ./internal/
RUN go build

ENV DB_URL $DB_URL

CMD ["./office-games"]
EXPOSE 8080