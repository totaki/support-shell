# AI agent

The agent is an optional OpenAI-compatible Chat Completions client. Unmatched user input is sent to the model, with read-only Registry commands exposed as function tools. It never receives unrestricted shell execution.

## Adaptive response policy

This is implemented in the **system instruction**, not by changing dispatch or adding a workflow engine:

| Request | Expected behavior |
| --- | --- |
| "Какие ноды запущены?" | `k8s.nodes` and short summary/table |
| "Покажи загрузку нод" | Node metrics as needed |
| "Проверь здоровье нод" | Relevant health indicators, investigate observed anomalies |
| "Почему calico рестартился?" | Container status, previous logs, events, node evidence; facts separated from hypotheses |

The agent should not call `describe` on all nodes, use unrelated resources, or force incident report headings for simple inventory queries. Detailed incident responses distinguish **Подтверждено / Гипотезы / Что проверить / Рекомендации**. This is probabilistic model guidance: actual tool usage and answer shape can still vary.

## Safety and data

Only read-risk Registry tools are exposed. Results from different tools backed by the same Kubernetes API are not independent corroboration. Cumulative restart counts do not indicate current restart rates; probe configuration alone is not proof of probe failures.

Tool outputs (including logs) may contain sensitive information and are sent to the configured external model provider. Tool responses are truncated for model context; reports may preserve operational detail. Use least privilege.

## Settings

```bash
SUPPORT_API_KEY=...
SUPPORT_API_BASE=https://opencode.ai/zen/go/v1
SUPPORT_MODEL=glm-5.3-flash
SUPPORT_AGENT_TIMEOUT_SECONDS=240
SUPPORT_AGENT_MAX_CALLS=16
SUPPORT_AGENT_MAX_ROUNDS=8
```

OpenCode Go receives a stable `x-opencode-session` header; `agent reset` rotates the session and resets history. Tool calls are traced with their arguments and durations. An agent response can be saved with `report save`.


## Prompt maintenance

The complete model instruction lives at `internal/agent/system_prompt.md` and is embedded in the binary via `//go:embed`. Changes require recompiling the Go binary. It tells the agent to use previous named resources as investigative context while refreshing live observations, to avoid broad claims when only a subset was checked, to report missing previous logs honestly, and to use clean Russian. This remains instruction-level guidance, not deterministic enforcement.

## Проверка полноты утверждений

Инструкция в `internal/agent/system_prompt.md` требует подтверждать каждое поле сводной таблицы фактическими данными для соответствующего объекта. Результат `k8s.nodes` может содержать условия для всех нод; выборочные `k8s.node.describe` не означают проверки остальных нод. Неизвестные значения нельзя заполнять предположениями. Отчёт о здоровье не должен уходить в overcommit и SPOF без запроса или наблюдаемой причины. Также модель должна оформлять маркированные списки с пробелом после дефиса; это рекомендация модели, а не детерминированная коррекция текста.
