# distributed-queue

Estudo sobre filas e workers concorrentes em Go. Mantive o escopo simples, mas com as mesmas preocupações que levaria para produção: cancelamento, backpressure, testes, observabilidade planejada e espaço para melhorias claras.

## Como rodar

```bash
GOCACHE=$(pwd)/.cache/go-build go test ./...
WORKERS=3 TASKS=10 go run ./cmd/demo
```

Flags via env:
- `WORKERS`: número de workers (default 3)
- `TASKS`: quantidade de tarefas publicadas (default 10)

Saída esperada: enfileiramentos e workers processando com tempos simulados. Ctrl+C encerra.

## Arquitetura atual

- **task** (`internal/task`): modelo de tarefa com ID, tipo, payload e timestamp.
- **queue** (`internal/queue`): broker em memória (channel bufferizado) com `Enqueue`, `Dequeue`, `Close`, respeitando contextos.
- **worker pool** (`internal/worker`): pool fixo consumindo do broker; Start/Stop, contexto e espera por shutdown limpo.
- **demo** (`cmd/demo`): wiring simples que instancia broker, pool, publica tarefas e loga processamento.

## Decisões e preocupações

- **Cancelamento/shutdown**: uso de `context.Context` na fila e nos workers; `Stop` aguarda handlers finalizarem.
- **Backpressure**: capacidade configurável do broker; `Enqueue` bloqueia quando cheio, respeitando contexto.
- **Testes**: cobrem fila (cancelamento, fechamento) e pool (processamento e parada); cache local para evitar restrições de permissões.
- **Simplicidade intencional**: usei channel para broker e handler síncrono; espaço para trocar por backends reais sem quebrar API.

## Checklist de próximos passos (estudo/aperfeiçoamento)

- [ ] Retentativas com backoff e DLQ simples
- [ ] Logs estruturados (campos para task_id, tentativa, duração)
- [ ] Métricas: contadores/latências + expor /metrics (Prometheus)
- [ ] Timeouts por tarefa e limites de concorrência por tipo
- [ ] Simulação de falhas (handlers aleatórios) e testes de tolerância
- [ ] CLI com flags/commandos claros (ex.: `publish`, `run`) usando cobra
- [ ] Benchmarks básicos e corrida (`-race`) no CI
- [ ] Dockerfile e docker-compose de demo (broker externo fake)

## Nota pessoal

O objetivo não é entregar um produto pronto, mas mostrar disciplina de engenharia em algo pequeno: testes primeiro, shutdown bem pensado, roadmap explícito e foco em clareza. A ideia é evoluir incrementalmente com commits pequenos e mensagens descritivas.***
# distributed-queue
