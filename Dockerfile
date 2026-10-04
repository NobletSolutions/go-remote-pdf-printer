FROM fedora:44 as builder

RUN dnf install -y golang

WORKDIR /app

COPY go.mod go.sum ./
COPY *.go .
COPY ./docs ./docs

RUN go mod download

RUN CGO_ENABLED=0 GOOS=linux go build -o ./remote-pdf-printer

FROM fedora:44 as prod
LABEL org.opencontainers.image.authors="nathanael@noblet.ca"

WORKDIR /app

RUN dnf install -y poppler-utils && dnf clean all
COPY css ./css
COPY docs/swagger* ./docs/
COPY --from=builder /app/remote-pdf-printer /app/remote-pdf-printer

EXPOSE 3000
CMD ["/app/remote-pdf-printer"]
