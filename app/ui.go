package app

import (
	"html/template"
	"net/http"

	"github.com/gorilla/mux"
)

var index = template.Must(template.New("index").Parse(`

`))

func RouteUI(r *mux.Router) {
	r.HandleFunc("/", func(rw http.ResponseWriter, _ *http.Request) {
		index.Execute(rw, struct{}{})
	})
}
