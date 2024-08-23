FROM golang:1.23.0 AS build

WORKDIR /app

COPY . /app

RUN go mod download
RUN CGO_ENABLED=0 go build -o quizcrud .

FROM ubuntu
WORKDIR /app
COPY --from=build /app/quizcrud /app/quizcrud
COPY --from=build /app/config.yaml /app/config.yaml

EXPOSE 8081
CMD ["/app/quizcrud"]
