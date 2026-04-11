import { Ripgrep } from "../file/ripgrep"

import { Instance } from "../project/instance"

import PROMPT_ANTHROPIC from "./prompt/anthropic.txt"
import PROMPT_DEFAULT from "./prompt/default.txt"
import PROMPT_BEAST from "./prompt/beast.txt"
import PROMPT_GEMINI from "./prompt/gemini.txt"
import PROMPT_GPT from "./prompt/gpt.txt"
import PROMPT_KIMI from "./prompt/kimi.txt"

import PROMPT_CODEX from "./prompt/codex.txt"
import PROMPT_TRINITY from "./prompt/trinity.txt"
import type { Provider } from "@/provider/provider"
import type { Agent } from "@/agent/agent"
import { Permission } from "@/permission"
import { Skill } from "@/skill"
import { existsSync, readdirSync } from "fs"
import { join } from "path"
import { generateGoAstSystemPrompt } from "@/provider/gemma4-go-ast-knowledge"

export function isGemma4(model: Provider.Model): boolean {
  const id = model.api.id.toLowerCase()
  return id.includes("gemma-4") || id.includes("gemma4")
}

export function isSmallGemma4(model: Provider.Model): boolean {
  const id = model.api.id.toLowerCase()
  return isGemma4(model) && /gemma.?4.*(e[24]b|[24]b)/i.test(id)
}

function truncateDescription(desc: string): string {
  const firstSentence = desc.split(/[.\n]/)[0].trim()
  return firstSentence.length > 80 ? firstSentence.slice(0, 77) + "..." : firstSentence
}

export function buildGemma4SystemPrompt(
  builtinTools: { id: string; description?: string; shortDescription?: string; shortHint?: string }[],
  mcpTools: Record<string, { description?: string }>,
): string {
  const lines: string[] = []

  lines.push("You are an expert software engineer running inside OpenCode, an agentic code editor.")
  lines.push("You work autonomously on programming tasks by using your tools.")
  lines.push("")

  lines.push("You operate in one of two modes:")
  lines.push("- BUILD mode: Execute tasks autonomously. Use all tools to implement, test, and verify changes.")
  lines.push("- PLAN mode: Form a plan without editing files. Only use read-only tools. Call the plan_exit tool when the plan is complete.")
  lines.push("The current mode is indicated before each conversation turn.")
  lines.push("")

  lines.push("You have tools. Each item listed below is a tool you can call.")
  lines.push("To accomplish tasks, call these tools by name with the required parameters.")
  lines.push("Do not describe what you would do — call the tool and do it.")
  lines.push("")

  const hasLsp = builtinTools.some((t) => t.id === "lsp")
  const hasMcp = Object.keys(mcpTools).length > 0

  lines.push("Your tools:")
  for (const tool of builtinTools) {
    const hint = tool.shortHint ?? tool.shortDescription ?? truncateDescription(tool.description ?? "")
    lines.push(`- ${tool.id}: ${hint}`)
  }
  lines.push("")

  if (hasMcp) {
    lines.push("Additional tools from MCP servers:")
    for (const [id, tool] of Object.entries(mcpTools)) {
      const desc = tool.description || "No description provided"
      lines.push(`- ${id}: ${truncateDescription(desc)}`)
    }
    lines.push("")
  }

  lines.push("Task tracking:")
  lines.push("- Call the todowrite tool to create a task list at the start of every task.")
  lines.push("- Call the todowrite tool to update todo status (in_progress/completed) as you complete each step.")
  lines.push("- The user sees your todo progress in the UI.")
  lines.push("")

  let step = 0
  lines.push("Workflow:")
  lines.push(`${++step}. Understand the task. Call the question tool if truly ambiguous.`)
  lines.push(`${++step}. Call the todowrite tool to create todos listing your planned steps.`)
  lines.push(`${++step}. Call the glob tool and grep tool to find relevant files and patterns.`)
  lines.push(`${++step}. Call the read tool to read the files you need to understand.`)
  if (hasLsp) {
    lines.push(`${++step}. Call the lsp tool to understand the code: find symbol definitions, references, types, and call hierarchies before making changes.`)
  }
  lines.push(`${++step}. Call the edit tool for targeted changes, the write tool for new files, the bash tool for commands.`)
  lines.push(`${++step}. Call the bash tool to run tests, lint, or typecheck to verify changes.`)
  lines.push(`${++step}. If something fails, diagnose the error and fix it. Do not give up.`)
  lines.push(`${++step}. Call the todowrite tool to update todos as you complete each step.`)
  lines.push(`${++step}. Report concisely what you did.`)
  lines.push("")

  lines.push("Rules:")
  if (hasLsp) {
    lines.push("- Always call the read tool and the lsp tool before calling the edit tool. Understand the code before modifying it.")
  } else {
    lines.push("- Always call the read tool before calling the edit tool. Understand the code before modifying it.")
  }
  lines.push("- Prefer the edit tool over the write tool for existing files. Only create files when necessary.")
  lines.push("- Make minimum changes. Do not refactor unrelated code.")
  lines.push("- Follow existing code style, naming conventions, and patterns.")
  lines.push("- Call the glob tool, grep tool, and read tool instead of the bash tool for file operations.")
  lines.push("- Call multiple independent tools in parallel when possible.")
  lines.push("- Never force-push, never commit secrets, never run destructive git commands via the bash tool.")
  if (hasLsp) {
    lines.push("- Call the lsp tool for code navigation: find definitions, references, and symbols.")
  }
  lines.push("- Call the task tool to delegate complex subtasks to subagents.")
  lines.push("- Be concise. Lead with the answer, not the reasoning.")

  // Append Go AST knowledge docs when Go files are present in the workspace.
  // Check go.mod at worktree root, working directory, and one level deep.
  // Also check for .go files in the working directory.
  if (detectGoProject()) {
    lines.push("")
    lines.push(generateGoAstSystemPrompt())
  }

  return lines.join("\n")
}

function hasGoFilesInDir(dir: string, depth: number): boolean {
  if (depth < 0) return false
  try {
    for (const entry of readdirSync(dir, { withFileTypes: true })) {
      if (entry.name.startsWith(".") || entry.name === "node_modules" || entry.name === "vendor") continue
      if (entry.isFile() && (entry.name === "go.mod" || entry.name.endsWith(".go"))) return true
      if (entry.isDirectory() && depth > 0 && hasGoFilesInDir(join(dir, entry.name), depth - 1)) return true
    }
  } catch {
    // Permission denied or similar — skip
  }
  return false
}

function detectGoProject(): boolean {
  // Check working directory and worktree for go.mod or .go files,
  // scanning up to 3 levels deep (handles monorepos).
  if (hasGoFilesInDir(Instance.directory, 3)) return true
  // Also check worktree root if different, but skip "/" (non-git projects)
  const wt = Instance.worktree
  if (wt !== Instance.directory && wt !== "/" && hasGoFilesInDir(wt, 3)) return true
  return false
}

export function getGemma4ModeIndicator(mode: "plan" | "build"): string {
  return mode === "plan"
    ? "[PLAN MODE ACTIVE] Do not edit files. Form a plan and call the plan_exit tool when done."
    : "[BUILD MODE ACTIVE] Execute autonomously."
}

export namespace SystemPrompt {
  export function provider(model: Provider.Model) {
    if (isGemma4(model)) {
      // Gemma 4 uses a dynamic system prompt built at call time with resolved tools.
      // Return a minimal placeholder here; the full prompt is built in llm.ts
      // when tools are available via buildGemma4SystemPrompt().
      return [
        "You are an expert software engineer running inside OpenCode, an agentic code editor. "
        + "You work autonomously on programming tasks by using your tools."
      ]
    }
    if (model.api.id.includes("gpt-4") || model.api.id.includes("o1") || model.api.id.includes("o3"))
      return [PROMPT_BEAST]
    if (model.api.id.includes("gpt")) {
      if (model.api.id.includes("codex")) {
        return [PROMPT_CODEX]
      }
      return [PROMPT_GPT]
    }
    if (model.api.id.includes("gemini-")) return [PROMPT_GEMINI]
    if (model.api.id.includes("claude")) return [PROMPT_ANTHROPIC]
    if (model.api.id.toLowerCase().includes("trinity")) return [PROMPT_TRINITY]
    if (model.api.id.toLowerCase().includes("kimi")) return [PROMPT_KIMI]
    return [PROMPT_DEFAULT]
  }

  export async function environment(model: Provider.Model) {
    const project = Instance.project
    return [
      [
        `You are powered by the model named ${model.api.id}. The exact model ID is ${model.providerID}/${model.api.id}`,
        `Here is some useful information about the environment you are running in:`,
        `<env>`,
        `  Working directory: ${Instance.directory}`,
        `  Workspace root folder: ${Instance.worktree}`,
        `  Is directory a git repo: ${project.vcs === "git" ? "yes" : "no"}`,
        `  Platform: ${process.platform}`,
        `  Today's date: ${new Date().toDateString()}`,
        `</env>`,
        `<directories>`,
        `  ${
          project.vcs === "git" && false
            ? await Ripgrep.tree({
                cwd: Instance.directory,
                limit: 50,
              })
            : ""
        }`,
        `</directories>`,
      ].join("\n"),
    ]
  }

  export async function skills(agent: Agent.Info) {
    if (Permission.disabled(["skill"], agent.permission).has("skill")) return

    const list = await Skill.available(agent)

    return [
      "Skills provide specialized instructions and workflows for specific tasks.",
      "Use the skill tool to load a skill when a task matches its description.",
      // the agents seem to ingest the information about skills a bit better if we present a more verbose
      // version of them here and a less verbose version in tool description, rather than vice versa.
      Skill.fmt(list, { verbose: true }),
    ].join("\n")
  }
}
