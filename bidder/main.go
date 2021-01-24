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

var (
	BIDDERID        string
	VALUE           float64
	BIDDER_PORT     string
	TIME_DELAY      int
	AUCTIONEER_PORT string
	AUCTIONEER_URL  string
	BIDDER_URL      string
)

type customResponse struct {
	Status  string      `json:"status,omitempty"`
	Message string      `json:"message,omitempty"`
	Data    interface{} `json:"data,omitemtpy"`
}

type auctionAD struct {
	AuctionID string `json:"auction_id,omitempty"`
}

type identity struct {
	BidderID   string `json:"bidder_id,omitempty"`
	BidderPort string `json:"bidder_port,omitempty"`
	BidderURL  string `json:"bidder_url,omitempty"`
}

type bidRequest struct {
	BidderID string  `json:"bidder_id,omitempty"`
	Value    float64 `json:"value,omitempty"`
}

func main() {
	fmt.Println("|-----Bidder System------|")
	var err error
	BIDDERID, err = getBidderID()
	if err != nil {
		fmt.Printf("%s\n", err.Error())
		return
	}
	port, err := getBidderPort()
	if err != nil {
		fmt.Printf("%s\n", err.Error())
		return
	}
	BIDDER_PORT = strconv.Itoa(port)
	BIDDER_URL, err = getBidderURL()
	if err != nil {
		fmt.Printf("%s\n", err.Error())
		return
	}
	VALUE, err = getBidValue()
	if err != nil {
		fmt.Printf("%s\n", err.Error())
		return
	}
	port, err = getAuctioneerPort()
	if err != nil {
		fmt.Printf("%s\n", err.Error())
		return
	}
	AUCTIONEER_URL, err = getAuctioneerURL()
	if err != nil {
		fmt.Printf("%s\n", err.Error())
		return
	}
	AUCTIONEER_PORT = strconv.Itoa(port)
	TIME_DELAY, err = getTimeDelay()
	err = registration(AUCTIONEER_URL, AUCTIONEER_PORT)
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
			Message: "Place the Bid request",
		}
		writeSuccessMessage(w, r, response)
		time.Sleep(time.Duration(TIME_DELAY) * time.Second)
		askForBid()
	}
}

func askForBid() {
	bid := bidRequest{
		BidderID: BIDDERID,
		Value:    VALUE,
	}
	payload, _ := json.Marshal(bid)
	resp, err := http.Post("http://"+AUCTIONEER_URL+":"+AUCTIONEER_PORT+"/bidding", "application/json", bytes.NewBuffer(payload))
	if err != nil {
		fmt.Printf("Sending request failed to auctioneer %s", err.Error())
		return
	}
	body, err := ioutil.ReadAll(resp.Body)
	defer resp.Body.Close()
	if err != nil {
		fmt.Printf("Error in reading body : %s", err.Error())
		return
	}
	fmt.Printf("%s", string(body))
	if resp.StatusCode == http.StatusOK {
		fmt.Printf("Bid response sent is successfully")
	} else {
		fmt.Println("Bid response was not sent")
	}
}

func registration(auctioneerURL, auctioneerPort string) error {
	id := identity{
		BidderID:   BIDDERID,
		BidderPort: BIDDER_PORT,
		BidderURL:  BIDDER_URL,
	}
	payload, _ := json.Marshal(id)
	resp, err := http.Post("http://"+auctioneerURL+":"+auctioneerPort+"/registration", "application/json", bytes.NewBuffer(payload))
	if err != nil {
		fmt.Printf("Sending request to auctioneer failed %s\n", err.Error())
		return err
	}
	body, err := ioutil.ReadAll(resp.Body)
	defer resp.Body.Close()
	if err != nil {
		fmt.Printf("Error in reading body : %s\n", err.Error())
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

func getBidderID() (string, error) {
	/*uid, err := uuid.NewRandom()
	if err != nil {
		fmt.Printf("Error in creating bidder id : %s", err.Error())
		return "", err
	}
	return uid.String(), nil
	*/
	id := os.Getenv("ID")
	if id == "" {
		return id, fmt.Errorf("ENV Variable ID is not define")
	}
	return id, nil
}
func getBidderPort() (int, error) {
	port := os.Getenv("PORT")
	return strconv.Atoi(port)
}
func getAuctioneerPort() (int, error) {
	port := os.Getenv("AUCTIONEER_PORT")
	if port == "" {
		return -1, fmt.Errorf("Environment DELAY is not set ")
	}
	return strconv.Atoi(port)
}
func getBidValue() (float64, error) {
	value := os.Getenv("VALUE")
	if value == "" {
		return -1, fmt.Errorf("Environment variable VALUE is not set ")
	}
	return strconv.ParseFloat(value, 64)
}
func getTimeDelay() (int, error) {
	delay := os.Getenv("DELAY")
	if delay == "" {
		return -1, fmt.Errorf("Environment variable DELAY is not set ")
	}
	delayInt, err := strconv.Atoi(delay)
	if err != nil {
		return -1, fmt.Errorf("Valeu of DELAY variable in not int")
	}
	return delayInt, nil
}
func getAuctioneerURL() (string, error) {
	url := os.Getenv("AUCTIONEER_URL")
	if url == "" {
		return url, fmt.Errorf("Environment variable AUCTIONEER_URL is not set ")
	}
	return url, nil
}
func getBidderURL() (string, error) {
	url := os.Getenv("HOSTNAME")
	if url == "" {
		return url, fmt.Errorf("Environment variale HOSTNAME is not defined")
	}
	return url, nil
}
