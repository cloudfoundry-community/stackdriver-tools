/*
Copyright 2014 The Kubernetes Authors.
Copyright 2017 Google Inc.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package main

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"strings"
	"time"

	"cloud.google.com/go/datastore"
	"cloud.google.com/go/errors"
	"github.com/codegangsta/negroni"
	"github.com/gorilla/mux"
)

type Entry struct {
	Value   string    `datastore:"value"`
	Created time.Time `datastore:"created"`
}

var (
	datastoreClient *datastore.Client
	projectID       string
)

func ListRangeHandler(rw http.ResponseWriter, req *http.Request) {
	key := mux.Vars(req)["key"]

	var entries []*Entry

	query := datastore.NewQuery(key).Order("created")
	_, err := datastoreClient.GetAll(context.Background(), query, &entries)
	if err != nil {
		panic(err)
	}

	membersJSON := PanicOnError(json.MarshalIndent(entries, "", "  ")).([]byte)
	PanicOnError(rw.Write(membersJSON))
}

func ListPushHandler(rw http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	// Bug:
	key := vars["k"]
	value := vars["v"]
	// Functional code:
	// key := vars["key"]
	// value := vars["value"]

	if key == "" || value == "" {
		panic("error: can not store empty values")
	}

	newEntry := &Entry{
		Value:   value,
		Created: time.Now(),
	}

	datastoreKey := datastore.IncompleteKey(key, nil)
	_, err := datastoreClient.Put(context.Background(), datastoreKey, newEntry)
	if err != nil {
		panic(err)
	}

	ListRangeHandler(rw, req)
}

func InfoHandler(rw http.ResponseWriter, req *http.Request) {
}

func EnvHandler(rw http.ResponseWriter, req *http.Request) {
	environment := make(map[string]string)
	for _, item := range os.Environ() {
		splits := strings.Split(item, "=")
		key := splits[0]
		val := strings.Join(splits[1:], "=")
		environment[key] = val
	}

	var output []byte
	output = append(output, []byte("Environment:\n")...)
	envJSON := PanicOnError(json.MarshalIndent(environment, "", "  ")).([]byte)
	output = append(output, envJSON...)
	output = append(output, []byte("\nRequest:\n")...)
	reqJSON := PanicOnError(json.MarshalIndent(req.Header, "", "  ")).([]byte)
	output = append(output, reqJSON...)
	PanicOnError(rw.Write(output))
}

func PanicOnError(result interface{}, err error) (r interface{}) {
	if err != nil {
		panic(err)
	}
	return result
}

var errorsClient *errors.Client

type ErrorsMiddleware struct{}

func (l *ErrorsMiddleware) ServeHTTP(w http.ResponseWriter, req *http.Request, next http.HandlerFunc) {
	defer errorsClient.Catch(context.Background(), errors.WithRequest(req))
	if next != nil {
		next(w, req)
	}
}

func main() {
	projectID = os.Getenv("GOOGLE_PROJECT")
	if projectID == "" {
		panic("Please set the env variable GOOGLE_PROJECT in manifest.yml to a project with Google Cloud Datastore and Stackdriver Error Reporting enabled")
	}

	ctx := context.Background()
	errorsClient = PanicOnError(errors.NewClient(ctx, projectID, "cf-stackdriver-example", "0.0.1", true)).(*errors.Client)

	var err error
	datastoreClient, err = datastore.NewClient(context.Background(), projectID)
	if err != nil {
		panic("error connecting to the Google Cloud Datastore. Does this project have an App Engine App? see: https://cloud.google.com/datastore/docs/activate")
	}

	r := mux.NewRouter()
	r.Path("/lrange/{key}").Methods("GET").HandlerFunc(ListRangeHandler)
	r.Path("/rpush/{key}/{value}").Methods("GET").HandlerFunc(ListPushHandler)
	r.Path("/info").Methods("GET").HandlerFunc(InfoHandler)
	r.Path("/env").Methods("GET").HandlerFunc(EnvHandler)

	n := negroni.Classic()
	n.Use(&ErrorsMiddleware{})
	n.UseHandler(r)
	port := os.Getenv("PORT")
	if port == "" {
		port = "3000"
	}

	if err := http.ListenAndServe(":"+port, n); err != nil {
		panic(err)
	}
}
