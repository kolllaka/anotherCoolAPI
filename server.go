package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"math/rand"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/joho/godotenv"
)

type Coasters struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	InPark       string `json:"in_park"`
	Manufacturer string `json:"manufacturer"`
	Height       int    `json:"height"`
}

type coasters struct {
	sync.Mutex
	store map[string]Coasters `json:"-"`
}

type adminPortal struct {
	Password string
}

func main() {
	if err := godotenv.Load(); err != nil {
		log.Fatalf("error loading env variables: %s", err.Error())
	}

	admin := newAdminPortal()
	coasters := newCoasters()
	// coasters.store["1649252773968000000"] = Coasters{
	// 	Name:         "Fury 325",
	// 	Height:       99,
	// 	ID:           "id1",
	// 	InPark:       "Carowinds",
	// 	Manufacturer: "B+M",
	// }
	// coasters.store["1649252831362000000"] = Coasters{
	// 	Name:         "Taron",
	// 	Height:       30,
	// 	ID:           "id2",
	// 	InPark:       "Phantasialand",
	// 	Manufacturer: "Intamin",
	// }

	http.HandleFunc("/coasters", coasters.handler)
	http.HandleFunc("/coasters/", coasters.getCoaster)
	http.HandleFunc("/admin", admin.handler)

	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatal(err)
	}
}

func newAdminPortal() *adminPortal {
	password := os.Getenv("ADMIN_PASSWORD")
	if password == "" {
		panic("required .env ADMIN_PASSWORD not set")
	}

	return &adminPortal{
		Password: password,
	}
}

func (a *adminPortal) handler(w http.ResponseWriter, r *http.Request) {
	user, pass, ok := r.BasicAuth()
	if !ok || user != "admin" || pass != a.Password {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte("401 - Unauthorized"))
		return
	}

	w.Write([]byte("<html><h1>Super secret admin portal</h1></html>"))
}

func newCoasters() *coasters {
	return &coasters{
		store: map[string]Coasters{},
	}
}

func (c *coasters) handler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		c.get(w, r)
		return
	case "POST":
		c.post(w, r)
		return
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
		w.Write([]byte("Method Not Allowed"))
		return
	}
}

func (c *coasters) get(w http.ResponseWriter, r *http.Request) {
	coasters := make([]Coasters, len(c.store))

	c.Lock()
	i := 0
	for _, coaster := range c.store {
		coasters[i] = coaster
		i++
	}
	c.Unlock()

	jsonBytes, err := json.Marshal(coasters)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(err.Error()))
		return
	}

	w.Header().Add("content-type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(jsonBytes)
}

func (c *coasters) post(w http.ResponseWriter, r *http.Request) {
	bodyBytes, err := ioutil.ReadAll(r.Body)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(err.Error()))
		return
	}
	defer r.Body.Close()

	ct := r.Header.Get("content-type")
	if ct != "application/json" {
		w.WriteHeader(http.StatusUnsupportedMediaType)
		w.Write([]byte(fmt.Sprintf("need content-type = 'application/json', but got '%s'", ct)))
		return
	}

	var coaster Coasters
	if err := json.Unmarshal(bodyBytes, &coaster); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(err.Error()))
	}

	coaster.ID = fmt.Sprintf("%d", time.Now().UnixNano())

	c.Lock()
	c.store[coaster.ID] = coaster
	defer c.Unlock()
}

func (c *coasters) getCoaster(w http.ResponseWriter, r *http.Request) {
	parts := strings.Split(r.URL.Path, "/")
	if len(parts) != 3 {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	if parts[2] == "random" {
		c.getRandomCoaster(w, r)
		return
	}

	c.Lock()
	coaster, ok := c.store[parts[2]]
	c.Unlock()
	if !ok {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	jsonBytes, err := json.Marshal(coaster)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(err.Error()))
		return
	}

	w.Header().Add("content-type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(jsonBytes)
}

func (c *coasters) getRandomCoaster(w http.ResponseWriter, r *http.Request) {
	c.Lock()
	ids := make([]string, len(c.store))
	i := 0
	for id := range c.store {
		ids[i] = id
		i++
	}
	c.Unlock()

	var target string

	if len(ids) == 0 {
		w.WriteHeader(http.StatusNotFound)
		return
	} else if len(ids) == 1 {
		target = ids[0]
	} else {
		rand.Seed(time.Now().UnixNano())
		target = ids[rand.Intn(len(ids))]
	}

	w.Header().Add("location", fmt.Sprintf("/coasters/%s", target))
	w.WriteHeader(http.StatusFound)
}
