# ADR 0003: MinIO para Armazenamento de Objetos (nao Postgres nem Disco Local)

## Status
Aceito

## Contexto
Cada upload de video pode ter centenas de megabytes ou mais, e o resultado
do processamento (zip com os frames extraidos) tambem e um blob grande. E
preciso decidir onde armazenar esses arquivos binarios: no mesmo Postgres
usado para os metadados relacionais (`users`, `videos`, `outbox`), em disco
local do container/pod, ou num object store dedicado.

## Decisao
Os bytes do video original e do zip de frames sao armazenados no **MinIO**,
via a API S3 (`aws-sdk-go-v2` apontando para o endpoint do MinIO com
path-style addressing). O Postgres deste servico guarda apenas metadados —
`source_bucket`/`source_object_key` e `zip_bucket`/`zip_object_key` como
referencias, nunca o conteudo binario.

## Racional
- **Blobs nao pertencem a um banco relacional**: guardar arquivos de
  centenas de MB como `bytea`/`large object` no Postgres infla o banco,
  degrada backup/restore e desperdica um recurso otimizado para dados
  estruturados em algo para o qual ele nao foi desenhado.
- **Disco local nao sobrevive a replicas/restarts**: os tres binarios deste
  servico (`server`, `worker`, `outbox-dispatcher`) rodam como pods
  independentes e potencialmente multiplas replicas; disco local por pod nao
  e compartilhado nem persistente entre eles, tornando invivel servir depois
  um arquivo que foi recebido em outro pod.
- **Compatibilidade com S3 real**: MinIO fala a API S3 nativamente. O
  `S3Client` deste servico e escrito contra a API padrao do
  `aws-sdk-go-v2/service/s3`; trocar o MinIO local por um bucket AWS S3 real
  em producao exige apenas mudar endpoint/credenciais, sem tocar em codigo —
  um caminho de evolucao natural para depois do hackathon.
- **Streaming sem buffer completo em memoria**: o upload usa
  `manager.NewUploader(...).Upload`, que faz multipart upload em chunks,
  entao o tamanho do arquivo nao fica limitado pela memoria disponivel do
  pod, ao contrario do que aconteceria bufferizando o corpo inteiro num
  `[]byte` antes de escrever em qualquer destino.
- **Download sem proxy pelo backend**: URLs presignadas
  (`PresignGetObject`, TTL de 15 minutos) deixam o cliente baixar o zip
  diretamente do MinIO, sem esse servico precisar proxiar centenas de MB de
  volta para o usuario.

## Consequencias
- Este servico depende de um MinIO acessivel (`MINIO_ENDPOINT`,
  `MINIO_ACCESS_KEY`, `MINIO_SECRET_KEY`, `MINIO_BUCKET`) alem do Postgres e
  do RabbitMQ.
- O upload HTTP (`POST /api/v1/videos`) so retorna `202 Accepted` depois que
  o objeto ja foi gravado no MinIO com sucesso — se o upload para o MinIO
  falhar, nenhuma linha de video nem evento de outbox e criada.
- O download real dos frames processados nunca passa por este servico: o
  cliente recebe a URL presignada e baixa diretamente do MinIO/S3.
