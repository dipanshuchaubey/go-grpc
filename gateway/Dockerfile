FROM golang:1.22-alpine AS build

WORKDIR /app

COPY . .

RUN go build -C cmd -o gateway

FROM alpine:3.14

COPY --from=build /app/cmd/gateway /bin/

EXPOSE 5000

CMD ["/bin/gateway"] 
