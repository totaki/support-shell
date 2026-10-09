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
