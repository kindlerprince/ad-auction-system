FROM golang

WORKDIR /app
COPY ./auctioneer .
RUN go build

#ENV AUCTIONEER_PORT=8085

CMD ["./auctioneer"]
