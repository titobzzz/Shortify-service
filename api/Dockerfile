FROM golang:alpine as builder

RUN mkdir /builder

ADD . /build/

WORKDIR /build

RUN go build -o main .

#stage 2

FROM alpine

RUN apk add --no-cache shadow

RUN adduser -S -D -h /app appuser

USER  appuser

COPY . /app

COPY --from=builder /build/main /app/

WORKDIR /app

EXPOSE 3000  
 
CMD ["./main"]
