import { Ripgrep } from "../file/ripgrep"

import { Instance } from "../project/instance"

import type { Provider } from "@/provider/provider"
import type { Agent } from "@/agent/agent"
import { Permission } from "@/permission"
import { buildSkillPrompt, type SkillEntry } from "@/skill/prompt"

export function buildSkillSystemPrompt(
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

  return buildSkillPrompt(skills, {
    role: "You are an expert software engineer running inside OpenCode, an agentic code editor.\nYou work autonomously on programming tasks by using your skills.",
    rules,
  })
}

export function getModeIndicator(mode: "plan" | "build"): string {
  return mode === "plan"
    ? "[PLAN MODE ACTIVE] Do not edit files. Form a plan and call the plan_exit tool when done."
    : "[BUILD MODE ACTIVE] Execute autonomously."
}

export namespace SystemPrompt {
  export function provider(_model: Provider.Model) {
    return [
      "You are an expert software engineer running inside OpenCode, an agentic code editor. "
      + "You work autonomously on programming tasks by using your tools."
    ]
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

  export async function skills(_agent: Agent.Info) {
    // Skill catalog is built in llm.ts from SkillRegistry — no file-based skills here
    return undefined
  }
}
