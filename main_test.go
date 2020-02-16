package main

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func testDescribeHandler(w http.ResponseWriter, r *http.Request) {
	name := r.URL.Path[1:]

	// when character is missing the host replies with "not found"
	if name == "pokeZ" {
		w.Header().Set("Content-Type", "application/text")
		w.WriteHeader(200)
		fmt.Fprintf(w, "Not found.")
	} else if name == "pokeY" {
		w.WriteHeader(http.StatusNotFound)
	} else if name == "pokeX" {
		time.Sleep(WAIT_FOR_REPLY * time.Second) // client waits for max
	} else {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(200)
		elEN := FlavorText{someUniqueID(name), Language{5, "en"}}
		elFR := FlavorText{someUniqueID(name), Language{6, "fr"}}
		rsp := descReply{[]FlavorText{elEN, elFR}}
		js, _ := json.Marshal(rsp)
		w.Write(js)
	}
}

func someUniqueID(s string) string {
	t := sha256.Sum256([]byte(s))
	res := fmt.Sprintf("%x", t)
	return res
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
	}{
		{"Item valid", webShakesPmon{url, ""}, args{"pokeA"}, someUniqueID("pokeA"), false},
		{"Item missing", webShakesPmon{url, ""}, args{"pokeZ"}, "error", true},
		{"Server Silence", webShakesPmon{url, ""}, args{"pokeY"}, "error", true},
		{"Server Timeout", webShakesPmon{url, ""}, args{"pokeX"}, "error", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.r.describe(tt.args.name)
			if err != nil {
				if !tt.wantErr {
					t.Errorf("webShakesPmon.describe() error = %v, wantErr %v", err, tt.wantErr)
				} else {
					// TODO test for same error type
					t.Errorf("Received error = %v, expecting ... todo", err)
				}
				return
			}
			if got != tt.want {
				t.Errorf("webShakesPmon.describe() = %v, want %v", got, tt.want)
			}
		})
	}
}
