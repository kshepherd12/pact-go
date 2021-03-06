package pact

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
	"testing"
)

var (
	validUsers    = map[int]*user{23: &user{23, "John", "Doe"}, 24: &user{24, "Jane", "Dame"}}
	mismatchUsers = map[int]*user{24: &user{24, "John", "Doe"}, 23: &user{23, "Jane", "Dame"}}
)

type user struct {
	Id        int    `json:"id"`
	FirstName string `json:"firstName"`
	LastName  string `json:"lastName"`
}

func userHandlerWithValidData(w http.ResponseWriter, r *http.Request) {
	userAction(w, r, validUsers)
}

func userHandlerWithMismatchedData(w http.ResponseWriter, r *http.Request) {
	userAction(w, r, mismatchUsers)
}

func userAction(w http.ResponseWriter, r *http.Request, users map[int]*user) {
	id, _ := strconv.Atoi(r.URL.Query().Get("id"))

	if users[id] == nil {
		http.Error(w, "", http.StatusNotFound)
	} else {
		w.Header().Set("Content-Type", "application/json")
		enc := json.NewEncoder(w)
		enc.Encode(users[id])
	}
}

func pactServer(w http.ResponseWriter, r *http.Request) {
	path := "./pact_examples/chrome_browser-go_api.json"
	b, err := ioutil.ReadFile(path)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
	}
	if _, err := w.Write(b); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
	}
	w.Header().Set("Content-Type", "application/json")
}

func Test_Verifier_CanVerify_Success(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/user", userHandlerWithValidData)
	mux.HandleFunc("/getpact", pactServer)
	server := httptest.NewServer(mux)

	defer server.Close()
	u, _ := url.Parse(server.URL)
	v := NewPactFileVerifier(nil, nil, nil).
		HonoursPactWith("chrome browser").
		PactUri(server.URL+"/getpact", nil).
		ServiceProvider("go api", &http.Client{}, u).
		ProviderState("there is a user with id {23}", nil, nil).
		ProviderState("there is no user with id {200}", nil, nil)
	if err := v.Verify(); err != nil {
		t.Error(err)
	}
}

func Test_Verifier_CanFilterInteractionsByState_Success(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/user", userHandlerWithValidData)
	mux.HandleFunc("/getpact", pactServer)
	server := httptest.NewServer(mux)

	defer server.Close()
	u, _ := url.Parse(server.URL)
	v := NewPactFileVerifier(nil, nil, nil).
		HonoursPactWith("chrome browser").
		PactUri(server.URL+"/getpact", nil).
		ServiceProvider("go api", &http.Client{}, u).
		ProviderState("there is a user with id {23}", nil, nil).
		ProviderState("there is no user with id {200}", nil, nil)
	if err := v.VerifyState("", "there is no user with id {200}"); err != nil {
		t.Error(err)
	}
	if err := v.Verify(); err != nil {
		t.Error(err)
	}
}

func Test_Verifier_CanFilterInteractionsByDescription_Success(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/user", userHandlerWithValidData)
	mux.HandleFunc("/getpact", pactServer)
	server := httptest.NewServer(mux)

	defer server.Close()
	u, _ := url.Parse(server.URL)
	v := NewPactFileVerifier(nil, nil, nil).
		HonoursPactWith("chrome browser").
		PactUri(server.URL+"/getpact", nil).
		ServiceProvider("go api", &http.Client{}, u).
		ProviderState("there is a user with id {23}", nil, nil).
		ProviderState("there is no user with id {200}", nil, nil)
	if err := v.VerifyState("get request for user with id {23}", ""); err != nil {
		t.Error(err)
	}
	if err := v.Verify(); err != nil {
		t.Error(err)
	}
}

func Test_Verifier_CanVerify_Mismatch(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/user", userHandlerWithMismatchedData)
	server := httptest.NewServer(mux)

	defer server.Close()
	u, _ := url.Parse(server.URL)
	v := NewPactFileVerifier(nil, nil, nil).
		HonoursPactWith("chrome browser").
		PactUri("./pact_examples/chrome_browser-go_api.json", nil).
		ServiceProvider("go api", &http.Client{}, u).
		ProviderState("there is a user with id {23}", nil, nil).
		ProviderState("there is no user with id {200}", nil, nil)
	if err := v.Verify(); err == nil {
		t.Error("Expected mismatch error")
	} else if err != errVerficationFailed {
		t.Error("expected verification failed error")
	}
}

func Test_Verifier_ThrowsError_InvalidPactUri(t *testing.T) {
	v := NewPactFileVerifier(nil, nil, nil).
		HonoursPactWith("consumer").
		ServiceProvider("provider", &http.Client{}, &url.URL{}).
		PactUri("badpath///", nil)
	if err := v.Verify(); err == nil {
		t.Error("Expected bad file error")
	} else if !strings.Contains(err.Error(), "badpath///") {
		t.Error("Expected bad file error")
	}
}

func Test_Verifier_ThrowsError_ConsumerNotSet(t *testing.T) {
	v := NewPactFileVerifier(nil, nil, nil)

	if err := v.Verify(); err == nil {
		t.Error("Expected empty conusmer name error")
	} else if err != errEmptyConsumer {
		t.Errorf("Expected %s, got %s", errEmptyConsumer, err)
	}
}

func Test_Verifier_ThrowsError_ProviderNotSet(t *testing.T) {
	v := NewPactFileVerifier(nil, nil, nil).
		HonoursPactWith("consumer")

	if err := v.Verify(); err == nil {
		t.Error("Expected empty provider name error")
	} else if err != errEmptyProvider {
		t.Errorf("Expected %s, got %s", errEmptyProvider, err)
	}
}

func Test_Verifier_ThrowsError_NoFilteredInteractions(t *testing.T) {
	v := NewPactFileVerifier(nil, nil, nil).
		HonoursPactWith("chrome browser").
		PactUri("./pact_examples/chrome_browser-go_api.json", nil).
		ServiceProvider("go api", &http.Client{}, &url.URL{}).
		ProviderState("there is a user with id {23}", nil, nil).
		ProviderState("there is no user with id {200}", nil, nil)

	if err := v.VerifyState("", "there is a user with id {0}"); err == nil {
		t.Error("Expected no filtered interactions found")
	} else if err != errNoFilteredInteractionsFound {
		t.Errorf("Expected %s, got %s", errNoFilteredInteractionsFound, err)
	}
}
