package main

import (
	"fmt"
	"net/http"
	"strings"

	"SvcMan/services"
)

func main() {
	go services.Receptionist()

	http.HandleFunc("/", ServiceManager)
	http.HandleFunc("/stop", StopManager)
	http.ListenAndServe(":9500", nil)
}

func StopManager(w http.ResponseWriter, r *http.Request) {
	svc := r.URL.Query()["svc"][0]
	services.RequestQueue <- services.NewCommandMessage("stop:" + svc)
	fmt.Fprintf(w, "OK!")
}

func ServiceManager(w http.ResponseWriter, r *http.Request) {
	defer func() {
        if r := recover(); r != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("500 - Internal server error!"))
		}
    }()

	var parts = strings.Split(r.URL.Path, "/")
	var svc = parts[1]

	if svc == "" {
		fmt.Fprintf(w, "OK!")
		return
	}

	if svc == "robots.txt" || svc == "favicon.ico" {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte("404 - Not Found!"))
		return
	}

	rc := make(chan services.ResponseMessage)
	services.RequestQueue <- services.NewRequestMessage(svc, rc, r)
	rm := <-rc
	if rm.Err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("500 - Internal error!"))
	} else {
		r.URL.Path = strings.Join(remove(parts, 1), "/")
		rm.Service.ReverseProxy.ServeHTTP(w, r)
	}
}

func remove(slice []string, s int) []string {
	return append(slice[:s], slice[s+1:]...)
}
