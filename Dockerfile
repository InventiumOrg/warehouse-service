FROM docker.io/golang:1.24

WORKDIR /app

COPY go.mod go.sum ./

RUN go mod download

COPY . .

RUN ls -lhR .

RUN CGO_ENABLED=0 GOOS=linux go build -v -o ./warehouse-service .

CMD ["./warehouse-service"]