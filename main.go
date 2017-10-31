package main

import (
	"database/sql"
	"encoding/xml"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"time"

	_ "github.com/lib/pq"
)

type Notification struct {
	ID          string `xml:"Id"`
	SObjectID   string `xml:"sObject>Id"`
	SObjectName string `xml:"sObject>Name"`
}
type Result struct {
	Notifications []Notification `xml:"Body>notifications>Notification"`
}

var (
	db  *sql.DB
	err error
	ack = `<?xml version="1.0" encoding="UTF-8"?>
			<soap:Envelope xmlns:soap="http://schemas.xmlsoap.org/soap/envelope/">
			<soap:Body>
				<notificationsResponse xmlns:ns2="urn:sobject.enterprise.soap.sforce.com" xmlns="http://soap.sforce.com/2005/09/outbound">
				<Ack>true</Ack>
				</notificationsResponse>
			</soap:Body>
		</soap:Envelope>`

	nack = `<?xml version="1.0" encoding="UTF-8"?>
    <soapenv:Envelope xmlns:soapenv="http://schemas.xmlsoap.org/soap/envelope/" xmlns:xsd="http://www.w3.org/2001/XMLSchema" xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance">
      <soapenv:Body>
        <soapenv:Fault>
          <faultcode>soap:Receiver</faultcode>
          <faultstring>%s</faultstring>
        </soapenv:Fault>
      </soapenv:Body>
    </soapenv:Envelope>`
)

func init() {
	db, err = sql.Open("postgres", os.Getenv("DATABASE_URL"))
	//db, err = sql.Open("postgres", "user=jsuarez password=welcome1 dbname=jsuarez sslmode=disable")
	if err != nil {
		log.Fatalf("Error opening database: %q", err)
	}

	if _, err := db.Exec("CREATE TABLE IF NOT EXISTS messages (sfid varchar, account_name varchar)"); err != nil {
		log.Fatalf("Error creating table: %q", err)
	} else {
		fmt.Println("Table initialited")
	}
}

func Logger(inner http.Handler, name string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		inner.ServeHTTP(w, r)

		log.Printf(
			"%s\t%s\t%s\t%s",
			r.Method,
			r.RequestURI,
			name,
			time.Since(start),
		)
	})
}

var myHandler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
	body, err := ioutil.ReadAll(r.Body)
	resp := ack
	if err != nil {
		log.Fatalf("Error reading body: %q", err)
		resp = fmt.Sprintf(nack, "Error reading body")
	}

	fmt.Printf("body %q\n", string(body))

	v := &Result{}

	err = xml.Unmarshal(body, v)

	for _, val := range v.Notifications {

		stmt, err := db.Prepare(fmt.Sprintf("insert into messages values('%v','%v')", val.SObjectID, val.SObjectName))
		if err != nil {
			log.Fatalf("Error inserting message: %q", err)
			resp = fmt.Sprintf(nack, "Error inserting message")
		}

		defer stmt.Close()

		_, err = stmt.Exec()
		if err != nil {
			log.Fatalf("Error inserting message: %q", err)
			resp = fmt.Sprintf(nack, "Error inserting message")
			break
		}
	}

	w.Header().Add("content-type", "text/xml")
	w.Write([]byte(resp))
})

func main() {

	port := fmt.Sprintf(":%v", os.Getenv("PORT"))

	s := &http.Server{
		Addr:           port,
		Handler:        Logger(myHandler, "myHandler"),
		ReadTimeout:    10 * time.Second,
		WriteTimeout:   10 * time.Second,
		MaxHeaderBytes: 1 << 20,
	}

	fmt.Printf("Server started and listening in the port: %v\n", port)

	log.Fatal(s.ListenAndServe())

}
