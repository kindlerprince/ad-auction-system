package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strconv"
	"sync"
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
	BidderURL  string `json:"bidder_url,omitempty"`
}

type bidderDetails struct {
	Port string
	URL  string
}

type safeDB struct {
	mu     sync.RWMutex
	bidMap map[string]float64
}
type bidRequest struct {
	BidderID string  `json:"bidder_id,omitempty"`
	Value    float64 `json:"value,omitempty"`
}

type adRequest struct {
	AuctionID string `json:"auction_id,omitempty"`
}

var (
	bidderMap       map[string]bidderDetails
	bidList         []bidRequest
	counter         int
	AUCTIONEER_PORT string
)

func main() {
	fmt.Println("Ad Auction System")
	port, err := getAuctioneerPort()
	if err != nil {
		fmt.Printf("%s\n", err.Error())
		return
	}
	AUCTIONEER_PORT = strconv.Itoa(port)
	fmt.Printf("AUCTIONEER PORT  %s\n", AUCTIONEER_PORT)
	r := mux.NewRouter()
	r.HandleFunc("/adrequest", adRequestHandler).Methods(http.MethodPost)
	r.HandleFunc("/registration", bidderRegistrationHandler).Methods(http.MethodPost)
	r.HandleFunc("/bidderlist", bidderListHandler).Methods(http.MethodGet)
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
		successBidderList := bidRequestToBidders(ad.AuctionID)
		resBid := bidResult(successBidderList)
		counter++
		fmt.Printf("Winner for %d round : %+v\n", counter, resBid)
		response = customResponse{
			Message: "Bid Result",
			Data:    resBid,
		}
		writeSuccessMessage(w, r, response)
	}
}

func createDB() *safeDB {
	db := &safeDB{
		bidMap: make(map[string]float64),
	}
	return db
}

func (db *safeDB) get(key string) (float64, bool) {
	db.mu.RLock()
	defer db.mu.RUnlock()
	value, ok := db.bidMap[key]
	return value, ok
}

func (db *safeDB) set(key string, value float64) {
	db.mu.Lock()
	defer db.mu.Unlock()
	db.bidMap[key] = value
}

func worker(bidderDet bidder, payload []byte, db *safeDB, wg *sync.WaitGroup) {
	defer wg.Done()
	client := http.Client{
		Timeout: 200 * time.Millisecond,
	}
	request, err := http.NewRequest("POST", "http://"+bidderDet.BidderURL+":"+bidderDet.BidderPort+"/auction/"+bidderDet.BidderID, bytes.NewBuffer(payload))
	if err != nil {
		fmt.Printf("Error in creating bid request [bidder id : %s ] %s\n", bidderDet.BidderID, err.Error())
		return
	}
	request.Header.Set("Content-type", "application/json")
	resp, err := client.Do(request)
	if err != nil {
		fmt.Printf("Sending request failed to bidder[bidder id : %s ] %s\n", bidderDet.BidderID, err.Error())
		return
	}
	if resp.StatusCode != http.StatusOK {
		fmt.Printf("Unable to place pid :%d\n", resp.StatusCode)
		return
	}
	fmt.Printf("Bidder[%s] Received bid request successfully\n", bidderDet.BidderID)
	defer resp.Body.Close()
	if resp != nil && resp.Body != nil {
		bidResponse, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			fmt.Printf("Error in reading body : %s\n", err.Error())
			return
		}
		var bid bidRequest
		err = json.Unmarshal(bidResponse, &bid)
		if err != nil {
			fmt.Printf("Error in unmarshalling body : %s\n", err.Error())
			return
		}
		checkBidding(bid, db)
	} else {
		fmt.Println("Bid Response is nil")
	}

}

func bidRequestToBidders(auctionID string) map[string]float64 {
	var ad adRequest
	var wg sync.WaitGroup
	currentUsers := fetchRegisteredUser()
	db := createDB()
	for _, bidderDet := range currentUsers {
		ad.AuctionID = auctionID
		payload, _ := json.Marshal(ad)
		wg.Add(1)
		go worker(bidderDet, payload, db, &wg)
	}
	wg.Wait()
	//time.Sleep(200 * time.Millisecond)
	return db.bidMap
}
func fetchRegisteredUser() []bidder {
	var currBidder []bidder
	for id, details := range bidderMap {
		currBidder = append(currBidder, bidder{
			BidderID:   id,
			BidderURL:  details.URL,
			BidderPort: details.Port,
		})
	}
	return currBidder
}

func bidResult(bidderList map[string]float64) bidRequest {
	var finalBid bidRequest
	for bidderID, bidValue := range bidderList {
		if bidValue > finalBid.Value {
			finalBid.BidderID = bidderID
			finalBid.Value = bidValue
		}
	}
	return finalBid
}

func checkBidding(bid bidRequest, db *safeDB) {
	db.set(bid.BidderID, bid.Value)
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
		bidderRegistration(bidderReg)
		response.Message = "Registration Successful"
		fmt.Printf("Registration Success for bidder %s\n", bidderReg.BidderID)
		writeSuccessMessage(w, r, response)
	}
}
func bidderRegistration(bidderEntity bidder) {
	if bidderMap == nil {
		bidderMap = make(map[string]bidderDetails)
	}
	bidderMap[bidderEntity.BidderID] = bidderDetails{
		Port: bidderEntity.BidderPort,
		URL:  bidderEntity.BidderURL,
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
func getAuctioneerPort() (int, error) {
	port := os.Getenv("AUCTIONEER_PORT")
	if port == "" {
		return -1, fmt.Errorf("Environment variale AUCTIONEER_PORT is not defined")
	}
	return strconv.Atoi(port)
}
