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
