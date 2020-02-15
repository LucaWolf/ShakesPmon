package main

import (
	"encoding/json"
	"net/http"
)

// N.B json marshalling requires upper case identifiers. Use fields' tag to set the json element name
type apiStatus struct {
	Ok     bool   `json:"success"`
	Reason string `json:"text"`
}

type apiReply struct {
	Name   string    `json:"name"`
	Desc   string    `json:"description"`
	Result apiStatus `json:"result"`
}

//  interface for converting from name -> translated attribute
type name2desc interface {
	describe(name string) (string, error)
	translate(text string) (string, error)
}

func getDescription(name string, d name2desc) (string, error) {
	return d.describe(name)
}
func getTranslation(text string, d name2desc) (string, error) {
	return d.translate(text)
}

//-------------

// Pokemon to Shakespeare converter implementing name2desc
type webShakesPmon struct {
	urlPokeAPI     string
	urlShakespeare string
}

// stub:
func (r webShakesPmon) describe(name string) (string, error) {
	// GET method from uri and parse json reply into string
	// or set the error on failure

	return "This is a noddy description of: " + name, nil
}

// stub: GET method from uri and parse json reply into string
func (r webShakesPmon) translate(text string) (string, error) {

	return "This is a noddy translation of: " + text, nil
}

func newAPIHandler(d name2desc) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		name := r.URL.Path[1:] // todo: validate no sub-paths
		var resp apiReply
		var description, translation string
		var err error

		if description, err = getDescription(name, d); err != nil {
			resp = apiReply{name, "", apiStatus{false, err.Error()}}
		} else if translation, err = getTranslation(description, d); err != nil {
			resp = apiReply{name, "", apiStatus{true, err.Error()}}
		} else {
			resp = apiReply{name, translation, apiStatus{true, "Conversion completed."}}
		}

		e := json.NewEncoder(w)
		e.Encode(resp)

	})
}

func main() {
	proxy := webShakesPmon{"https://pokeapi.co/api/v2/pokemon-species/",
		"https://api.funtranslations.com/translate/shakespeare.json"}

	http.HandleFunc("/", newAPIHandler(proxy))
	http.ListenAndServe(":5000", nil)
}
