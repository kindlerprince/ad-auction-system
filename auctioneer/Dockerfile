FROM ubuntu:18.04

WORKDIR /app
COPY ./auctioneer ./
ENV AUCTIONEER_PORT=8085
EXPOSE 8085
CMD ["./auctioneer"]
