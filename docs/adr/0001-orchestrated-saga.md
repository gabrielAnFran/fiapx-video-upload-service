# ADR 0001: Saga Orquestrada (nao Coreografada)

## Status
Aceito

## Contexto
Este servico e um dos quatro microsservicos de um sistema de processamento
de video para um hackathon FIAP X (upload -> extracao assincrona de frames
-> zip para download). O fluxo de negocio em si e uma pipeline linear (upload
-> processamento -> zip pronto ou falha), sem ramificacoes nem necessidade de
compensacao entre multiplas etapas de negocio distintas, ao contrario de uma
saga de pagamento/estoque classica. Ainda assim, e preciso decidir se a
comunicacao entre os quatro servicos e coordenada por um orquestrador central
(`fiapx-saga-orchestrator`) ou puramente coreografada, com cada servico
reagindo diretamente aos eventos de dominio dos outros.

## Decisao
Este servico participa de uma **saga orquestrada**, mesmo a pipeline sendo
linear. Ele emite `video.uploaded` (via outbox, apos gravar o video e o
objeto no MinIO) e consome os comandos terminais `video.status.completed` /
`video.status.failed` emitidos pelo orquestrador — nunca reage diretamente a
eventos de dominio de outros servicos (ex: nao escuta o evento de "frames
extraidos" do servico de processamento; escuta apenas o comando final que o
orquestrador decide emitir).

## Racional
- **Consistencia com os outros tres servicos**: os quatro microsservicos do
  desafio compartilham o mesmo envelope de evento e o mesmo
  `fiapx-saga-orchestrator`. Fazer este servico coreografado enquanto os
  outros sao orquestrados criaria dois modelos mentais diferentes no mesmo
  sistema, sem necessidade.
- **Trilha de auditoria**: mesmo numa pipeline linear, "por que esse video
  ficou em PROCESSING por 10 minutos e depois falhou" e uma pergunta que se
  responde melhor olhando o log do orquestrador do que juntando eventos de
  quatro servicos independentes.
- **Extensibilidade futura**: se o pipeline ganhar mais etapas (ex:
  moderacao de conteudo, transcodificacao em multiplas resolucoes, retry com
  compensacao parcial), ja existe um lugar natural para adicionar a logica
  de orquestracao sem reescrever o modelo de comunicacao deste servico.
- **Trade-off aceito**: para uma pipeline puramente linear como esta, um
  orquestrador e mais infraestrutura do que o estritamente necessario — uma
  coreografia direta (`video.uploaded` -> processing service reage -> emite
  `video.status.completed` direto) funcionaria e teria menos partes moveis.
  Aceitamos esse over-engineering modesto em troca da consistencia entre os
  quatro servicos e da trilha de auditoria centralizada.

## Consequencias
- Este servico implementa o padrao outbox para publicar `video.uploaded` de
  forma confiavel em resposta a um upload bem-sucedido.
- Este servico implementa tratamento idempotente dos comandos
  `video.status.completed` / `video.status.failed` via
  `ProcessedEventRepository`, ja que a entrega do orquestrador e
  "at-least-once".
- Este servico e um **consumidor terminal** desses dois comandos: ao
  processa-los, apenas atualiza seu proprio estado (`videos.status`), sem
  emitir nenhum evento adicional.
