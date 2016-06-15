package compton

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/fatih/structs"
	"gopkg.in/mgo.v2"
	"log"
	"net/http"
	"time"
)

type DateRate struct {
	Year  int    `bson:"year"`
	Month int    `bson:"month"`
	Day   int    `bson:"day"`
	Base  string `bson:"base"`
	Rates Rates  `bson:"rates"`
}

type Rates struct {
	CHF float64 `bson:"CHF"`
	EUR float64 `bson:"EUR"`
	USD float64 `bson:"USD"`
	// AED float64 `bson:"AED"`
	// AFN float64 `bson:"AFN"`
	// ALL float64 `bson:"ALL"`
	// AMD float64 `bson:"AMD"`
	// ANG float64 `bson:"ANG"`
	// AOA float64 `bson:"AOA"`
	// ARS float64 `bson:"ARS"`
	// AUD float64 `bson:"AUD"`
	// AWG float64 `bson:"AWG"`
	// AZN float64 `bson:"AZN"`
	// BAM float64 `bson:"BAM"`
	// BBD float64 `bson:"BBD"`
	// BDT float64 `bson:"BDT"`
	// BGN float64 `bson:"BGN"`
	// BHD float64 `bson:"BHD"`
	// BIF float64 `bson:"BIF"`
	// BMD float64 `bson:"BMD"`
	// BND float64 `bson:"BND"`
	// BOB float64 `bson:"BOB"`
	// BRL float64 `bson:"BRL"`
	// BSD float64 `bson:"BSD"`
	// BTC float64 `bson:"BTC"`
	// BTN float64 `bson:"BTN"`
	// BWP float64 `bson:"BWP"`
	// BYR float64 `bson:"BYR"`
	// BZD float64 `bson:"BZD"`
	// CAD float64 `bson:"CAD"`
	// CDF float64 `bson:"CDF"`
	// CLF float64 `bson:"CLF"`
	// CLP float64 `bson:"CLP"`
	// CNY float64 `bson:"CNY"`
	// COP float64 `bson:"COP"`
	// CRC float64 `bson:"CRC"`
	// CUC float64 `bson:"CUC"`
	// CUP float64 `bson:"CUP"`
	// CVE float64 `bson:"CVE"`
	// CZK float64 `bson:"CZK"`
	// DJF float64 `bson:"DJF"`
	// DKK float64 `bson:"DKK"`
	// DOP float64 `bson:"DOP"`
	// DZD float64 `bson:"DZD"`
	// EEK float64 `bson:"EEK"`
	// EGP float64 `bson:"EGP"`
	// ERN float64 `bson:"ERN"`
	// ETB float64 `bson:"ETB"`
	// FJD float64 `bson:"FJD"`
	// FKP float64 `bson:"FKP"`
	// GBP float64 `bson:"GBP"`
	// GEL float64 `bson:"GEL"`
	// GGP float64 `bson:"GGP"`
	// GHS float64 `bson:"GHS"`
	// GIP float64 `bson:"GIP"`
	// GMD float64 `bson:"GMD"`
	// GNF float64 `bson:"GNF"`
	// GTQ float64 `bson:"GTQ"`
	// GYD float64 `bson:"GYD"`
	// HKD float64 `bson:"HKD"`
	// HNL float64 `bson:"HNL"`
	// HRK float64 `bson:"HRK"`
	// HTG float64 `bson:"HTG"`
	// HUF float64 `bson:"HUF"`
	// IDR float64 `bson:"IDR"`
	// ILS float64 `bson:"ILS"`
	// IMP float64 `bson:"IMP"`
	// INR float64 `bson:"INR"`
	// IQD float64 `bson:"IQD"`
	// IRR float64 `bson:"IRR"`
	// ISK float64 `bson:"ISK"`
	// JEP float64 `bson:"JEP"`
	// JMD float64 `bson:"JMD"`
	// JOD float64 `bson:"JOD"`
	// JPY float64 `bson:"JPY"`
	// KES float64 `bson:"KES"`
	// KGS float64 `bson:"KGS"`
	// KHR float64 `bson:"KHR"`
	// KMF float64 `bson:"KMF"`
	// KPW float64 `bson:"KPW"`
	// KRW float64 `bson:"KRW"`
	// KWD float64 `bson:"KWD"`
	// KYD float64 `bson:"KYD"`
	// KZT float64 `bson:"KZT"`
	// LAK float64 `bson:"LAK"`
	// LBP float64 `bson:"LBP"`
	// LKR float64 `bson:"LKR"`
	// LRD float64 `bson:"LRD"`
	// LSL float64 `bson:"LSL"`
	// LTL float64 `bson:"LTL"`
	// LVL float64 `bson:"LVL"`
	// LYD float64 `bson:"LYD"`
	// MAD float64 `bson:"MAD"`
	// MDL float64 `bson:"MDL"`
	// MGA float64 `bson:"MGA"`
	// MKD float64 `bson:"MKD"`
	// MMK float64 `bson:"MMK"`
	// MNT float64 `bson:"MNT"`
	// MOP float64 `bson:"MOP"`
	// MRO float64 `bson:"MRO"`
	// MTL float64 `bson:"MTL"`
	// MUR float64 `bson:"MUR"`
	// MVR float64 `bson:"MVR"`
	// MWK float64 `bson:"MWK"`
	// MXN float64 `bson:"MXN"`
	// MYR float64 `bson:"MYR"`
	// MZN float64 `bson:"MZN"`
	// NAD float64 `bson:"NAD"`
	// NGN float64 `bson:"NGN"`
	// NIO float64 `bson:"NIO"`
	// NOK float64 `bson:"NOK"`
	// NPR float64 `bson:"NPR"`
	// NZD float64 `bson:"NZD"`
	// OMR float64 `bson:"OMR"`
	// PAB float64 `bson:"PAB"`
	// PEN float64 `bson:"PEN"`
	// PGK float64 `bson:"PGK"`
	// PHP float64 `bson:"PHP"`
	// PKR float64 `bson:"PKR"`
	// PLN float64 `bson:"PLN"`
	// PYG float64 `bson:"PYG"`
	// QAR float64 `bson:"QAR"`
	// RON float64 `bson:"RON"`
	// RSD float64 `bson:"RSD"`
	// RUB float64 `bson:"RUB"`
	// RWF float64 `bson:"RWF"`
	// SAR float64 `bson:"SAR"`
	// SBD float64 `bson:"SBD"`
	// SCR float64 `bson:"SCR"`
	// SDG float64 `bson:"SDG"`
	// SEK float64 `bson:"SEK"`
	// SGD float64 `bson:"SGD"`
	// SHP float64 `bson:"SHP"`
	// SLL float64 `bson:"SLL"`
	// SOS float64 `bson:"SOS"`
	// SRD float64 `bson:"SRD"`
	// STD float64 `bson:"STD"`
	// SVC float64 `bson:"SVC"`
	// SYP float64 `bson:"SYP"`
	// SZL float64 `bson:"SZL"`
	// THB float64 `bson:"THB"`
	// TJS float64 `bson:"TJS"`
	// TMT float64 `bson:"TMT"`
	// TND float64 `bson:"TND"`
	// TOP float64 `bson:"TOP"`
	// TRY float64 `bson:"TRY"`
	// TTD float64 `bson:"TTD"`
	// TWD float64 `bson:"TWD"`
	// TZS float64 `bson:"TZS"`
	// UAH float64 `bson:"UAH"`
	// UGX float64 `bson:"UGX"`
	// UYU float64 `bson:"UYU"`
	// UZS float64 `bson:"UZS"`
	// VEF float64 `bson:"VEF"`
	// VND float64 `bson:"VND"`
	// VUV float64 `bson:"VUV"`
	// WST float64 `bson:"WST"`
	// XAF float64 `bson:"XAF"`
	// XAG float64 `bson:"XAG"`
	// XAU float64 `bson:"XAU"`
	// XCD float64 `bson:"XCD"`
	// XDR float64 `bson:"XDR"`
	// XOF float64 `bson:"XOF"`
	// XPD float64 `bson:"XPD"`
	// XPF float64 `bson:"XPF"`
	// XPT float64 `bson:"XPT"`
	// YER float64 `bson:"YER"`
	// ZAR float64 `bson:"ZAR"`
	// ZMK float64 `bson:"ZMK"`
	// ZMW float64 `bson:"ZMW"`
	// ZWL float64 `bson:"ZWL"`
}

func getJSON(url string, target interface{}) error {
	r, err := http.Get(url)
	if err != nil {
		return err
	}
	defer r.Body.Close()

	return json.NewDecoder(r.Body).Decode(target)
}

func currencyAtDate(date time.Time) (rs DateRate) {
	url := fmt.Sprintf("https://openexchangerates.org/api/historical/%04d-%02d-%02d.json?app_id=7b42eb7fbc644d4ba1c3a885fd4a23ae",
		date.Year(), date.Month(), date.Day())
	err := getJSON(url, &rs)
	if err != nil {
		log.Fatal(err)
	}
	rs.Year = date.Year()
	rs.Month = int(date.Month())
	rs.Day = date.Day()
	return
}

func fetchCurrenciesAtDate(date time.Time, db *mgo.Database) (rs DateRate) {
	rs = currencyAtDate(time.Now())
	log.Printf("Retrieved the following rates: %+v", rs)

	mongoSession, err := mgo.Dial("localhost:27017")
	if err != nil {
		log.Fatal(err)
	}
	defer mongoSession.Close()
	err = db.C("currency").Insert(rs)
	if err != nil {
		log.Fatal(err)
	}
	log.Println("Successfully updated the currency database.")
	return
}

func convert(amount float64, from, to string, rates Rates) (newAmount float64, err error) {
	mapItUp := map[string]string{"€": "EUR", "$": "USD", "CHF": "CHF"}

	ratesMap := structs.Map(rates)
	f, ok := ratesMap[mapItUp[from]]
	if !ok {
		err = errors.New("Did not find currency " + from)
		return
	}
	t, ok := ratesMap[mapItUp[to]]
	if !ok {
		err = errors.New("Did not find currency " + to)
		return
	}
	rate := t.(float64) / f.(float64)
	newAmount = amount * rate
	return
}
