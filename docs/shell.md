# Shell and commands

The REPL first tries to resolve input as a known command. Unmatched input is sent to the configured agent; arguments invalid for a recognized command remain a CLI error.

## Navigation

- `Tab`: longest common prefix, followed by interactive candidates.
- `↑/↓`: history; `←/→`: cursor navigation.
- `Ctrl+R`: history search; `Ctrl+A/E`: beginning/end.
- `help`, `commands`, `history`, `exit`.
- `NO_COLOR=1` disables ANSI styling.

## Kubernetes

```text
k8s contexts
k8s context use <context>
k8s nodes
k8s namespaces
k8s pods <namespace>
k8s events <namespace>
k8s pod describe <namespace> <pod>
k8s pod logs <namespace> <pod> --tail=100 --since=15m
k8s pod logs <namespace> <pod> --previous=true --timestamps=true
k8s pod restarts <namespace> <pod>
k8s node describe <node>
k8s top nodes
k8s top pods <namespace>
```

`k8s context use` presents an interactive `Apply? [y/N]` confirmation and changes the active local kubeconfig context. Read commands return structured data internally; the Shell renders concise tables for common resource lists and terminal Markdown for agent responses.

`report` displays the latest agent report; `report save` writes Markdown into the user's configuration directory. Saved reports and history can include sensitive operational details.

Current limitations: simple whitespace argument parsing, no streaming `logs --follow`, and resource completions require permission to list Kubernetes objects.

## Формат вывода и границы команд

По умолчанию используется `set format table`: ресурсные списки Kubernetes и `plugins list` представлены таблицами, `plugins info <name>` — читаемым подробным описанием. Для машинной обработки используйте `set format json`: результаты Registry-команд печатаются как JSON; обычный текст превращается в JSON-строку. Ответ агента в этом режиме выводится объектом `{"answer":"..."}` — это текст ответа, не структурированный JSON-диагноз. Вернуться к обычному выводу: `set format table`.

Вывод результата команды и ответа агента сдвигается на два пробела вправо относительно приглашения. Это сохраняет взаимное выравнивание строк таблиц, JSON и Markdown. Дополнительные вертикальные отступы убраны. Настройка формата действует в пределах текущего процесса Shell и не записывается в конфигурационный файл.

## Разделение инструментов и ответа AI

При первом вызове инструмента Shell печатает заголовок `┌ Tools`, затем компактные строки вызовов и ошибок. После завершения агента печатается `└──────────────` и заголовок `Answer`, далее — ответ с обычным левым отступом. Для запросов без вызовов tools блок не рисуется. Оформление учитывает `NO_COLOR`.

## Markdown и списки

Рендерер принимает стандартные маркеры списка (`- пункт`, `* пункт`, `1. пункт`) и сохраняет буквальные дефисы, например `--previous=true`. Строка `-heartbeat` не является Markdown-списком и остаётся без изменений: её необходимо корректно сформировать на стороне модели. Для этого системная инструкция агента требует использовать корректный синтаксис списков.

## Ctrl+C и выбор из меню

Во время исполнения команды, включая запросы AI и read-only tools, `Ctrl+C` отменяет текущий контекст выполнения и возвращает управление интерактивному Shell. Незавершённый диалоговый ход AI не сохраняется в его истории. В режиме ввода `Ctrl+C` по-прежнему завершает REPL (поведение редактора). В меню completion подсказка `↑↓ select...` фиксирована; при перемещении меняются только строки вариантов. При отмене не выводится пустой заголовок `Answer`.

`network health` отображается таблицей findings по умолчанию. Для исходного JSON используйте `set format json`.

## Command discovery

`help <command>` now renders metadata from the central Registry (e.g. `help network health`): description, declared risk, argument names, example invocations and JSON input/output schemas where supplied by the WASM plugin. `plugins list` shows the short description supplied by each loaded plugin; `plugins info NAME` includes its title and description.
