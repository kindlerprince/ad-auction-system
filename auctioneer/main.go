package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/gorilla/mux"
)

type customResponse struct {
	Status  string      `json:"status,omitempty"`
	Message string      `json:"message,omitempty"`
	Data    interface{} `json:"data,omitemtpy"`
}

type bidder struct {
	BidderID   string `json:"bidder_id,omitempty"`
	BidderPort string `json:"bidder_port,omitempty"`
}

type bidderDetails struct {
	Port string
}

type bidRequest struct {
	BidderID string  `json:"bidder_id,omitempty"`
	Value    float32 `json:"value,omitempty"`
}

type adRequest struct {
	AuctionID string `json:"auction_id,omitempty"`
}

var (
	bidderMap       map[string]bidderDetails
	bidMap          map[string]float32
	bidList         []bidRequest
	auctionGoing    bool
	counter         int
	AUCTIONEER_PORT string
)

const (
	BIDDER_URL = "localhost"
)

func main() {
	fmt.Println("Ad Auction System")
	port, err := getPort()
	if err != nil {
		fmt.Printf("%s\n", err.Error())
		return
	}
	AUCTIONEER_PORT = strconv.Itoa(port)
	fmt.Printf("AUCTIONEER PORT  %s\n", AUCTIONEER_PORT)
	auctionGoing = false
	r := mux.NewRouter()
	r.HandleFunc("/adrequest", adRequestHandler).Methods(http.MethodPost)
	r.HandleFunc("/registration", bidderRegistrationHandler).Methods(http.MethodPost)
	r.HandleFunc("/bidderlist", bidderListHandler).Methods(http.MethodGet)
	r.HandleFunc("/bidding", biddingHandler).Methods(http.MethodPost)
	http.Handle("/", r)
	err = http.ListenAndServe(":"+AUCTIONEER_PORT, nil)
	if err != nil {
		fmt.Printf("Error in starting server : %s", err.Error())
	}
}

func adRequestHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		var response customResponse
		setupResponse(&w, r)
		adReq, err := ioutil.ReadAll(r.Body)
		if err != nil {
			fmt.Printf("Error in reading body : %s", err.Error())
			response.Message = "Error in reading data"
			writeSuccessMessage(w, r, response)
			return
		}
		defer r.Body.Close()
		var ad adRequest
		err = json.Unmarshal(adReq, &ad)
		if err != nil {
			fmt.Printf("Error in unmarshalling body : %s", err.Error())
			response.Message = "Error in parsing JSON"
			writeSuccessMessage(w, r, response)
			return
		}
		auctionGoing = true
		bidPlacing(ad.AuctionID)
		time.Sleep(20 * time.Second)
		resBid := bidResult()
		counter++
		fmt.Printf("Winner for %d round : %+v\n", counter, resBid)
		auctionGoing = false
		response = customResponse{
			Message: "Bid Result",
			Data:    resBid,
		}
		writeSuccessMessage(w, r, response)
	}
}

func bidPlacing(auctionID string) {

	var ad adRequest
	for bidderID, bidderDet := range bidderMap {
		ad.AuctionID = auctionID
		payload, _ := json.Marshal(ad)
		resp, err := http.Post("http://"+BIDDER_URL+":"+bidderDet.Port+"/auction/"+bidderID, "application/json", bytes.NewBuffer(payload))
		if err != nil {
			fmt.Printf("Sending request failed to bidder[bidder id : %s ] %s", bidderID, err.Error())
			continue
		}
		fmt.Printf("Placing bid for bidder %s\n", bidderID)
		body, err := ioutil.ReadAll(resp.Body)
		defer resp.Body.Close()
		if err != nil {
			fmt.Printf("Error in reading body : %s", err.Error())
		}
		fmt.Printf("Response of placing bidder %s\n", string(body))
		if resp.StatusCode != http.StatusOK {
			fmt.Printf("Unable to place pid :%d\n", resp.StatusCode)
		}
	}
}

func bidResult() bidRequest {
	var finalBid bidRequest
	for _, bid := range bidList {
		if bid.Value > finalBid.Value {
			finalBid.BidderID = bid.BidderID
			finalBid.Value = bid.Value
		}
	}
	return finalBid
}

func biddingHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		var response customResponse
		setupResponse(&w, r)
		biddingBody, err := ioutil.ReadAll(r.Body)
		if err != nil {
			fmt.Printf("Error in reading body : %s", err.Error())
			response.Message = "Error in reading data"
			writeSuccessMessage(w, r, response)
			return
		}
		defer r.Body.Close()
		var bid bidRequest
		err = json.Unmarshal(biddingBody, &bid)
		if err != nil {
			fmt.Printf("Error in unmarshalling body : %s", err.Error())
			response.Message = "Error in parsing JSON"
			writeSuccessMessage(w, r, response)
			return
		}
		fmt.Printf("%+v\n", bid)
		err = checkBidding(bid)
		if err != nil {
			response = customResponse{
				Message: err.Error(),
			}
		} else {
			response = customResponse{
				Message: "Bid Request Placed",
			}
		}
		writeSuccessMessage(w, r, response)
	}
}

func checkBidding(bid bidRequest) error {
	if _, ok := bidderMap[bid.BidderID]; !ok {
		return fmt.Errorf("Bidder with id %s is not registered", bid.BidderID)
	}
	bidList = append(bidList, bid)
	return nil
}

func bidderRegistrationHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		var response customResponse
		setupResponse(&w, r)
		regReq, err := ioutil.ReadAll(r.Body)
		if err != nil {
			fmt.Printf("Error in reading body : %s", err.Error())
			response.Message = "Error in reading data"
			writeSuccessMessage(w, r, response)
			return
		}
		defer r.Body.Close()
		var bidderReg bidder
		err = json.Unmarshal(regReq, &bidderReg)
		if err != nil {
			fmt.Printf("Error in unmarshalling body : %s", err.Error())
			response.Message = "Error in parsing JSON"
			writeSuccessMessage(w, r, response)
			return
		}
		fmt.Printf("%+v\n", bidderReg)
		if auctionGoing == true {
			response.Message = "Registration Failed OnGoing Auction"
		} else {
			bidderRegistration(bidderReg)
			response.Message = "Registration Successful"
			fmt.Printf("Registration Success for bidder %s\n", bidderReg.BidderID)
		}
		writeSuccessMessage(w, r, response)
	}
}
func bidderRegistration(bidderEntity bidder) {
	if bidderMap == nil {
		bidderMap = make(map[string]bidderDetails)
	}
	bidderMap[bidderEntity.BidderID] = bidderDetails{
		Port: bidderEntity.BidderPort,
	}
}

func bidderListHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		var bidderList []string
		for bidderEntity, _ := range bidderMap {
			bidderList = append(bidderList, bidderEntity)
		}
		response := customResponse{
			Message: "List of Bidders",
			Data:    bidderList,
		}
		writeSuccessMessage(w, r, response)
	}
}

func setupResponse(w *http.ResponseWriter, req *http.Request) {
	(*w).Header().Set("content-type", "application/json")
}
func writeSuccessMessage(w http.ResponseWriter, r *http.Request, data interface{}) {
	fmt.Printf(
		"%s %s \n",
		r.Method,
		r.RequestURI,
	)
	w.WriteHeader(http.StatusOK)
	body, err := json.Marshal(data)
	if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	w.Write(body)
}
func getPort() (int, error) {
	port := os.Getenv("AUCTIONEER_PORT")
	return strconv.Atoi(port)
}
