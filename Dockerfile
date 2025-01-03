FROM fedora:41 as builder

RUN dnf install -y golang

WORKDIR /app

COPY go.mod go.sum ./
COPY *.go .

RUN go mod download

RUN CGO_ENABLED=0 GOOS=linux go build -o ./remote-pdf-printer

FROM fedora:41 as prod

WORKDIR /app

COPY css ./css
COPY --from=builder /app/remote-pdf-printer /app/remote-pdf-printer
RUN dnf install -y poppler-utils && dnf clean all

EXPOSE 3000
CMD ["/app/remote-pdf-printer"]
