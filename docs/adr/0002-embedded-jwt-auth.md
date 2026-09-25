# ADR 0002: Autenticacao JWT Embutida (nao um Servico de Auth Separado)

## Status
Aceito

## Contexto
Este servico precisa autenticar usuarios (registro com email/senha, login) e
proteger os endpoints de upload/listagem/download de video, garantindo que
cada usuario so acesse seus proprios videos. E preciso decidir se essa
responsabilidade fica embutida neste proprio servico ou se e extraida para um
quinto microsservico dedicado de autenticacao/identidade.

## Decisao
A autenticacao (bcrypt para hash de senha + JWT HS256 assinado por este
servico, 24h de expiracao) fica **embutida neste servico**, na tabela
`users` do proprio banco Postgres deste servico. Nao existe um servico de
auth separado nem um lookup de documento multi-tenant (como CPF/CNPJ de
cliente em outro dominio) que justificasse extrair essa responsabilidade.

## Racional
- **Sem necessidade de lookup multi-tenant**: ao contrario de um cenario
  onde varios servicos precisam resolver "quem e esse usuario" a partir de
  um documento compartilhado entre dominios (cliente de oficina, por
  exemplo), aqui o unico consumidor da identidade do usuario e este proprio
  servico — os outros tres microsservicos do pipeline (processamento,
  notificacao, orquestrador) trabalham com `video_id`/`user_id` como
  identificadores opacos vindos do evento `video.uploaded`, sem nunca
  precisar validar credenciais.
- **Escopo do hackathon**: um servico de identidade separado adicionaria uma
  chamada de rede sincrona (ou mais um evento) no caminho critico de todo
  request autenticado, sem nenhum ganho de isolamento real dado que so este
  servico emite e valida esses tokens.
- **Simplicidade operacional**: um segredo JWT (`UPLOAD_JWT_SECRET`), uma
  tabela, dois endpoints (`/auth/register`, `/auth/login`) e um middleware
  (`AuthRequired`) resolvem o problema por completo sem mais um banco, mais
  um deployment, mais um ponto de falha na cadeia de autenticacao.
- **Trade-off aceito**: se um dia outro dos quatro servicos precisar
  autenticar usuarios diretamente (por exemplo, um servico de gestao de
  conta), essa decisao precisaria ser revisitada — o segredo JWT teria que
  ser compartilhado ou a validacao centralizada. Para o escopo atual (um
  unico servico expondo endpoints autenticados), isso nao se aplica.

## Consequencias
- Senhas sao hasheadas com bcrypt (`DefaultCost`) antes de qualquer escrita
  em banco; o hash em texto claro nunca e logado nem retornado em resposta
  HTTP.
- O segredo de assinatura JWT e passado explicitamente como parametro
  (`GenerateToken`/`ValidateToken`/`AuthRequired(secret)`) em vez de guardado
  numa variavel global mutavel, evitando acoplamento entre testes que
  rodariam em paralelo com segredos diferentes.
- Todo endpoint de video (`POST/GET /videos`, `GET /videos/:id/download`)
  exige um Bearer token valido e usa o `user_id` das claims — nunca um
  `user_id` vindo do corpo/query da requisicao — para escopar queries e
  decidir 403 em tentativas de acesso a video de outro usuario.
