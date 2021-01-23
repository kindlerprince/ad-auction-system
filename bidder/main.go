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
	VALUE = 154.36
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
	uid, err := uuid.NewRandom()
	if err != nil {
		fmt.Printf("Error in creating bidder id")
		return
	}
	BIDDERID = uid.String()
	err = registration("localhost", "8080", 3)
	if err != nil {
		fmt.Printf("Registration Failed : %s", err.Error())
		return
	}
	fmt.Println("Bidder System")
	r := mux.NewRouter()
	r.HandleFunc("/auction/"+BIDDERID, auctionAdHandler).Methods(http.MethodPost)
	http.Handle("/", r)
	err = http.ListenAndServe(":8081", nil)
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
		fmt.Printf("%s\n%s\n%s\n", r.Host, r.URL.Port(), r.URL.Path)
		askForBid(r.Host + r.URL.Port() + r.URL.Path)
	}
}

func askForBid(url string) error {
	bid := bidRequest{
		BidderID: BIDDERID,
		Value:    VALUE,
	}
	payload, _ := json.Marshal(bid)
	resp, err := http.Post("http://"+url+"/bidding", "application/json", bytes.NewBuffer(payload))
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
