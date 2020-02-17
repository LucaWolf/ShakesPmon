package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

var stringVectors map[string]string = map[string]string{
	"cite1":  "To be, or not to be, that is the question...",
	"cite2":  "Bad Request: text field is missing",
	"cite3":  "Too Many Requests: Rate limit of 5 requests per hour exceeded",
	"Sparky": "Lively and high-spirited dragon.",
}

func testTranslateHandler(w http.ResponseWriter, r *http.Request) {

	// only testing for POST methods
	if err := r.ParseForm(); err != nil {
		return
	}

	value := r.FormValue("text")

	if value == "cite1" {
		rsp := translateAPIResponse{
			Status: translateStatus{1},
			Contents: translateContents{
				Translated: stringVectors[value],
				Original:   value,
				Scheme:     "shakespeare",
			},
		}
		js, _ := json.Marshal(rsp)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write(js)
	} else if value == "cite2" {
		rsp := translateErrReply{
			Error: translateErr{
				ID:   TranslateErrNoText,
				Text: stringVectors[value],
			},
		}
		js, _ := json.Marshal(rsp)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write(js)
	} else if value == "cite3" {
		rsp := translateErrReply{
			Error: translateErr{
				ID:   TranslateErrAbuse,
				Text: stringVectors[value],
			},
		}
		js, _ := json.Marshal(rsp)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write(js)
	} else if value == "cite4" {
		w.WriteHeader(http.StatusNotFound)
	} else if value == "cite5" {
		time.Sleep((WaitForHostReply + 1) * time.Second) // client waits for max WaitForHostReply
	}

}

func testDescribeHandler(w http.ResponseWriter, r *http.Request) {
	name := r.URL.Path[1:]

	// when character is missing the host replies with "not found"
	if name == "InvisibleMan" {
		w.Header().Set("Content-Type", "application/text")
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, "Not found.")
	} else if name == "Silentium" {
		w.WriteHeader(http.StatusNotFound)
	} else if name == "TimeShifter" {
		time.Sleep((WaitForHostReply + 1) * time.Second) // client waits for max WaitForHostReply
	} else if name == "FrèreJacques" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		elDE := pokeAPIFlavorText{"Guten Tag " + name, pokeAPILanguage{7, "de"}}
		elIT := pokeAPIFlavorText{"Buon giorno " + name, pokeAPILanguage{8, "it"}}
		elFR := pokeAPIFlavorText{"Bonjour " + name, pokeAPILanguage{6, "fr"}}
		rsp := pokeAPIResponse{FlavorTextEntries: []pokeAPIFlavorText{elFR, elDE, elIT}}
		js, _ := json.Marshal(rsp)
		w.Write(js)
	} else {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		elEN := pokeAPIFlavorText{stringVectors[name], pokeAPILanguage{5, "en"}}
		elFR := pokeAPIFlavorText{"C'est le " + name, pokeAPILanguage{6, "fr"}}
		rsp := pokeAPIResponse{FlavorTextEntries: []pokeAPIFlavorText{elFR, elEN}}
		js, _ := json.Marshal(rsp)
		w.Write(js)
	}
}

func Test_webShakesPmon_describe(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(testDescribeHandler))
	defer srv.Close()

	url := srv.URL + "/"

	type args struct {
		name string
	}
	tests := []struct {
		name    string
		r       webShakesPmon
		args    args
		want    string
		wantErr bool
		whatErr apiError
	}{
		{"Item valid", webShakesPmon{url, ""}, args{"Sparky"}, stringVectors["Sparky"], false, apiErrNone},
		{"Item missing", webShakesPmon{url, ""}, args{"InvisibleMan"}, "discard", true, apiErrDescJSON},
		{"Server Silence", webShakesPmon{url, ""}, args{"Silentium"}, "discard", true, apiErrDescJSON},
		{"Server Timeout", webShakesPmon{url, ""}, args{"TimeShifter"}, "discard", true, apiErrDescGet},
		{"Non English", webShakesPmon{url, ""}, args{"FrèreJacques"}, "discard", true, apiErrDescLang},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.r.describe(tt.args.name)
			if err != nil {
				if !tt.wantErr {
					t.Errorf("webShakesPmon.describe() error = %v, wantErr %v", err, tt.wantErr)
				} else if !errors.Is(err, tt.whatErr) {
					t.Errorf("Received error = %v, expecting %v", err, tt.whatErr)
				}
				return
			}
			if got != tt.want {
				t.Errorf("webShakesPmon.describe() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_webShakesPmon_translate(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(testTranslateHandler))
	defer srv.Close()

	type args struct {
		text string
	}
	tests := []struct {
		name    string
		r       webShakesPmon
		args    args
		want    string
		wantErr bool
		whatErr apiError
	}{
		{"Translate OK", webShakesPmon{"", srv.URL}, args{"cite1"}, stringVectors["cite1"], false, apiErrNone},
		{"Translate N/A", webShakesPmon{"", srv.URL}, args{"cite2"}, "discard", true, apiErrTransRequest},
		{"Translate Rate", webShakesPmon{"", srv.URL}, args{"cite3"}, "discard", true, apiErrTransOverload},
		{"Translate Silence", webShakesPmon{"", srv.URL}, args{"cite4"}, "discard", true, apiErrTransJSON},
		{"Translate Timeout", webShakesPmon{"", srv.URL}, args{"cite5"}, "discard", true, apiErrTransFetch},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.r.translate(tt.args.text)
			if err != nil {
				if !tt.wantErr {
					t.Errorf("webShakesPmon.translate() error = %v, wantErr %v", err, tt.wantErr)
				} else if !errors.Is(err, tt.whatErr) {
					t.Errorf("Translate error = %v, expecting %v", err, tt.whatErr)
				}
				return
			}
			if got != tt.want {
				t.Errorf("webShakesPmon.translate() = %v, want %v", got, tt.want)
			}
		})
	}
}
