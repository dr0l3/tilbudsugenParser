package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/mozillazg/request"

	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

type OfferRecord struct {
	Id             int       `form:"id" json:"id,omitempty"`
	Item           string    `form:"item" json:"item,omitempty"`
	Priceper       float32   `form:"priceper" json:"priceper,omitempty"`
	Unit           string    `form:"unit" json:"unit,omitempty"`
	Duration_start time.Time `form:"duration_start" json:"duration_start,omitempty"`
	Duration_end   time.Time `form:"duration_end" json:"duration_end,omitempty"`
	Brand          string    `form:"brand" json:"brand,omitempty"`
	Store          string    `form:"store" json:"store,omitempty"`
}

func (o OfferRecord) String() string {
	return fmt.Sprintf("{Item: %s, Priceper: %f, Unit: %s, Brand: %s, Store: %s}\n", o.Item, o.Priceper, o.Unit, o.Brand, o.Store)
}

func CustomParser(term string) []OfferRecord {
	defer timeTrack(time.Now(), "Customparsing")
	c := new(http.Client)
	req := request.NewRequest(c)
	resp, _ := req.Get("http://www.tilbudsugen.dk/ajax_getSearch.php?chains_string=alle&search_item=" + term + "&organic_get_val=off&key_hole_get_val=off&border_get_val=off&order_by=price_quantity_mult")
	defer resp.Body.Close()

	rawhtml, _ := resp.Text()
	htmlTokenizer := html.NewTokenizer(strings.NewReader(rawhtml))

	offers := []OfferRecord{}
	item := ""
	brand := ""
	store := ""
	dur_start := time.Now()
	dur_end := time.Now()
	priceper := float32(0.0)
	unit := ""

	prefix := ""
	tdnumber := 0
	for {
		tokenType := htmlTokenizer.Next()

		switch {
		case tokenType == html.ErrorToken:
			return offers

		case tokenType == html.StartTagToken:
			currentToken := htmlTokenizer.Token()
			switch {
			case currentToken.DataAtom == atom.Td:
				prefix += "  "
				tdnumber += 1
			case currentToken.DataAtom == atom.Tr:
				prefix += "  "
				tdnumber = 0
			}

		case tokenType == html.EndTagToken:
			t := htmlTokenizer.Token()
			if t.DataAtom == atom.Img {
			} else {
				prefixlengh := prefixLength(len(prefix) - 2)
				prefix = prefix[:prefixlengh]
			}

		default:
			t := htmlTokenizer.Token()
			switch {
			case tdnumber == 1:
				store = getstore(t.String())
			case tdnumber == 3:
				item = t.String()
			case tdnumber == 4:
				brand = t.String()
			case tdnumber == 7:
				priceper, unit = getPricePerUnit(t.String())
			case tdnumber == 9:
				dur_start, dur_end = getStartAndEnd(t.String())
				offers = append(offers, OfferRecord{Item: item, Brand: brand, Priceper: priceper, Unit: unit, Store: store, Duration_start: dur_start, Duration_end: dur_end})
			}

		}
	}
}

func getstore(rawinput string) string {
	rawinput = strings.ToLower(rawinput)
	if strings.Contains(rawinput, ("netto")) {
		return "Netto"
	}
	if strings.Contains(rawinput, ("foetex")) {
		return "Fotex"
	}
	if strings.Contains(rawinput, ("rema1000")) {
		return "Rema 1000"
	}
	if strings.Contains(rawinput, ("fakta")) {
		return "Fakta"
	}
	if strings.Contains(rawinput, ("lidl")) {
		return "Lidl"
	}
	if strings.Contains(rawinput, ("matas")) {
		return "Matas"
	}
	if strings.Contains(rawinput, ("superbrugsen")) {
		return "Super Brugsen"
	}
	if strings.Contains(rawinput, ("coop")) {
		return "Coop"
	}
	if strings.Contains(rawinput, ("bilka")) {
		return "Bilka"
	}
	if strings.Contains(rawinput, ("kvickly")) {
		return "Kvickly"
	}
	if strings.Contains(rawinput, ("dagli_brugsen")) {
		return "Daglig Brugsen"
	}
	if strings.Contains(rawinput, ("lokalbrugsen")) {
		return "Lokalbrugsen"
	}
	if strings.Contains(rawinput, ("kiwi")) {
		return "Kiwi"
	}
	if strings.Contains(rawinput, ("nemlig")) {
		return "Nemlig"
	}
	return ""
}

func getPricePerUnit(rawinput string) (float32, string) {
	postSplit := strings.Split(rawinput, "/")
	numberstring := postSplit[0]
	numberstring = strings.Replace(numberstring, ",", ".", -1)
	number, _ := strconv.ParseFloat(numberstring, 32)
	unit := ""
	if len(postSplit) > 1 {
		unit = postSplit[1]
	}
	return float32(number), unit
}

func getStartAndEnd(rawinput string) (time.Time, time.Time) {
	dates := strings.Split(rawinput, "-")
	currentyear := time.Now().Year()
	start_date, err := time.Parse("02/01", strings.Trim(dates[0], " "))
	if err != nil {
		fmt.Println("Error in dateparsing: " + err.Error())
	}
	start_date = start_date.AddDate(currentyear, 0, 0)

	end_date, err := time.Parse("02/01", strings.Trim(dates[1], " "))
	if err != nil {
		fmt.Println("Error in dateparsing: " + err.Error())
	}
	end_date = end_date.AddDate(currentyear, 0, 0)
	return start_date, end_date
}

func prefixLength(proposed int) int {
	if proposed > 0 {
		return proposed
	} else {
		return 0
	}
}

func timeTrack(start time.Time, name string) {
	elapsed := time.Since(start)
	log.Printf("%s took %s", name, elapsed)
}

func sendToApi(records []OfferRecord, address string) error {
	for _, singleRecord := range records {
		json, err := json.Marshal(singleRecord)
		fmt.Println("JSON:")
		fmt.Println(string(json))
		if err != nil {
			fmt.Println("Error during json marshalling: " + err.Error())
		}
		res, err := http.Post("http://"+address+":8080/insert", "application/json", bytes.NewReader(json))
		if err != nil {
			fmt.Println("Error during post request:" + err.Error())
		}
		fmt.Println(res)
		body, err := ioutil.ReadAll(res.Body)
		if err != nil {
			fmt.Println("Error during reading of body: " + err.Error())
		}
		fmt.Println(string(body))
	}
	return nil
}

func main() {
	apiAddress := os.Getenv("APIADDRESS")
	searchTermFilePath := os.Getenv("SEARCHTERMPATH")
	file, err := os.Open(searchTermFilePath)
	if err != nil {
		log.Fatal("Error during opening of file: " + err.Error())
		return
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	offers := []OfferRecord{}
	for scanner.Scan() {
		offers = append(offers, CustomParser(scanner.Text())...)
	}
	fmt.Println(offers)
	sendToApi(offers, apiAddress)
}
