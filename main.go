package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"regexp"
	"strings"

	"unicode"

	"github.com/gorilla/mux"
	"github.com/joho/godotenv"
	"github.com/spf13/viper"
	"golang.org/x/text/transform"
	"golang.org/x/text/unicode/norm"
)

type ViaCEPResponse struct {
	CEP         string `json:"cep"`
	Logradouro  string `json:"logradouro"`
	Complemento string `json:"complemento"`
	Bairro      string `json:"bairro"`
	Localidade  string `json:"localidade"`
	UF          string `json:"uf"`
	IBGE        string `json:"ibge"`
	GIA         string `json:"gia"`
	DDD         string `json:"ddd"`
	SIAFI       string `json:"siafi"`
	Erro        bool   `json:"erro,omitempty"`
}

type WeatherAPIResponse struct {
	Location struct {
		Name           string  `json:"name"`
		Region         string  `json:"region"`
		Country        string  `json:"country"`
		Lat            float64 `json:"lat"`
		Lon            float64 `json:"lon"`
		TzID           string  `json:"tz_id"`
		LocaltimeEpoch int64   `json:"localtime_epoch"`
		Localtime      string  `json:"localtime"`
	} `json:"location"`
	Current struct {
		LastUpdatedEpoch int64   `json:"last_updated_epoch"`
		LastUpdated      string  `json:"last_updated"`
		TempC            float64 `json:"temp_c"`
		TempF            float64 `json:"temp_f"`
		IsDay            int     `json:"is_day"`
		Condition        struct {
			Text string `json:"text"`
			Icon string `json:"icon"`
			Code int    `json:"code"`
		} `json:"condition"`
	} `json:"current"`
}

type TemperatureResponse struct {
	TempC float64 `json:"temp_C"`
	TempF float64 `json:"temp_F"`
	TempK float64 `json:"temp_K"`
}

type ErrorResponse struct {
	Message string `json:"message"`
}

type CEPService struct {
	httpClient HTTPClient
}

type WeatherService struct {
	httpClient HTTPClient
	apiKey     string
}

type HTTPClient interface {
	Get(url string) (*http.Response, error)
}

func celsiusToFahrenheit(celsius float64) float64 {
	return celsius*1.8 + 32
}

func celsiusToKelvin(celsius float64) float64 {
	return celsius + 273
}

func isValidCEP(cep string) bool {
	cep = strings.ReplaceAll(cep, "-", "")
	cep = strings.ReplaceAll(cep, " ", "")
	match, _ := regexp.MatchString("^[0-9]{8}$", cep)
	return match
}

func normalizeCEP(cep string) string {
	cep = strings.ReplaceAll(cep, "-", "")
	cep = strings.ReplaceAll(cep, " ", "")
	return cep
}

func NewCEPService(client HTTPClient) *CEPService {
	return &CEPService{httpClient: client}
}

func (s *CEPService) GetCEPInfo(cep string) (*ViaCEPResponse, error) {
	url := fmt.Sprintf("https://viacep.com.br/ws/%s/json/", cep)
	resp, err := s.httpClient.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	var viaCEPResp ViaCEPResponse
	if err := json.NewDecoder(resp.Body).Decode(&viaCEPResp); err != nil {
		return nil, err
	}
	if viaCEPResp.Erro {
		return nil, fmt.Errorf("CEP not found")
	}
	return &viaCEPResp, nil
}

func NewWeatherService(client HTTPClient, apiKey string) *WeatherService {
	return &WeatherService{
		httpClient: client,
		apiKey:     apiKey,
	}
}

func removeAccents(s string) string {
	t := transform.Chain(norm.NFD, transform.RemoveFunc(isMn), norm.NFC)
	result, _, _ := transform.String(t, s)
	return result
}

func isMn(r rune) bool {
	return unicode.Is(unicode.Mn, r)
}

func (s *WeatherService) GetTemperature(city, state string) (*WeatherAPIResponse, error) {
	city = removeAccents(city)
	query := fmt.Sprintf("%s,%s,Brazil", city, state)
	url := fmt.Sprintf("https://api.weatherapi.com/v1/current.json?key=%s&q=%s&aqi=no", s.apiKey, query)
	resp, err := s.httpClient.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("weather API error: %d", resp.StatusCode)
	}
	var weatherResp WeatherAPIResponse
	if err := json.NewDecoder(resp.Body).Decode(&weatherResp); err != nil {
		return nil, err
	}
	return &weatherResp, nil
}

func (app *App) handleWeatherByCEP(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	cep := vars["cep"]
	if !isValidCEP(cep) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnprocessableEntity)
		json.NewEncoder(w).Encode(ErrorResponse{Message: "invalid zipcode"})
		return
	}
	normalizedCEP := normalizeCEP(cep)
	cepInfo, err := app.cepService.GetCEPInfo(normalizedCEP)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(ErrorResponse{Message: "can not find zipcode"})
		return
	}
	weatherInfo, err := app.weatherService.GetTemperature(cepInfo.Localidade, cepInfo.UF)
	if err != nil {
		log.Printf("Error getting weather info: %v", err)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(ErrorResponse{Message: "error getting weather information"})
		return
	}
	tempC := weatherInfo.Current.TempC
	tempF := celsiusToFahrenheit(tempC)
	tempK := celsiusToKelvin(tempC)
	response := TemperatureResponse{
		TempC: tempC,
		TempF: tempF,
		TempK: tempK,
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

type App struct {
	cepService     *CEPService
	weatherService *WeatherService
}

func NewApp(cepService *CEPService, weatherService *WeatherService) *App {
	return &App{
		cepService:     cepService,
		weatherService: weatherService,
	}
}

func (app *App) setupRoutes() http.Handler {
	r := mux.NewRouter()
	r.HandleFunc("/weather/{cep}", app.handleWeatherByCEP).Methods("GET")
	return r
}

func main() {
	godotenv.Load()
	viper.AutomaticEnv()
	weatherAPIKey := viper.GetString("WEATHER_API_KEY")
	if weatherAPIKey == "" {
		log.Fatal("WEATHER_API_KEY environment variable is required")
	}
	port := viper.GetString("PORT")
	if port == "" {
		port = "8080"
	}
	httpClient := &http.Client{}
	cepService := NewCEPService(httpClient)
	weatherService := NewWeatherService(httpClient, weatherAPIKey)
	app := NewApp(cepService, weatherService)
	router := app.setupRoutes()
	log.Printf("Server starting on port %s", port)
	log.Fatal(http.ListenAndServe(":"+port, router))
}
