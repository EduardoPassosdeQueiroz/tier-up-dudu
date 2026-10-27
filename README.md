# Tier Up

Jogo web multiplayer em tempo real com Go + WebSocket e interface mobile-first.

## Rodar localmente

```bash
go mod tidy
go run .
```

Acesse `http://localhost:8080`.

## Deploy no Render

1. Crie um novo Web Service no Render conectando este repositório.
2. Configure:
   - **Build Command:** `go build -o tierup .`
   - **Start Command:** `./tierup`
3. Health Check Path: `/healthz`

O Render define automaticamente a variável `PORT`.
