package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strings"
)

const (
	libraryVersion = "0.0.1"
	userAgent      = "govat/" + libraryVersion
	mediaType      = "application/json"
	format         = "json"
)

//VatRate VatRate type
type VatRate struct {
	Name        string `json:"name"`
	Code        string `json:"code"`
	CountryCode string `json:"country_code"`
	Periods     []struct {
		EffectiveFrom string `json:"effective_from"`
		Rates         struct {
			SuperReduced float64 `json:"super_reduced"`
			Reduced      float64 `json:"reduced"`
			Standard     float64 `json:"standard"`
		} `json:"rates"`
	} `json:"periods"`
}

//VatRates Base structure of API response
type VatRates struct {
	Details string      `json:"details"`
	Version interface{} `json:"version"`
	Rates   []VatRate   `json:"rates"`
}

//Client The client we use to communicate with the API
type Client struct {
	client  HttpClient
	BaseURL *url.URL
}
type VatClient interface {
	fetch(url string) ([]VatRate, error)
}

type HttpClient interface {
	Do(req *http.Request) (*http.Response, error)
}
type countriesCodes []string

//Config The configuration we want to use
type Config struct {
	baseURL   string
	UserAgent string
	Client    HttpClient
}

//NewClient Creates a new client with the configuration provided
func NewClient(config *Config) *Client {
	baseURL, _ := url.Parse(config.baseURL)
	if config.Client == nil {
		config.Client = http.DefaultClient
	}
	c := &Client{
		client:  config.Client,
		BaseURL: baseURL,
	}
	return c
}
func (client *Client) fetch(urlStr string) ([]VatRate, error) {
	var vatRates VatRates
	rel, err := url.Parse(urlStr)
	if err != nil {
		return nil, err
	}
	url := client.BaseURL.ResolveReference(rel)
	resolvedURL := url.String()
	req, _ := http.NewRequest("GET", resolvedURL, nil)
	if err := client.doRequest(req, &vatRates); err != nil {
		return nil, err
	}
	return vatRates.Rates, nil
}
func (client *Client) doRequest(req *http.Request, into interface{}) error {
	resp, err := client.client.Do(req)
	if err != nil {
		return err
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("non valid response. Code: %d", resp.StatusCode)
	}

	json.NewDecoder(resp.Body).Decode(into)
	defer resp.Body.Close()
	return err

}
func fetchCountriesVat(client VatClient, CountryCodes countriesCodes) ([]VatRate, error) {
	vatRates, err := client.fetch("/")
	if err != nil {
		return nil, err
	}
	var result []VatRate
	for _, vatRate := range vatRates {
		for i, countryCode := range CountryCodes {
			if vatRate.CountryCode == countryCode {
				result = append(result, vatRate)
				CountryCodes[len(CountryCodes)-1], CountryCodes[i] = CountryCodes[i], CountryCodes[len(CountryCodes)-1]
				CountryCodes = CountryCodes[:len(CountryCodes)-1]
				break
			}

		}
	}
	if len(result) == 0 {
		return nil, fmt.Errorf("countries with codes: %s not found", strings.Join(CountryCodes, ","))
	}
	return result, nil
}
func getUrl() string {
	var str string
	var ok bool
	if str, ok = os.LookupEnv("SERVER_URL"); !ok {
		return "https://jsonvat.com/"
	}
	return str
}

func (i *countriesCodes) String() string {
	return strings.Join(*i, ",")
}
func (i *countriesCodes) Set(value string) error {
	values := strings.Split(strings.ToUpper(value), ",")
	*i = countriesCodes(values)
	return nil
}
func getCountries(args []string) countriesCodes {
	var countries countriesCodes
	fs := flag.NewFlagSet("options", flag.ContinueOnError)
	fs.Var(&countries, "countries", "A list of comma separated countries")
	fs.Parse(args)
	return countries
}
func realmain() int {
	var vatClient *Client
	vatConfig := &Config{
		baseURL:   getUrl(),
		UserAgent: userAgent,
	}
	vatClient = NewClient(vatConfig)
	spRate, err := fetchCountriesVat(vatClient, getCountries(os.Args[1:]))
	if err != nil {
		fmt.Fprintf(os.Stderr, "%#v", err)
		return 1
	}
	var jsonStr []byte

	if len(spRate) == 1 {
		jsonStr, _ = json.MarshalIndent(spRate[0], "", "    ")
	} else {
		jsonStr, _ = json.MarshalIndent(spRate, "", "    ")

	}
	fmt.Fprintf(os.Stdout, string(jsonStr))
	return 0
}
func main() {
	os.Exit(realmain())
}
