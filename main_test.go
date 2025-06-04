package main

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gorilla/mux"
)

type MockHTTPClient struct {
	responses map[string]*http.Response
	errors    map[string]error
}

func NewMockHTTPClient() *MockHTTPClient {
	return &MockHTTPClient{
		responses: make(map[string]*http.Response),
		errors:    make(map[string]error),
	}
}

func (m *MockHTTPClient) AddResponse(url string, statusCode int, body string) {
	m.responses[url] = &http.Response{
		StatusCode: statusCode,
		Body:       io.NopCloser(strings.NewReader(body)),
		Header:     make(http.Header),
	}
}

func (m *MockHTTPClient) AddError(url string, err error) {
	m.errors[url] = err
}

func (m *MockHTTPClient) Get(url string) (*http.Response, error) {
	if err, exists := m.errors[url]; exists {
		return nil, err
	}
	if resp, exists := m.responses[url]; exists {
		return resp, nil
	}
	return &http.Response{
		StatusCode: 404,
		Body:       io.NopCloser(strings.NewReader("")),
		Header:     make(http.Header),
	}, nil
}

func TestIsValidCEP(t *testing.T) {
	tests := []struct {
		name     string
		cep      string
		expected bool
	}{
		{"CEP válido - 8 dígitos", "12345678", true},
		{"CEP válido - com traço", "12345-678", true},
		{"CEP válido - com espaços", "123 456 78", true},
		{"CEP inválido - 7 dígitos", "1234567", false},
		{"CEP inválido - 9 dígitos", "123456789", false},
		{"CEP inválido - com letras", "1234567a", false},
		{"CEP inválido - vazio", "", false},
		{"CEP inválido - caracteres especiais", "12345@78", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isValidCEP(tt.cep)
			if result != tt.expected {
				t.Errorf("isValidCEP(%s) = %v, expected %v", tt.cep, result, tt.expected)
			}
		})
	}
}

func TestNormalizeCEP(t *testing.T) {
	tests := []struct {
		name     string
		cep      string
		expected string
	}{
		{"CEP com traço", "12345-678", "12345678"},
		{"CEP com espaços", "123 456 78", "12345678"},
		{"CEP normal", "12345678", "12345678"},
		{"CEP com múltiplos caracteres", "123-45 678", "12345678"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := normalizeCEP(tt.cep)
			if result != tt.expected {
				t.Errorf("normalizeCEP(%s) = %s, expected %s", tt.cep, result, tt.expected)
			}
		})
	}
}

func TestTemperatureConversions(t *testing.T) {
	tests := []struct {
		name      string
		celsius   float64
		expectedF float64
		expectedK float64
	}{
		{"Zero Celsius", 0.0, 32.0, 273.0},
		{"Temperatura ambiente", 25.0, 77.0, 298.0},
		{"Ponto de ebulição da água", 100.0, 212.0, 373.0},
		{"Temperatura negativa", -10.0, 14.0, 263.0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fahrenheit := celsiusToFahrenheit(tt.celsius)
			kelvin := celsiusToKelvin(tt.celsius)

			if fahrenheit != tt.expectedF {
				t.Errorf("celsiusToFahrenheit(%.1f) = %.1f, expected %.1f", tt.celsius, fahrenheit, tt.expectedF)
			}

			if kelvin != tt.expectedK {
				t.Errorf("celsiusToKelvin(%.1f) = %.1f, expected %.1f", tt.celsius, kelvin, tt.expectedK)
			}
		})
	}
}

func TestCEPService_GetCEPInfo(t *testing.T) {
	mockClient := NewMockHTTPClient()
	service := NewCEPService(mockClient)

	t.Run("CEP válido encontrado", func(t *testing.T) {
		cepResponse := `{
			"cep": "01310-100",
			"logradouro": "Avenida Paulista",
			"complemento": "",
			"bairro": "Bela Vista",
			"localidade": "São Paulo",
			"uf": "SP",
			"ibge": "3550308",
			"gia": "1004",
			"ddd": "11",
			"siafi": "7107"
		}`
		mockClient.AddResponse("https://viacep.com.br/ws/01310100/json/", 200, cepResponse)

		result, err := service.GetCEPInfo("01310100")

		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}

		if result.Localidade != "São Paulo" {
			t.Errorf("Expected localidade 'São Paulo', got '%s'", result.Localidade)
		}

		if result.UF != "SP" {
			t.Errorf("Expected UF 'SP', got '%s'", result.UF)
		}
	})

	t.Run("CEP não encontrado", func(t *testing.T) {
		cepResponse := `{"erro": true}`
		mockClient.AddResponse("https://viacep.com.br/ws/99999999/json/", 200, cepResponse)

		result, err := service.GetCEPInfo("99999999")

		if err == nil {
			t.Error("Expected error for non-existent CEP")
		}

		if result != nil {
			t.Error("Expected nil result for non-existent CEP")
		}
	})

	t.Run("Erro de conexão", func(t *testing.T) {
		mockClient.AddError("https://viacep.com.br/ws/12345678/json/", errors.New("connection error"))

		result, err := service.GetCEPInfo("12345678")

		if err == nil {
			t.Error("Expected connection error")
		}

		if result != nil {
			t.Error("Expected nil result on connection error")
		}
	})
}

func TestWeatherService_GetTemperature(t *testing.T) {
	mockClient := NewMockHTTPClient()
	service := NewWeatherService(mockClient, "test-api-key")

	t.Run("Consulta de temperatura bem-sucedida", func(t *testing.T) {
		weatherResponse := `{
			"location": {
				"name": "São Paulo",
				"region": "Sao Paulo",
				"country": "Brazil",
				"lat": -23.55,
				"lon": -46.64,
				"tz_id": "America/Sao_Paulo",
				"localtime_epoch": 1234567890,
				"localtime": "2023-01-01 12:00"
			},
			"current": {
				"last_updated_epoch": 1234567890,
				"last_updated": "2023-01-01 12:00",
				"temp_c": 25.0,
				"temp_f": 77.0,
				"is_day": 1,
				"condition": {
					"text": "Sunny",
					"icon": "//cdn.weatherapi.com/weather/64x64/day/113.png",
					"code": 1000
				}
			}
		}`
		expectedURL := "https://api.weatherapi.com/v1/current.json?key=test-api-key&q=São Paulo,SP,Brazil&aqi=no"
		mockClient.AddResponse(expectedURL, 200, weatherResponse)

		result, err := service.GetTemperature("São Paulo", "SP")

		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}

		if result.Current.TempC != 25.0 {
			t.Errorf("Expected temperature 25.0°C, got %.1f°C", result.Current.TempC)
		}

		if result.Location.Name != "São Paulo" {
			t.Errorf("Expected location 'São Paulo', got '%s'", result.Location.Name)
		}
	})

	t.Run("Erro da API do clima", func(t *testing.T) {
		expectedURL := "https://api.weatherapi.com/v1/current.json?key=test-api-key&q=Invalid City,XX,Brazil&aqi=no"
		mockClient.AddResponse(expectedURL, 400, `{"error": {"code": 1006, "message": "No matching location found."}}`)

		result, err := service.GetTemperature("Invalid City", "XX")

		if err == nil {
			t.Error("Expected error for invalid location")
		}

		if result != nil {
			t.Error("Expected nil result for invalid location")
		}
	})
}

func TestHandleWeatherByCEP(t *testing.T) {
	mockClient := NewMockHTTPClient()
	cepService := NewCEPService(mockClient)
	weatherService := NewWeatherService(mockClient, "test-api-key")
	app := NewApp(cepService, weatherService)

	t.Run("Consulta bem-sucedida", func(t *testing.T) {
		cepResponse := `{
			"cep": "01310-100",
			"logradouro": "Avenida Paulista",
			"complemento": "",
			"bairro": "Bela Vista",
			"localidade": "São Paulo",
			"uf": "SP",
			"ibge": "3550308",
			"gia": "1004",
			"ddd": "11",
			"siafi": "7107"
		}`
		mockClient.AddResponse("https://viacep.com.br/ws/01310100/json/", 200, cepResponse)

		weatherResponse := `{
			"location": {
				"name": "São Paulo",
				"region": "Sao Paulo",
				"country": "Brazil",
				"lat": -23.55,
				"lon": -46.64,
				"tz_id": "America/Sao_Paulo",
				"localtime_epoch": 1234567890,
				"localtime": "2023-01-01 12:00"
			},
			"current": {
				"last_updated_epoch": 1234567890,
				"last_updated": "2023-01-01 12:00",
				"temp_c": 25.0,
				"temp_f": 77.0,
				"is_day": 1,
				"condition": {
					"text": "Sunny",
					"icon": "//cdn.weatherapi.com/weather/64x64/day/113.png",
					"code": 1000
				}
			}
		}`
		weatherURL := "https://api.weatherapi.com/v1/current.json?key=test-api-key&q=São Paulo,SP,Brazil&aqi=no"
		mockClient.AddResponse(weatherURL, 200, weatherResponse)

		req, err := http.NewRequest("GET", "/weather/01310-100", nil)
		if err != nil {
			t.Fatal(err)
		}

		rr := httptest.NewRecorder()

		router := mux.NewRouter()
		router.HandleFunc("/weather/{cep}", app.handleWeatherByCEP).Methods("GET")

		router.ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusOK {
			t.Errorf("Handler returned wrong status code: got %v want %v", status, http.StatusOK)
		}

		var response TemperatureResponse
		err = json.Unmarshal(rr.Body.Bytes(), &response)
		if err != nil {
			t.Errorf("Error parsing response: %v", err)
		}

		if response.TempC != 25.0 {
			t.Errorf("Expected temp_C 25.0, got %.1f", response.TempC)
		}

		if response.TempF != 77.0 {
			t.Errorf("Expected temp_F 77.0, got %.1f", response.TempF)
		}

		if response.TempK != 298.0 {
			t.Errorf("Expected temp_K 298.0, got %.1f", response.TempK)
		}
	})

	t.Run("CEP inválido - formato incorreto", func(t *testing.T) {
		req, err := http.NewRequest("GET", "/weather/123", nil)
		if err != nil {
			t.Fatal(err)
		}

		rr := httptest.NewRecorder()
		router := mux.NewRouter()
		router.HandleFunc("/weather/{cep}", app.handleWeatherByCEP).Methods("GET")
		router.ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusUnprocessableEntity {
			t.Errorf("Handler returned wrong status code: got %v want %v", status, http.StatusUnprocessableEntity)
		}

		var response ErrorResponse
		err = json.Unmarshal(rr.Body.Bytes(), &response)
		if err != nil {
			t.Errorf("Error parsing response: %v", err)
		}

		if response.Message != "invalid zipcode" {
			t.Errorf("Expected message 'invalid zipcode', got '%s'", response.Message)
		}
	})

	t.Run("CEP não encontrado", func(t *testing.T) {
		cepResponse := `{"erro": true}`
		mockClient.AddResponse("https://viacep.com.br/ws/99999999/json/", 200, cepResponse)

		req, err := http.NewRequest("GET", "/weather/99999999", nil)
		if err != nil {
			t.Fatal(err)
		}

		rr := httptest.NewRecorder()
		router := mux.NewRouter()
		router.HandleFunc("/weather/{cep}", app.handleWeatherByCEP).Methods("GET")
		router.ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusNotFound {
			t.Errorf("Handler returned wrong status code: got %v want %v", status, http.StatusNotFound)
		}

		var response ErrorResponse
		err = json.Unmarshal(rr.Body.Bytes(), &response)
		if err != nil {
			t.Errorf("Error parsing response: %v", err)
		}

		if response.Message != "can not find zipcode" {
			t.Errorf("Expected message 'can not find zipcode', got '%s'", response.Message)
		}
	})
}
