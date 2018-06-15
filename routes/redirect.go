package routes

import (
	"io"
	"fmt"
	"net/http"
	"net/url"
	"bufio"
)

func HandleRedirectGet(w http.ResponseWriter, r *http.Request, p Params) {
	u, _ := url.Parse(r.RequestURI)
	q := u.Query()
	urlRequestFromUser := q.Get("url")

	resp, err := http.Get(urlRequestFromUser)
	defer resp.Body.Close()
	if err != nil {
		w.WriteHeader(resp.StatusCode)
		fmt.Fprint(w, err)
	} else {
		reader := bufio.NewReader(resp.Body)
		byteData := make([]byte, 1024)
		for {
			n, err := reader.Read(byteData)
			if err != nil {
				if err == io.EOF {
					break
				} else {
					fmt.Println(err)
				}
				return
			}
			io.WriteString(w, string(byteData[:n]))
		}
	}
}