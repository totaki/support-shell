# Support Shell

Интерактивная консоль поддержки на Go: команды и автодополнение, Kubernetes через локальный kubectl, AI-агент через OpenAI-compatible Chat Completions и расширения Extism/WASM.

**Состояние:** PoC. Изменяющие операции через обычный Executor запрещены. Результаты Kubernetes-инструментов отправляются выбранному LLM-провайдеру — не используйте подключение с доступом к секретам, если не допускается передача этих данных.

## Быстрый запуск

Требования: Go 1.23+, macOS/Linux, kubectl и рабочий kubeconfig для Kubernetes-команд.

```bash
go test ./...
make run
```

Команды для знакомства:

```text
help
commands
k8s contexts
k8s nodes
k8s pods kube-system
k8s pod restarts kube-system <pod>
k8s pod logs kube-system <pod> --previous=true
report
report save
agent reset
exit
```

Tab дополняет команды и имена ресурсов; повторное нажатие открывает список с управлением стрелками. История сохраняется локально, Ctrl+R выполняет поиск. `NO_COLOR=1` отключает цвета.

## Подключение агента

```bash
export SUPPORT_API_KEY="YOUR_KEY"
export SUPPORT_API_BASE="https://opencode.ai/zen/go/v1"
export SUPPORT_MODEL="glm-5.3-flash"
make run
```

Если введённый текст не является зарегистрированной командой, Shell передаёт его агенту. Агент получает только read-only инструменты Registry; результат их выполнения включается в контекст LLM. Без API-ключа можно пользоваться обычными командами (при стандартном API endpoint). Лимиты: `SUPPORT_AGENT_MAX_CALLS`, `SUPPORT_AGENT_MAX_ROUNDS`, `SUPPORT_AGENT_TIMEOUT_SECONDS`.

**Поведение ответов:** для простого списка ресурсов агенту предписано отвечать кратко с минимумом вызовов; для расследования инцидента — подробно проверять факты и отделять гипотезы. Это системная инструкция модели, а не жёстко enforced workflow.

## WASM-плагин

```bash
rustup target add wasm32-wasip1
make plugin
make run-plugin
# В Shell: diagnose k8s kube-system
```

Плагин `diagnostics` объявляет `support.shell/v1alpha1`, операции и capabilities. Host разрешает только заявленные read-only вызовы из allowlist. WASM не получает credentials Kubernetes напрямую. Для Extism-сборки нужен `github.com/extism/go-sdk` и его runtime.

## Документация

- [Архитектура](docs/architecture.md) — компоненты и путь выполнения.
- [Команды и Shell](docs/shell.md) — пользовательские операции и вывод.
- [Агент](docs/agent.md) — маршрутизация, политика ответов, лимиты, данные.
- [Плагины](docs/plugins.md) — Extism, контракт и пример.
- [Английский README](README.md).

Ограничения: нет MCP-сервера, полноценного управления permissions и безопасного approval для изменений; `kubectl` выполняется локально доверенным Go-модулем; агента нельзя считать детерминированным диагностическим workflow.


### Расширения и зависимости

По умолчанию `PLUGIN_PATH=./plugins`. При сборке `make plugin` загрузчик находит `plugins/diagnostics/target/wasm32-wasip1/release/*.wasm`. Можно положить готовые файлы в `plugins/*.wasm` или указать собственную директорию: `PLUGIN_PATH=/opt/plugins make run-plugin`. Поддерживается несколько WASM-плагинов. Старый `SUPPORT_PLUGIN` остаётся резервным вариантом.

```bash
make deps     # go mod tidy: сформировать/обновить go.sum
make plugin
make run-plugin
```

Системная инструкция агента хранится в `internal/agent/system_prompt.md`, а не в длинной строке Go-кода. Подробнее: [агент](docs/agent.md) и [плагины](docs/plugins.md).

### Менеджер WASM-плагинов

`plugins list` показывает загруженные плагины, версии, пути и capabilities; `plugins info diagnostics` — подробности одного модуля, в том числе команды. Данные берутся из загрузчика в текущем процессе, а не из сканирования диска после запуска. Без `-tags extism` список пустой.

Установка, удаление, горячая перезагрузка и проверка подписей пока не реализованы. Подробнее: [docs/plugins.md](docs/plugins.md).

Общий Rust SDK находится в `plugins/sdk`. Примеры `diagnostics` и `network` используют его без изменений Go Core. Собрать оба: `make plugin`, запустить: `make run-plugin`, проверить: `plugins list` и `network nodes`. Подробности: `docs/plugins.md`.

### Вывод команд

`plugins list` теперь отображается таблицей, а `plugins info diagnostics` — читаемыми полями и списком команд. Переключить общий режим вывода: `set format json`; обратно: `set format table` (по умолчанию). В JSON-режиме ответ AI выводится объектом с полем `answer`, а не разбирается на структурированные факты. Вывод команд и ответы агента сдвигаются на два пробела вправо от приглашения, без дополнительных пустых строк. См. [docs/shell.md](docs/shell.md).

Вызовы инструментов AI теперь выделены отдельным блоком `Tools` с линией окончания; текст агента начинается после заголовка `Answer`. Если инструменты не вызывались, лишние границы не выводятся.

Второй WASM-плагин теперь умеет не только `network nodes`, но и `network health`: проверяет Ready и Pressure по всем полученным нодам и возвращает структурированные findings (severity, code, source, condition). Неизвестные условия не считаются здоровыми. Подробнее — [docs/plugins.md](docs/plugins.md).

Небольшие улучшения UX: `network health` по умолчанию отображает таблицу findings; `Ctrl+C` при работе агента отменяет активный запрос и возвращает к Shell. Меню выбора не перерисовывает строку подсказки при каждом нажатии стрелки. У каждого WASM-плагина есть собственная документация: [diagnostics](plugins/diagnostics/README.md), [network](plugins/network/README.md).

### Единая документация инструментов

Плагины могут описывать себя в `describe()`: `title`, `description`, `author`, схемы `inputSchema` / `outputSchema` и примеры `examples`. Теперь это видно через `plugins list`, `plugins info network` и `help network health`; встроенный агент получает схему и примеры из того же Registry. Старые плагины совместимы. MCP-адаптер поверх Registry — следующий этап, пока он не реализован.

### MCP Server (stdio)

Support Shell теперь умеет работать как локальный MCP-сервер поверх того же Registry, что используют CLI и встроенный агент. Команда `support mcp serve` публикует read-only инструменты, включая команды загруженных WASM-плагинов, через `tools/list` и `tools/call`. Соберите `go build -tags extism -o support ./cmd/support`, затем настройте MCP-клиент на запуск `/path/to/support mcp serve` с `PLUGIN_PATH=/absolute/path/to/plugins`. Инструкции и пример конфигурации: [docs/mcp.md](docs/mcp.md).

Подсказки Tab работают также для встроенных команд: `help <команда>`, `set format table|json`, `plugins info <имя>`, `report save`, `agent reset` и Registry-команд. Динамическое дополнение аргументов зависит от наличия `Completer` у конкретной команды.

Plugin Runtime теперь проверяет уникальность ID команд в Registry и базовые JSON Schema аргументов перед выполнением — одинаково для CLI, AI и MCP. Поддерживаемые ограничения описаны в [docs/plugins.md](docs/plugins.md); полная JSON Schema пока не реализована.

### Сквозная проверка WASM + MCP

Команда `make test-e2e` собирает тестовый WASM-плагин `fixture`, запускает реальный MCP Server и проверяет `tools/list`, `tools/call` и валидацию аргументов. Тест не требует кластера Kubernetes. Код сценария: [scripts/e2e_mcp.py](scripts/e2e_mcp.py).

Tool permissions: SUPPORT_DENY_CLI, SUPPORT_DENY_AGENT and SUPPORT_DENY_MCP accept comma-separated command IDs. MCP and AI remain read-only. See docs/architecture.md.

### YAML-политики и прозрачность тестов

Можно указать `SUPPORT_POLICY_FILE=examples/tool-policy.yaml`. Формат: `deny.cli`, `deny.agent`, `deny.mcp` со списками ID команд. Переменные `SUPPORT_DENY_*` добавляют запреты поверх файла. Команда `tools permissions` показывает доступность команд и причины блокировки для всех интерфейсов.

GitHub Actions теперь формирует отчёт Go coverage в Summary workflow и сохраняет `coverage.out`/`coverage.txt` в артефакте `go-coverage`. Rust SDK, сетевые тесты и WASM→MCP E2E помечаются отдельными шагами CI. Coverage относится к Go-тестам; он не является покрытием Rust или полного Kubernetes E2E.
