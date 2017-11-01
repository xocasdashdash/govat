package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"reflect"
	"testing"
)

var (
	mux    *http.ServeMux
	client *Client
	server *httptest.Server
)

func setup() {
	mux = http.NewServeMux()
	server = httptest.NewServer(mux)
	testConfig := &Config{
		baseURL: server.URL,
	}
	client = NewClient(testConfig)

}
func teardown() {
	defer server.Close()
}
func testMethod(t *testing.T, r *http.Request, expected string) {
	if expected != r.Method {
		t.Errorf("Request method = %v, expected %v", r.Method, expected)
	}
}

func loadTestRates() []VatRate {
	var expected []VatRate
	response := []byte(`
	[{
		"name": "Spain",
		"code": "ES",
		"country_code": "ES",
		"periods": [
			{
				"effective_from": "0000-01-01",
				"rates": {
					"super_reduced": 4.0,
					"reduced": 10.0,
					"standard": 21.0
				}
			}
		]
	},
	{
		"name": "Bulgaria",
		"code": "BG",
		"country_code": "BG",
		"periods": [
			{
				"effective_from": "0000-01-01",
				"rates": {
					"reduced": 9.0,
					"standard": 20.0
				}
			}
		]
	}]`)
	json.Unmarshal(response, &expected)
	return expected

}
func TestVatGet(t *testing.T) {
	setup()
	defer teardown()
	response, _ := ioutil.ReadFile("test.json")
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, http.MethodGet)

		fmt.Fprint(w, string(response))
	})
	var expected VatRates
	json.Unmarshal(response, &expected)
	result, _ := client.fetch("/")
	if !reflect.DeepEqual(result, expected.Rates) {
		t.Errorf("Get failed. Returned: %+v, expected: %+v", result, expected.Rates)
	}
}

func TestVatGetHTTPError(t *testing.T) {
	setup()
	defer teardown()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "server broken :(", 500)
	})
	_, err := client.fetch("/")
	expected := errors.New("non valid response. Code: 500")
	if !reflect.DeepEqual(err, expected) {
		t.Errorf("Get didn't fail like it should. Returned: %+v, Expected: %+v", err, expected)
	}
}
func TestVatInvalidURL(t *testing.T) {
	setup()
	defer teardown()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "server broken :(", 500)
	})
	_, err := client.fetch(":1231231")

	expected := &url.Error{"parse", ":1231231", errors.New("missing protocol scheme")}
	if !reflect.DeepEqual(err, expected) {
		t.Errorf("Get didn't fail like it should. Returned: %+v, Expected: %+v", err.Error(), expected)
	}
}
func TestVatGetFailedConnection(t *testing.T) {
	testConfig := &Config{
		baseURL: "https://ThisURLDOESNOTEXISTS.COM.ES",
		Client: &HttpClientMock{
			Error: errors.New("failed connection"),
		},
	}
	client = NewClient(testConfig)
	_, err := client.fetch("/")
	expected := errors.New("failed connection")
	if !reflect.DeepEqual(err.Error(), expected.Error()) {
		t.Errorf("Get didn't fail like it should. Returned: *%#v*, Expected: *%#v*", err.Error(), expected.Error())
	}
}

type HttpClientMock struct {
	Error    error
	Response *http.Response
}

func (c *HttpClientMock) Do(req *http.Request) (*http.Response, error) {
	return c.Response, c.Error
}

type FakeVatClientER struct {
	CountriesVateRate []VatRate
	ExpectedErr       error
}

func (f FakeVatClientER) fetch(url string) ([]VatRate, error) {
	if f.ExpectedErr != nil {
		return nil, f.ExpectedErr
	}
	return f.CountriesVateRate, nil
}
func TestVatByCountry(t *testing.T) {
	fakeVatClient := FakeVatClientER{
		CountriesVateRate: loadTestRates(),
	}
	vatRates, err := fetchCountriesVat(fakeVatClient, []string{"ES"})
	if err != nil {
		t.Errorf("Err should not be nil")
	}
	if vatRates != nil && len(vatRates) == 1 && vatRates[0].CountryCode != "ES" {
		t.Errorf("Did not fetch the right country code. Expected %s, Received: %s", "ES", vatRates[0].CountryCode)
	}
}
func TestVatByCountryNotFound(t *testing.T) {
	fakeVatClient := FakeVatClientER{
		CountriesVateRate: loadTestRates(),
	}
	vatRates, err := fetchCountriesVat(fakeVatClient, []string{"NVM"})
	if vatRates != nil {
		t.Errorf("Neverland is not a valid country")
	}
	expectedErr := errors.New("countries with codes: NVM not found")
	if !reflect.DeepEqual(err, expectedErr) {
		t.Errorf("Expected err to be %q but it was %q", expectedErr, err)
	}

}
func TestVatTCPErrors(t *testing.T) {
	fakeVatClient := FakeVatClientER{
		ExpectedErr: errors.New("Failed connection"),
	}
	_, err := fetchCountriesVat(fakeVatClient, []string{"NVM"})
	if !reflect.DeepEqual(err, fakeVatClient.ExpectedErr) {
		t.Errorf("Expected err to be %q but it was %q", fakeVatClient.ExpectedErr, err)
	}
}
func TestMain(m *testing.M) {
	// os.Exit() does not respect defer statements
	ret := m.Run()

	os.Exit(ret)
}
func TestGetUrlOK(t *testing.T) {
	os.Setenv("SERVER_URL", "test")
	url := getUrl()
	if url != "test" {
		t.Errorf("Expected %+v. Got %+v", "test", url)
	}
	os.Unsetenv("SERVER_URL")
}
func TestGetUrlKO(t *testing.T) {
	url := getUrl()
	if url != "https://jsonvat.com/" {
		t.Errorf("Expected %+v. Got %+v", "https://jsonvat.com/", url)
	}
}
func TestRealMainOK(t *testing.T) {
	setup()
	defer teardown()
	response, _ := ioutil.ReadFile("test.json")
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, http.MethodGet)

		fmt.Fprint(w, string(response))
	})
	os.Setenv("SERVER_URL", server.URL)
	os.Args = []string{"govat", "-countries=es"}
	result := realmain()
	if result != 0 {
		t.Errorf("Failed during test run")
	}
	defer os.Unsetenv("SERVER_URL")
}

func TestRealMainOKMultipleCountries(t *testing.T) {
	setup()
	defer teardown()
	response, _ := ioutil.ReadFile("test.json")
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, http.MethodGet)

		fmt.Fprint(w, string(response))
	})
	os.Setenv("SERVER_URL", server.URL)
	os.Args = []string{"govat", "-countries=es,de"}
	result := realmain()
	if result != 0 {
		t.Errorf("Failed during test run")
	}
	defer os.Unsetenv("SERVER_URL")
}
func TestRealMainKO(t *testing.T) {
	os.Setenv("SERVER_URL", "FAKE")
	result := realmain()
	if result == 0 {
		t.Errorf("Expecting %d got %d", 1, 0)
	}
}
func TestGetCountries(t *testing.T) {
	testArgs := []string{
		"-countries=es,de,fr",
	}
	expected := countriesCodes{
		"ES", "DE", "FR",
	}
	result := getCountries(testArgs)
	if !reflect.DeepEqual(result, expected) {
		t.Errorf("Expected %v. Got: %v", expected, result)
	}

}
