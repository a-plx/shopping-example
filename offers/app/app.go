// Copyright 2018 Google Inc. All rights reserved.
// Use of this source code is governed by the Apache 2.0
// license that can be found in the LICENSE file.

// This example is for creating a comparison shopping website with offers
// stored in a MySQL database on Google cloud. The example also uses a handler
// which can be used to update the tables using the latest offers in Google
// Merchant Center using the Google content API.
package main

import (
	"errors"
	"fmt"
	"log"
	"net/http"
	"offers"
	"os"
	"strconv"

	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"

	"google.golang.org/appengine"
)

var (
	// See template.go
	listTmpl          = parseTemplate("list.html")
	detailTmpl        = parseTemplate("detail.html")
	updateSuccessTmpl = parseTemplate("update.html")
)

const (
	merchantIDEnv = "MERCHANT_ID"
)

func main() {
	registerHandlers()
	appengine.Main()
}

func registerHandlers() {
	// Use gorilla/mux for rich routing.
	// See http://www.gorillatoolkit.org/pkg/mux
	r := mux.NewRouter()

	r.Handle("/", http.RedirectHandler("/offers", http.StatusFound))

	r.Methods("GET").Path("/offers").
		Handler(appHandler(listHandler))

	r.Methods("GET").Path("/search").
		Handler(appHandler(searchHandler))

	r.Methods("GET").Path("/offers/{offer_id}").
		Handler(appHandler(detailHandler))

	r.Methods("GET").Path("/tasks/update_db").
		Handler(appHandler(updateHandler))
	// Respond to App Engine and Compute Engine health checks.
	// Indicate the server is healthy.
	r.Methods("GET").Path("/_ah/health").HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte("ok"))
		})

	// [START request_logging]
	// Delegate all of the HTTP routing and serving to the gorilla/mux router.
	// Log all requests using the standard Apache format.
	http.Handle("/", handlers.CombinedLoggingHandler(os.Stderr, r))
	// [END request_logging]
}

// listHandler displays a list with summaries of offers in the database.
func listHandler(w http.ResponseWriter, r *http.Request) *appError {
	offers, err := offers.DB.ListOffers()
	if err != nil {
		fmt.Printf("there was an error querying offers: %v", err)
	}
	return listTmpl.Execute(w, r, offers)
}

// searchHandler displays a list based on the search query.
func searchHandler(w http.ResponseWriter, r *http.Request) *appError {
	queries, ok := r.URL.Query()["q"]
	if !ok {
		return appErrorf(errors.New("bad offer query"), "could not find offers")
	}
	offers, err := offers.DB.SearchOffers(queries[0])
	if err != nil {
		fmt.Printf("there was an error querying offers: %v", err)
	}
	return listTmpl.Execute(w, r, offers)
}

// offerFromRequest retrieves an offer from the database given a offer ID in the
// URL's path.
func offerFromRequest(r *http.Request) (*offers.Offer, error) {
	id := mux.Vars(r)["offer_id"]
	offer, err := offers.DB.GetOffer(id)
	if err != nil {
		return nil, fmt.Errorf("could not find offer: %v", err)
	}
	return offer, nil
}

// detailHandler displays the details of a given offer.
func detailHandler(w http.ResponseWriter, r *http.Request) *appError {
	offer, err := offerFromRequest(r)
	if err != nil {
		return appErrorf(err, "%v", err)
	}
	return detailTmpl.Execute(w, r, offer)
}

// updateHandler updates the sqlDB with the latest offers using the contentAPI.
func updateHandler(w http.ResponseWriter, r *http.Request) *appError {
	id, err := strconv.ParseInt(mustGetenv(merchantIDEnv), 10, 64)
	if err != nil {
		return appErrorf(err, "error while parsing merchant id")
	}
	// TODO(asheem): Set log file path.
	offers.RunUpdate(id, "")
	return updateSuccessTmpl.Execute(w, r, nil)
}

// http://blog.golang.org/error-handling-and-go
type appHandler func(http.ResponseWriter, *http.Request) *appError

type appError struct {
	Error   error
	Message string
	Code    int
}

func (fn appHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if e := fn(w, r); e != nil { // e is *appError, not os.Error.
		log.Printf("Handler error: status code: %d, message: %s, underlying err: %#v",
			e.Code, e.Message, e.Error)

		http.Error(w, e.Message, e.Code)
	}
}

func appErrorf(err error, format string, v ...interface{}) *appError {
	return &appError{
		Error:   err,
		Message: fmt.Sprintf(format, v...),
		Code:    500,
	}
}

func mustGetenv(k string) string {
	v := os.Getenv(k)
	if v == "" {
		log.Panicf("%s environment variable not set.", k)
	}
	return v
}
