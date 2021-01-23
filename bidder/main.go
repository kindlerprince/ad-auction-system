package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
)

const (
	VALUE           = 154.36
	BIDDER_PORT     = "8081"
	AUCTIONEER_URL  = "localhost"
	AUCTIONEER_PORT = "8080"
)

var BIDDERID string

type customResponse struct {
	Status  string      `json:"status,omitempty"`
	Message string      `json:"message,omitempty"`
	Data    interface{} `json:"data,omitemtpy"`
}

type auctionAD struct {
	AuctionID string `json:"auction_id,omitempty"`
}

type identity struct {
	BidderID string `json:"bidder_id,omitempty"`
}

type bidRequest struct {
	BidderID string  `json:"bidder_id,omitempty"`
	Value    float32 `json:"value,omitempty"`
}

func main() {
	fmt.Println("Bidder System")
	var err error
	BIDDERID, err = getBidderId()
	fmt.Printf("BIDDERID = %s\n", BIDDERID)
	if err != nil {
		fmt.Printf("No bidder id assigned : %s\n", err.Error())
		return
	}
	err = registration(AUCTIONEER_URL, AUCTIONEER_PORT, 3)
	if err != nil {
		fmt.Printf("Registration Failed : %s\n", err.Error())
		return
	}
	fmt.Println("Registration Successful")
	r := mux.NewRouter()
	r.HandleFunc("/auction/"+BIDDERID, auctionAdHandler).Methods(http.MethodPost)
	http.Handle("/", r)
	err = http.ListenAndServe(":"+BIDDER_PORT, nil)
	if err != nil {
		fmt.Printf("Error in starting server : %s", err.Error())
	}
}

func auctionAdHandler(w http.ResponseWriter, r *http.Request) {
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
		var ad auctionAD
		err = json.Unmarshal(biddingBody, &ad)
		if err != nil {
			fmt.Printf("Error in unmarshalling body : %s", err.Error())
			response.Message = "Error in parsing JSON"
			writeSuccessMessage(w, r, response)
			return
		}
		fmt.Printf("%+v\n", ad)
		response = customResponse{
			Message: "Bid Request Placed",
		}
		writeSuccessMessage(w, r, response)
		time.Sleep(5 * time.Second)
		askForBid()
	}
}

func askForBid() error {
	bid := bidRequest{
		BidderID: BIDDERID,
		Value:    VALUE,
	}
	payload, _ := json.Marshal(bid)
	resp, err := http.Post("http://"+AUCTIONEER_URL+":"+AUCTIONEER_PORT+"/bidding", "application/json", bytes.NewBuffer(payload))
	if err != nil {
		fmt.Printf("Sending request failed to auctioneer %s", err.Error())
		return err
	}
	body, err := ioutil.ReadAll(resp.Body)
	defer resp.Body.Close()
	if err != nil {
		fmt.Printf("Error in reading body : %s", err.Error())
		return err
	}
	fmt.Printf("%s", string(body))
	if resp.StatusCode == http.StatusOK {
		return nil
	}
	return nil
}

func registration(url, port string, time_ms int) error {
	id := identity{
		BidderID: BIDDERID,
	}
	fmt.Printf("%+v", id)
	payload, _ := json.Marshal(id)
	resp, err := http.Post("http://"+url+":"+port+"/registration", "application/json", bytes.NewBuffer(payload))
	if err != nil {
		fmt.Printf("Sending request failed to auctioneer %s", err.Error())
		return err
	}
	body, err := ioutil.ReadAll(resp.Body)
	defer resp.Body.Close()
	if err != nil {
		fmt.Printf("Error in reading body : %s", err.Error())
		return err
	}
	fmt.Printf("%s", string(body))
	if resp.StatusCode == http.StatusOK {
		return nil
	}
	return fmt.Errorf("Unable to register :%d", resp.StatusCode)
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

func setupResponse(w *http.ResponseWriter, req *http.Request) {
	(*w).Header().Set("content-type", "application/json")
}

func getBidderId() (string, error) {
	uid, err := uuid.NewRandom()
	if err != nil {
		fmt.Printf("Error in creating bidder id : %s", err.Error())
		return "", err
	}
	return uid.String(), nil
}
