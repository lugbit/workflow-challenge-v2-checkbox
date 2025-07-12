## Setup Instructions

First step is the same as in the main README.md file.

### 1. Start All Services

```bash
cd workflow-challenge-v2
docker-compose up --build
```

- This launches frontend, backend, and database with hot reloading enabled for code changes.
- To stop and clean up:
  ```bash
  docker-compose down
  ```

### 2. Access Applications

- **Frontend (Workflow Editor):** [http://localhost:3003](http://localhost:3003)
- **Backend API:** [http://localhost:8086](http://localhost:8086)
- **Database:** PostgreSQL on `localhost:5876`

### 3. Load DB migrations

Note: This is currently handled manually. In a real-world application, you should use a migration tool like golang-migrate to manage version control and ensure schema changes can be easily deployed and rolled back.

- DB migration files are located in `api/sql`
- Manually execute the up migration SQL `001_create_workflows_table.up.sql` by connecting to the Postgres DB (CLI or PgAdmin)

### `workflows` Table Schema

| Column       | Type        | Constraints                               | Description                               |
| ------------ | ----------- | ----------------------------------------- | ----------------------------------------- |
| `id`         | UUID        | Primary Key, Default: `gen_random_uuid()` | Unique identifier for each workflow       |
| `definition` | JSONB       | Not Null                                  | JSON representation of the workflow graph |
| `name`       | TEXT        | Not Null                                  | Human-readable name for the workflow      |
| `created_at` | TIMESTAMPTZ | Default: `NOW()`                          | Timestamp of creation                     |
| `updated_at` | TIMESTAMPTZ | Default: `NOW()`                          | Timestamp of last update                  |

## 🏗️ Project Architecture

```text
workflow-code-test/
├── api/                                  # Go Backend (Port 8086)
│   ├── main.go
│   ├── go.mod
│   ├── go.sum
│   ├── Dockerfile
│   ├── sql/                              # SQL migration files
│   ├── tmp/
│   ├── vendor/
│   ├── pkg/
│   └── services/
│       └── workflow/
│           ├── errors.go                 # Custom errors
│           ├── node.go                   # Workflow struct definitions
│           ├── node_processor.go         # Main function for processing workflows
│           ├── node_processor_test.go    # Unit tests for process workflow + node type logic
│           ├── repository.go             # Re-usable DB methods
│           ├── service.go
│           └── workflow.go               # API layer
├── README.md
├── README_MARCK.md                       # This document
```

## Assumptions and trade-offs

### Assumptions

- The workflow editor does **not** allow adding or removing nodes for this exercise.
- **Individual nodes are immutable** — their structure (e.g., form fields) cannot be edited.
- During execution, if a node **fails**, its error is reported and the **workflow halts immediately**, skipping any remaining nodes.
- The workflow is a **directed graph**: nodes are executed in sequence, and execution cannot move backward.
- The **condition node** controls branching:
  - If the condition evaluates to `true`, the **email node** is executed.
  - If the condition evaluates to `false`, the email node is **skipped**, and execution proceeds directly to the **end node**.

### Tradeoffs

- Used raw SQL to avoid introducing unnecessary abstraction for a small project. While lightweight and performant, this sacrifices compile-time safety and can be more error-prone. A tool like SQLBoiler would improve maintainability at scale.
- Node execution is currently determined using a hard-coded switch statement based on node.ID. While this approach is straightforward and effective for a limited set of predefined nodes, it couples logic tightly to specific identifiers, making it less adaptable as new node types are introduced.

## Libraries/Tools

No special libraries was used for this task.

## Future Node-Type Extensions

- To support additional node types, define a new node processor function in `services/workflow/node_processor.go`, and update the dispatch logic (currently a `switch` statement) to include the new node's type or ID.
- Workflow definitions are stored as a `JSONB` column in the database, allowing flexibility to represent any node type with varying structures. This also enables efficient querying of nested JSON fields.
- Since the schema is dynamic, it's important to validate the workflow structure **before persisting to the database** (though this is out of scope for the current project). Implementing a [JSON Schema](https://json-schema.org) would provide a contract for what a valid workflow definition should look like and serve as the source of truth for validation.

An example JSON schema for the workflow definition might looks like this:

```
{
  "$schema": "https://json-schema.org/draft/2020-12/schema",
  "title": "WorkflowDefinition",
  "type": "object",
  "required": ["id", "nodes", "edges"],
  "properties": {
    "id": { "type": "string", "format": "uuid" },
    "nodes": {
      "type": "array",
      "items": {
        "type": "object",
        "required": ["id", "type", "position", "data"],
        "properties": {
          "id": { "type": "string" },
          "type": {
            "type": "string",
            "enum": ["start", "form", "integration", "condition", "email", "end"]
          },
          "position": {
            "type": "object",
            "required": ["x", "y"],
            "properties": {
              "x": { "type": "number" },
              "y": { "type": "number" }
            }
          },
          "data": {
            "type": "object",
            "required": ["label", "description", "metadata"],
            "properties": {
              "label": { "type": "string" },
              "description": { "type": "string" },
              "metadata": {
                "type": "object"
              }
            }
          }
        }
      }
    },
    "edges": {
      "type": "array",
      "items": {
        "type": "object",
        "required": ["id", "source", "target"],
        "properties": {
          "id": { "type": "string" },
          "source": { "type": "string" },
          "target": { "type": "string" },
          "label": { "type": "string" },
          "animated": { "type": "boolean" },
          "type": { "type": "string" },
          "sourceHandle": { "type": "string" },
          "style": {
            "type": "object",
            "properties": {
              "stroke": { "type": "string" },
              "strokeWidth": { "type": "number" }
            }
          },
          "labelStyle": {
            "type": "object",
            "properties": {
              "fill": { "type": "string" },
              "fontWeight": { "type": "string" }
            }
          }
        }
      }
    }
  }
}
```

## Testing

- Unit tests exist for the main `processNodes` function as well as some node type processors located in `services/workflow/node_processor_test.go` - Simply execute the unit test from your IDE to run.

## Future Improvements

- Integrate a migration tool like **golang-migrate** for versioned and reversible schema changes.
- Introduce **JSON Schema validation** for workflow definitions to ensure consistency and correctness.
- Use an ORM such as **SQLBoiler** to gain type safety, compile-time checks, and simplify testing.
- Add **unit tests** for database interaction methods to ensure reliability.
- Add **unit tests** for the API layer to verify endpoint behavior.
- Expand **node processing tests** to cover a wider range of scenarios and edge cases.
- Implement **end-to-end testing** (e.g., using Playwright) to validate the full workflow from editor to execution.
