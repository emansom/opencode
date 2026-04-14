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
import { buildGemma4SkillPrompt, type SkillEntry } from "@/skill/prompt"

export function isGemma4(model: Provider.Model): boolean {
  const id = model.api.id.toLowerCase()
  return id.includes("gemma-4") || id.includes("gemma4")
}

export function isSmallGemma4(model: Provider.Model): boolean {
  const id = model.api.id.toLowerCase()
  return isGemma4(model) && /gemma.?4.*(e[24]b|[24]b)/i.test(id)
}

export function buildGemma4SystemPrompt(
  skills: SkillEntry[],
): string {
  const rules: string[] = []

  rules.push("Always load the read skill before editing. Understand the code first.")
  rules.push("Prefer the edit skill over the write skill for existing files.")
  rules.push("Make minimum changes. Do not refactor unrelated code.")
  rules.push("Follow existing code style, naming conventions, and patterns.")
  rules.push("Use the glob, grep, and read skills instead of bash for file operations.")
  rules.push("Never force-push, never commit secrets, never run destructive git commands.")
  rules.push("Be concise. Lead with the answer, not the reasoning.")

  return buildGemma4SkillPrompt(skills, {
    role: "You are an expert software engineer running inside OpenCode, an agentic code editor.\nYou work autonomously on programming tasks by using your skills.",
    rules,
  })
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
