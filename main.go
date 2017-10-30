package main

import (
	"database/sql"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"time"

	_ "github.com/lib/pq"
)

var (
	db  *sql.DB
	err error
)

func init() {
	db, err = sql.Open("postgres", os.Getenv("DATABASE_URL"))
	if err != nil {
		log.Fatalf("Error opening database: %q", err)
	}

	if _, err := db.Exec("CREATE TABLE IF NOT EXISTS messages (id varchar)"); err != nil {
		log.Fatalf("Error creating table: %q", err)
	} else {
		fmt.Println("Table initialited")
	}
}

// Logger
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
	if err != nil {
		log.Fatalf("Error reading body: %q", err)
	}

	fmt.Printf("body %q\n", string(body))

	w.Header().Add("content-type", "text/html")
	w.Write([]byte("<h2>hello world</h2>"))
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
