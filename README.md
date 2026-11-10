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
- Três temas por partida, escolhidos aleatoriamente entre os temas cadastrados; o tema atual é excluído do próximo sorteio.
- Respostas: 120 segundos (2:00), inclusive após trocar o tema. Poderes: 30 segundos. Resumo dos poderes: 10 segundos ou avanço pelo host.
- A troca de tema exige maioria absoluta dos jogadores conectados, inclusive quando o voto vem do host. A troca limpa as respostas e reinicia o tempo, mantendo as letras.
- Cada letra é sorteada independentemente para cada jogador. Todos devem confirmar a tela de instruções antes da partida.
- Na revisão, é possível curtir respostas de outras pessoas. O resultado final mostra as rodadas, pontos das respostas, curtidas e contribuição individual dos poderes.
- Cada jogador pode usar um poder por tema. Subidas e descidas se anulam antes de aplicar os limites S/F.
- O host pode remover jogadores no lobby. Ao desconectar, o participante conectado mais antigo assume.

## Retorno à sala

O navegador salva uma credencial privada de sessão, a última tela e o rascunho. Ao perder a conexão, mantém a tela aberta e tenta reconectar automaticamente. Ao voltar de outra aba, de um período em segundo plano ou após atualizar/reabrir a página, recupera o mesmo jogador, sem duplicar sua vaga. O rascunho é recuperado apenas na mesma rodada e no mesmo tema; trocar o tema limpa o texto.

O servidor preserva jogadores desconectados no lobby e valida a credencial no retorno. Jogadores removidos pelo host não podem recuperar a sessão removida. Se outro jogador assumiu como host, ele mantém essa função quando o host anterior retorna.

As salas continuam armazenadas na memória do processo. Reiniciar ou republicar o servidor encerra as salas existentes; persistência entre reinícios exige armazenamento durável no servidor. Os cronômetros da partida continuam correndo durante a ausência.

## Imagem do tabuleiro

A tela está preparada para carregar `static/board.png`, com o tema da vez exibido como texto dinâmico. Enquanto esse arquivo não estiver disponível, apresenta um tabuleiro HTML com as seis letras e legendas. A imagem original ainda precisa ser adicionada: o download não pôde ser concluído no ambiente de edição.

## Verificação

Com Go instalado:

```bash
gofmt -w main.go game/room.go game/room_test.go game/session.go game/session_test.go
go test ./...
go vet ./...
```

Os testes em `game/room_test.go` cobrem votação, tempos, cancelamento dos poderes, extremos S/F, troca de host, remoção e o fluxo de três rodadas.

O teste de interface foi executado com Playwright e Chrome. Para repetir, disponibilize `playwright` no Node e execute:

```bash
node tests/interface.cjs
node tests/reconnect.cjs
```

Use `CHROME_PATH` para indicar um Chrome local, ou instale o Chromium do Playwright. O teste verifica rascunho e cursor, cores, seis rankings, plaquinhas, poderes, notas e layout móvel.

O teste de reconexão usa um servidor WebSocket simulado no navegador para verificar a repetição automática, o reaproveitamento da credencial, o retorno à sala após recarregar, a restauração do rascunho e a expiração da sessão. Os testes Go de sessão validam a preservação do jogador, o sigilo da credencial, os 120 segundos e a invalidação após remoção.

## Deploy no Render

1. Crie um novo Web Service no Render conectando este repositório.
2. Configure:
   - **Build Command:** `go build -o tierup .`
   - **Start Command:** `./tierup`
3. Health Check Path: `/healthz`

O Render define automaticamente a variável `PORT`.

<!-- Teste: PR sem deploy -->