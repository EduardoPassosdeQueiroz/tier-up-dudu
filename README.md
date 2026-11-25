# Tier Du

Jogo web multiplayer em tempo real com Go + WebSocket e interface mobile-first.

## Rodar localmente

```bash
go mod tidy
go run .
```

Acesse `http://localhost:8080`.

> `go mod tidy` é usado apenas para o desenvolvimento local. A pipeline de CI não executa `go mod tidy`, para não alterar `go.mod` e `go.sum` automaticamente.

## Regras e interface

- Rankings: S, A, B, C, E (azul) e F (roxo).
- Três rodadas por partida. No início, são sorteados 15 temas distintos da lista atualizada: antes de cada rodada, todos os jogadores conectados votam em um grupo de 5 opções. O mais votado vence; empates são sorteados entre os empatados. Nenhuma sugestão se repete na mesma partida.
- Respostas: 120 segundos (2:00). Poderes: 15 segundos. Resumo dos poderes: 10 segundos ou avanço pelo host.
- A votação termina quando todos os jogadores conectados votam. É possível alterar o voto enquanto estiver aberta; desconectar remove o voto da contagem. O host não escolhe sozinho nem troca o tema durante as respostas.
- Cada letra é sorteada independentemente para cada jogador. Todos devem confirmar a tela de instruções antes da partida.
- Na revisão, é possível curtir respostas de outras pessoas. O resultado final mostra as rodadas, pontos das respostas, curtidas e contribuição individual dos poderes.
- A nota secreta aparece apenas enquanto o quadrado é pressionado com mouse, toque ou teclado. Ao soltar ou sair da janela, volta ao ponto de interrogação.
- As comparações mostram a nota original à esquerda e a escolhida à direita; após poderes, mostram antes à esquerda e depois à direita.
- Cada jogador pode usar um poder por tema; quem já usou aguarda os outros automaticamente. Subidas e descidas se anulam antes de aplicar os limites S/F.
- O host pode remover jogadores no lobby. Ao desconectar, o participante conectado mais antigo assume.

Ao terminar, todos permanecem na mesma sala. O host pode usar “Jogar de novo na mesma sala” para voltar ao lobby com os mesmos jogadores e credenciais; o próximo início sorteia 15 temas novamente e zera os resultados e poderes.

## Retorno à sala

O navegador salva uma credencial privada de sessão, a última tela e o rascunho. Ao perder a conexão, mantém a tela aberta e tenta reconectar automaticamente. Ao voltar de outra aba, de um período em segundo plano ou após atualizar/reabrir a página, recupera o mesmo jogador, sem duplicar sua vaga. O rascunho é recuperado apenas na mesma rodada e no mesmo tema; trocar o tema limpa o texto.

O servidor preserva jogadores desconectados no lobby e valida a credencial no retorno. Jogadores removidos pelo host não podem recuperar a sessão removida. Se outro jogador assumiu como host, ele mantém essa função quando o host anterior retorna.

As salas continuam armazenadas na memória do processo. Reiniciar ou republicar o servidor encerra as salas existentes; persistência entre reinícios exige armazenamento durável no servidor. Os cronômetros da partida continuam correndo durante a ausência.

## Imagem do tabuleiro

A tela está preparada para carregar `static/board.png`, com o tema da vez exibido como texto dinâmico. Enquanto esse arquivo não estiver disponível, apresenta um tabuleiro HTML com as seis letras e legendas. A imagem original ainda precisa ser adicionada: o download não pôde ser concluído no ambiente de edição.

## Verificação

Com Go instalado, rode na raiz do projeto:

```bash
go test ./...
go vet ./...
```

Para formatar o código localmente, use `gofmt -w` nos arquivos alterados. Esse comando modifica os arquivos, por isso é usado apenas no desenvolvimento local e não faz parte da CI.

Os testes em `game/room_test.go` cobrem votação, tempos, cancelamento dos poderes, extremos S/F, troca de host, remoção e o fluxo de três rodadas.

### Testes de navegador

O teste de interface foi executado com Playwright e Chrome. Para repetir, disponibilize `playwright` no Node e execute:

```bash
node tests/interface.cjs
node tests/reconnect.cjs
node tests/new-rules.cjs
node tests/requested-updates.cjs
```

Use `CHROME_PATH` para indicar um Chrome local, ou instale o Chromium do Playwright. O teste verifica rascunho e cursor, cores, seis rankings, plaquinhas, poderes, notas e layout móvel.

O teste de reconexão usa um servidor WebSocket simulado no navegador para verificar a repetição automática, o reaproveitamento da credencial, o retorno à sala após recarregar, a restauração do rascunho e a expiração da sessão. Os testes Go de sessão validam a preservação do jogador, o sigilo da credencial, os 120 segundos e a invalidação após remoção.

> Os testes de navegador **não são executados pela pipeline de CI**, que roda apenas os testes Go.

## Integração Contínua e Entrega Contínua (CI/CD)

Este projeto usa GitHub Actions para testar o código automaticamente e, depois, implantar a versão aprovada no Render.

### CI (Integração Contínua)

Arquivo: `.github/workflows/ci.yml`

**Quando roda:** em todo `push` para a branch `main` e em todo Pull Request direcionado à `main`.

**O que faz, nesta ordem:**
1. Baixa o código do repositório.
2. Configura o Go 1.22.
3. Verifica as dependências com `go mod download` e `go mod verify`.
4. Executa os testes automatizados com `go test ./...`.
5. Executa a análise estática com `go vet ./...`.
6. Compila o projeto com `go build` para confirmar que não há erros de build.

**Por que existe:** garante que cada alteração compila e passa nos testes antes de ir para a `main`. Se qualquer etapa falhar, a CI fica vermelha e a CD não é executada.

### CD (Entrega Contínua)

Arquivo: `.github/workflows/cd.yml`

**Quando roda:** somente após a CI ser concluída com sucesso em um `push` para a `main`.

**Não roda em:** Pull Requests, nem quando a CI falha.

**O que faz:** envia uma requisição ao Deploy Hook do Render, que inicia a implantação da versão aprovada.

### Configuração do segredo

A URL do Deploy Hook é privada e não fica no repositório.

1. No Render, abra o serviço `tier-up-dudu` → **Settings** → seção **Deploy Hook** e copie a URL.
2. No GitHub, abra **Settings** → **Secrets and variables** → **Actions** → **New repository secret**.
3. Em **Name**, digite `RENDER_DEPLOY_HOOK_URL`. Em **Secret**, cole a URL e salve.

### Como interpretar os resultados

- Acesse a aba **Actions** do repositório para ver cada execução.
- **CI verde:** dependências, testes, análise estática e build passaram.
- **CI vermelha:** abra o job e veja a etapa que falhou. A CD não é executada nesse caso.
- **CD ignorada (skipped):** a CI não terminou com sucesso, ou o evento foi um Pull Request.
- **CD verde:** o Deploy Hook foi acionado. Confirme o andamento em **Events** do serviço no Render.
- Para verificar o serviço no ar, acesse `/healthz`, que deve responder `ok`.

## Deploy no Render

O serviço `tier-up-dudu` no Render está configurado com:

- **Build Command:** `go build -o app .`
- **Start Command:** `./app`
- **Health Check Path:** `/healthz`
- **Auto-Deploy:** desligado (a implantação é feita pela CD do GitHub Actions)

O arquivo `render.yaml` documenta essa configuração, mas não cria um Blueprint. O Render define automaticamente a variável `PORT`.
