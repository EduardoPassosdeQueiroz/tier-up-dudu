# Tier Du

Jogo web multiplayer em tempo real com Go + WebSocket e interface mobile-first.

## Rodar localmente

```bash
go mod tidy
go run .
```

Acesse `http://localhost:8080`.

## Regras e interface

- Rankings: S, A, B, C, E (azul) e F (roxo).
- Cinco temas por partida, escolhidos entre os 64 temas cadastrados, sem repetição até esgotar a lista.
- Respostas: 80 segundos. Poderes: 30 segundos. Resumo dos poderes: 10 segundos ou avanço pelo host.
- O host troca o tema durante as respostas; os demais precisam de maioria absoluta dos jogadores conectados. A troca limpa as respostas e reinicia o tempo, mantendo as letras.
- Cada jogador pode usar um poder por tema. Subidas e descidas se anulam antes de aplicar os limites S/F.
- O host pode remover jogadores no lobby. Ao desconectar, o participante conectado mais antigo assume.

## Imagem do tabuleiro

A tela está preparada para carregar `static/board.png`, com o tema da vez exibido como texto dinâmico. Enquanto esse arquivo não estiver disponível, apresenta um tabuleiro HTML com as seis letras e legendas. A imagem original ainda precisa ser adicionada: o download não pôde ser concluído no ambiente de edição.

## Verificação

Com Go instalado:

```bash
gofmt -w main.go game/room.go game/room_test.go
go test ./...
go vet ./...
```

Os testes em `game/room_test.go` cobrem votação, tempos, cancelamento dos poderes, extremos S/F, troca de host, remoção e o fluxo de cinco rodadas. Não foram executados neste ambiente, que não possui Go.

O teste de interface foi executado com Playwright e Chrome. Para repetir, disponibilize `playwright` no Node e execute:

```bash
node tests/interface.cjs
```

Use `CHROME_PATH` para indicar um Chrome local, ou instale o Chromium do Playwright. O teste verifica rascunho e cursor, cores, seis rankings, plaquinhas, poderes, notas e layout móvel.

## Deploy no Render

1. Crie um novo Web Service no Render conectando este repositório.
2. Configure:
   - **Build Command:** `go build -o tierup .`
   - **Start Command:** `./tierup`
3. Health Check Path: `/healthz`

O Render define automaticamente a variável `PORT`.
