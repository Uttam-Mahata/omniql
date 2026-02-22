# Agent Skills

Agent Skills allow you to extend Gemini CLI with specialized expertise,
procedural workflows, and task-specific resources. Based on the
[Agent Skills](https://agentskills.io) open standard, a "skill" is a
self-contained directory that packages instructions and assets into a
discoverable capability.

## Overview

Unlike general context files (`GEMINI.md`), which provide
persistent workspace-wide background, Skills represent **on-demand expertise**.
This allows Gemini to maintain a vast library of specialized capabilities—such
as security auditing, cloud deployments, or codebase migrations—without
cluttering the model's immediate context window.

Gemini autonomously decides when to employ a skill based on your request and the
skill's description. When a relevant skill is identified, the model "pulls in"
the full instructions and resources required to complete the task using the
`activate_skill` tool.

## Key Benefits

- **Shared Expertise:** Package complex workflows (like a specific team's PR
  review process) into a folder that anyone can use.
- **Repeatable Workflows:** Ensure complex multi-step tasks are performed
  consistently by providing a procedural framework.
- **Resource Bundling:** Include scripts, templates, or example data alongside
  instructions so the agent has everything it needs.
- **Progressive Disclosure:** Only skill metadata (name and description) is
  loaded initially. Detailed instructions and resources are only disclosed when
  the model explicitly activates the skill, saving context tokens.

## OmniQL Specialized Skills

The following skills have been developed specifically for the OmniQL project:

| Skill Name | Purpose |
|------------|---------|
| **omniql-driver-scaffold** | Bootstraps new database drivers with templates and boilerplate. |
| **omniql-query-debugger** | Validates and troubleshoots OQL query syntax and translation. |
| **omniql-feature-extender** | Guides implementation of missing OQL features across drivers. |
| **omniql-binding-sync** | Ensures multi-language bindings remain synchronized with Go FFI. |
| **omniql-schema-validator** | Manages collection schemas and type validation rules. |
| **omniql-cli-expert** | Specialized guidance for using the `omniql` CLI tool. |

## How to Use These Skills

To use these skills, you first need to **install them**, and then they will be **automatically activated** by Gemini CLI whenever you ask for a task that matches their specialty.

### 1. Installation (First Step)
Before they can be used, the `.skill` files must be installed. You can do this by running the following command in your terminal:

```bash
gemini skills install ./omniql-driver-scaffold.skill --scope workspace
```

### 2. Automatic Activation
Once installed and your session is reloaded (via `/skills reload`), you don't need to manually "run" a skill. Instead:
- **You ask a question** related to the skill's purpose (e.g., *"How do I add support for Redis?"*).
- **Gemini CLI recognizes the intent** and calls `activate_skill`.
- **You approve the activation** when prompted in the UI.
- **The skill's expert instructions** are loaded into my context, and I perform the task using that specialized knowledge.

### Trigger Examples

| If you want to... | Say something like... | Triggered Skill |
| :--- | :--- | :--- |
| **Add a database** | "I want to create a new driver for Cassandra." | `omniql-driver-scaffold` |
| **Fix a query** | "This OQL query is returning an error, can you help me debug it?" | `omniql-query-debugger` |
| **Add a feature** | "We need to implement the `$or` operator in the Postgres driver." | `omniql-feature-extender` |
| **Update Python/Java** | "I added a new function to the FFI, update the Python binding." | `omniql-binding-sync` |
| **Validate Data** | "Register a schema for the 'orders' table with an 'id' and 'price'." | `omniql-schema-validator` |
| **Run CLI Queries** | "How do I run a count query against MongoDB from the terminal?" | `omniql-cli-expert` |

## Manual Management

### In an Interactive Session

Use the `/skills` slash command to view and manage available expertise:

- `/skills list` (default): Shows all discovered skills and their status.
- `/skills disable <name>`: Stop a skill from being automatically suggested.
- `/skills enable <name>`: Re-enables a disabled skill.
- `/skills reload`: Refreshes the list of discovered skills from all tiers. **Note: Must be run after installation to refresh the list.**

### From the Terminal

The `gemini skills` command provides management utilities:

```bash
# List all discovered skills
gemini skills list

# Install a skill from a zipped skill file (.skill)
gemini skills install ./omniql-driver-scaffold.skill --scope workspace

# Enable a skill
gemini skills enable omniql-driver-scaffold

# Disable a skill
gemini skills disable omniql-driver-scaffold --scope workspace
```

## How it Works

1.  **Discovery**: At the start of a session, Gemini CLI scans the discovery
    tiers and injects the name and description of all enabled skills into the
    system prompt.
2.  **Activation**: When Gemini identifies a task matching a skill's
    description, it calls the `activate_skill` tool.
3.  **Injection**: Upon your approval, the skill's body and resources are added to the conversation history.
4.  **Execution**: The model proceeds with the specialized expertise active.
