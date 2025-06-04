# Weather API - Sistema de Consulta de Clima por CEP

Este sistema em Go recebe um CEP válido, identifica a cidade e retorna o clima atual em Celsius, Fahrenheit e Kelvin.

## Funcionalidades

- ✅ Validação de CEP (8 dígitos)
- ✅ Consulta de localização via API ViaCEP
- ✅ Consulta de clima via WeatherAPI
- ✅ Conversão automática de temperaturas (C°, F°, K)
- ✅ Tratamento de erros adequado
- ✅ Testes automatizados
- ✅ Deploy no Google Cloud Run
- ✅ Containerização com Docker

## Requisitos

### Para desenvolvimento local:
- Go 1.21+
- Docker e Docker Compose
- Chave da API WeatherAPI ([obtenha aqui](https://www.weatherapi.com/))

### Para deploy no Google Cloud:
- Google Cloud SDK
- Projeto no Google Cloud Platform
- Cloud Run e Container Registry habilitados

## Instalação e Configuração

### 1. Clone o repositório
```bash
git clone <repository-url>
cd projeto-deploy
```

### 2. Configure as variáveis de ambiente
```bash
# Copie o arquivo de exemplo
cp .env.example .env

# Edite o arquivo .env com sua chave da WeatherAPI
WEATHER_API_KEY=your_weather_api_key_here
PORT=8080
```

### 3. Instale as dependências
```bash
go mod tidy
```

## Como usar

### Execução local

#### Método 1: Go direto
```bash
export WEATHER_API_KEY=your_api_key_here
go run main.go
```

#### Método 2: Docker Compose (recomendado)
```bash
# Configure sua WEATHER_API_KEY no arquivo .env
docker-compose up --build
```

### Endpoints da API

#### Consultar clima por CEP
```http
GET /weather/{cep}
```

**Exemplos de uso:**
```bash
# CEP com formatação
curl http://localhost:8080/weather/01310-100

# CEP sem formatação
curl http://localhost:8080/weather/01310100
```

### Respostas da API

#### Sucesso (200)
```json
{
  "temp_C": 25.0,
  "temp_F": 77.0,
  "temp_K": 298.0
}
```

#### CEP inválido (422)
```json
{
  "message": "invalid zipcode"
}
```

#### CEP não encontrado (404)
```json
{
  "message": "can not find zipcode"
}
```

## Testes

### Executar todos os testes
```bash
go test -v
```

### Executar testes com cobertura
```bash
go test -v -cover
```

### Executar testes específicos
```bash
# Testes de validação
go test -v -run TestIsValidCEP

# Testes de conversão
go test -v -run TestTemperatureConversions

# Testes de integração
go test -v -run TestHandleWeatherByCEP
```

## Deploy no Google Cloud Run

### 1. Configuração inicial
```bash
# Login no Google Cloud
gcloud auth login

# Definir projeto
gcloud config set project YOUR_PROJECT_ID

# Habilitar APIs necessárias
gcloud services enable cloudbuild.googleapis.com
gcloud services enable run.googleapis.com
gcloud services enable secretmanager.googleapis.com
```

### 2. Configurar secrets
```bash
# Criar secret para a API key
gcloud secrets create weather-api-key --data-file=<(echo -n "YOUR_WEATHER_API_KEY")
```

### 3. Deploy manual
```bash
# Build e push da imagem
gcloud builds submit --tag gcr.io/YOUR_PROJECT_ID/weather-api

# Deploy no Cloud Run
gcloud run deploy weather-api \
  --image gcr.io/YOUR_PROJECT_ID/weather-api \
  --platform managed \
  --region us-central1 \
  --allow-unauthenticated \
  --set-env-vars WEATHER_API_KEY=YOUR_API_KEY \
  --memory 512Mi \
  --cpu 1
```

### 4. Deploy automatizado (com Cloud Build)
```bash
# Trigger manual
gcloud builds submit --config cloudbuild.yaml

# Ou configure trigger automático no repositório Git
gcloud beta builds triggers create github \
  --repo-name=YOUR_REPO \
  --repo-owner=YOUR_USERNAME \
  --branch-pattern="^main$" \
  --build-config=cloudbuild.yaml
```

## Deploy em produção

O serviço está disponível em:

https://projeto-deploy-875860357089.us-central1.run.app

## Estrutura do Projeto

```
projeto-deploy/
├── main.go              # Código principal da aplicação
├── main_test.go         # Testes automatizados
├── go.mod              # Dependências do Go
├── go.sum              # Checksums das dependências
├── Dockerfile          # Configuração do container
├── docker-compose.yml  # Configuração do Docker Compose
├── cloudbuild.yaml     # Configuração do Cloud Build
├── .gitignore          # Arquivos ignorados pelo Git
└── README.md           # Este arquivo
```

## APIs Utilizadas

### ViaCEP
- **URL**: https://viacep.com.br/ws/{cep}/json/
- **Documentação**: https://viacep.com.br/
- **Uso**: Consulta de informações de localização por CEP

### WeatherAPI
- **URL**: https://api.weatherapi.com/v1/current.json
- **Documentação**: https://www.weatherapi.com/docs/
- **Uso**: Consulta de informações climáticas atuais
- **Requer**: Chave de API gratuita

## Fórmulas de Conversão

### Celsius para Fahrenheit
```
F = C * 1.8 + 32
```

### Celsius para Kelvin
```
K = C + 273
```

## Monitoramento e Logs

### Logs no Cloud Run
```bash
# Visualizar logs em tempo real
gcloud logging tail "resource.type=cloud_run_revision"

# Logs específicos do serviço
gcloud logging read "resource.type=cloud_run_revision AND resource.labels.service_name=weather-api"
```

### Métricas
- Latência das requisições
- Taxa de erro
- Uso de CPU e memória
- Número de instâncias ativas

## Contribuição

1. Fork o projeto
2. Crie uma branch para sua feature (`git checkout -b feature/AmazingFeature`)
3. Commit suas mudanças (`git commit -m 'Add some AmazingFeature'`)
4. Push para a branch (`git push origin feature/AmazingFeature`)
5. Abra um Pull Request

## Licença

Este projeto está sob a licença MIT. Veja o arquivo `LICENSE` para mais detalhes.

## Suporte

Para suporte ou dúvidas:
- Abra uma issue no GitHub
- Entre em contato via email

## Changelog

### v1.0.0 (2024-01-01)
- ✅ Implementação inicial
- ✅ Validação de CEP
- ✅ Integração com ViaCEP e WeatherAPI
- ✅ Conversões de temperatura
- ✅ Testes automatizados
- ✅ Containerização
- ✅ Deploy no Google Cloud Run 