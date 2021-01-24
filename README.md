# Ad Auction System

* This a simple auction bidder system made in golang

## Execution
* Execute the docker-compose.yml  
`docker compose up`  
* Send post request ad request to the auction with auction_id
* for e.g you can use curl utitlity
`curl -X POST http://localhost:8085/adrequest -d '{"auction_id" : "1020"}'`

## Troubleshooting
* If you have made some changes and it is not showing then  
`docker compose up --build`


# Adding new bidders
* Currently only two bidders are added but we can scale this simply by
adding these lines

```
bidder3:
  environment:
	- ID=BIDDER3
	- DELAY=8
	- VALUE=832
	- PORT=8730
	- AUCTIONEER_URL=auctioneer
	- AUCTIONEER_PORT=8085
  build:
	context: .
	dockerfile: ./bidder/docker/Dockerfile
  ports:
  - "8730:8730"
  depends_on:
  - "auctioneer"
```
