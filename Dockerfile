FROM golang:1.9 as oven
WORKDIR /go/src/github.com/xocasdashdash/govat/
COPY vat.go .
RUN CGO_ENABLED=0 GOOS=linux go build -o govat .

FROM alpine:latest  
RUN apk --no-cache add ca-certificates
WORKDIR /root/
COPY --from=oven /go/src/github.com/xocasdashdash/govat/ .
CMD ["./govat"]  
