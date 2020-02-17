package main

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// WaitForHostReply is how long to host comeback in seconds
const WaitForHostReply = 5

const TranslateErrNoText = 400
const TranslateErrAbuse = 429

// These are PokeAPI expected data
type pokeAPILanguage struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}
type pokeAPIFlavorText struct {
	Text     string          `json:"flavor_text"`
	Language pokeAPILanguage `json:"language"`
}

// we poll directly from "pokemon-species" interface
type pokeAPIResponse struct {
	FlavorTextEntries []pokeAPIFlavorText `json:"flavor_text_entries"`
	ID                int                 `json:"id"`
	CaptureRate       int                 `json:"capture_rate"`
	// etc...
}

// These are ::Shakespeare expected data
type translateStatus struct {
	Total int `json:"total"`
}
type translateContents struct {
	Translated string `json:"translated"`
	Original   string `json:"text"`
	Scheme     string `json:"translation"`
}
type translateAPIResponse struct {
	Status   translateStatus   `json:"success"`
	Contents translateContents `json:"contents"`
}

type translateErr struct {
	ID   int    `json:"code"`
	Text string `json:"message"`
}
type translateErrReply struct {
	Error translateErr `json:"error"`
}

//-------------------

// N.B json marshalling requires exportable fields
// Use fields' tag to set the json element name
type apiStatus struct {
	Ok     bool   `json:"success"`
	Reason string `json:"text"`
}

type apiReply struct {
	Name   string    `json:"name"`
	Desc   string    `json:"description"`
	Result apiStatus `json:"result"`
}

// apiErrors hides the low level errors into dedicates values for this API.
type apiError struct {
	ID   int
	Text string
}

var apiErrNone = apiError{0, "SUCCESS"}
var apiErrDescGet = apiError{100, "Cannot GET description"}
var apiErrDescRead = apiError{101, "Cannot read description"}
var apiErrDescJSON = apiError{102, "Invalid description or N/A"}
var apiErrDescLang = apiError{103, "Language not supported"}

var apiErrTransFetch = apiError{200, "Cannot fetch translation"}
var apiErrTransRead = apiError{201, "Cannot read translation"}
var apiErrTransJSON = apiError{202, "Invalid translation or N/A"}
var apiErrTransRequest = apiError{203, "Missing 'text' field"}
var apiErrTransOverload = apiError{204, "Too Many Requests"}

func (e apiError) Error() string {
	return e.Text
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

/**
Pulls the description of the 'name' character over the provided service point
Returns: the English description or one of errors:
  - apiErrDescGet: http failed
  - apiErrDescRead: failure to read http reply
  - apiErrDescJSON: the reply is not a valid JSON
  - apiErrDescLang: the JSON reply does not contain a description
*/
func (r webShakesPmon) describe(name string) (string, error) {
	// todo: validate no sub-paths

	client := &http.Client{
		Timeout: time.Second * WaitForHostReply,
	}

	res, err := client.Get(r.urlPokeAPI + name)
	if err != nil {
		return "", apiErrDescGet
	}

	var reply pokeAPIResponse
	js, err := ioutil.ReadAll(res.Body)
	res.Body.Close()
	if err != nil {
		return "", apiErrDescRead
	}

	if err = json.Unmarshal(js, &reply); err != nil {
		return "", apiErrDescJSON
	}

	for _, r := range reply.FlavorTextEntries {
		if r.Language.Name == "en" {
			return r.Text, nil
		}
	}

	return "", apiErrDescLang
}

/**
Pulls the Shakespearean translation of a text
Returns: text or one of errors:
  - apiErrTransFetch: http exchange failed
  - apiErrTransRead: failure to read http reply
  - apiErrTransJSON: the reply is not a valid JSON
  - apiErrTransRequest when the 'text' field is missing from the request
  - apiErrTransOverload too many requests on the translation service
*/
func (r webShakesPmon) translate(text string) (string, error) {

	client := &http.Client{
		Timeout: time.Second * WaitForHostReply,
	}

	params := url.Values{}
	params.Set("text", text)

	request, _ := http.NewRequest("POST", r.urlShakespeare, strings.NewReader(params.Encode()))
	request.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	request.Header.Add("Content-Length", strconv.Itoa(len(params.Encode())))

	res, err := client.Do(request)
	if err != nil {
		return "", apiErrTransFetch
	}

	var replyOK translateAPIResponse
	var replyErr translateErrReply

	js, err := ioutil.ReadAll(res.Body)
	res.Body.Close()
	if err != nil {
		return "", apiErrTransRead
	}

	err = json.Unmarshal(js, &replyOK)
	if err != nil || replyOK.Contents.Original == "" {
		err = json.Unmarshal(js, &replyErr)
		if err != nil || replyErr.Error.Text == "" {
			return "", apiErrTransJSON
		}

		switch replyErr.Error.ID {
		case TranslateErrNoText:
			return "", apiErrTransRequest
		case TranslateErrAbuse:
			return "", apiErrTransOverload
		default:
			return "", apiErrTransJSON
		}
	}

	return replyOK.Contents.Translated, nil
}

func newAPIHandler(d name2desc) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		name := r.URL.Path[1:]
		var resp apiReply
		var description, translation string
		var err error

		if description, err = getDescription(name, d); err != nil {
			resp = apiReply{name, "", apiStatus{false, err.Error()}}
		} else if translation, err = getTranslation(description, d); err != nil {
			resp = apiReply{name, "", apiStatus{false, err.Error()}}
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
