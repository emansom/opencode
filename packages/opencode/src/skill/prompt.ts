/**
 * Skill Prompt Library
 *
 * Builds skill catalog prompts compatible with:
 * - Agent Skills spec (https://agentskills.io/specification) — XML format
 * - Gallery/LiteRT-LM Gemma 4 variant — plain text format with 3-step flow
 *
 * Inspired by the skills-ref reference implementation:
 * https://github.com/agentskills/agentskills/tree/main/skills-ref
 *
 * Implements 3-tier progressive disclosure (per spec):
 * Tier 1 — Catalog: brief name:description in system prompt (~50-100 tokens/skill)
 * Tier 2 — Activation: full instructions loaded on-demand via skill tool (<5000 tokens)
 * Tier 3 — Resources: supporting files loaded via file-read (existing behavior)
 */

// ---------------------------------------------------------------------------
// Types
// ---------------------------------------------------------------------------

/** Skill properties matching Agent Skills spec YAML frontmatter. */
export interface SkillProperties {
  name: string
  description: string
  license?: string
  compatibility?: string
  metadata?: Record<string, string>
  allowedTools?: string
}

/** A skill entry for the Tier 1 catalog. */
export interface SkillEntry {
  name: string
  description: string
  location?: string
  source: "file" | "tool" | "mcp" | "lsp"
}

/** Format variants for the skill catalog. */
export type SkillCatalogFormat = "xml" | "gemma4"

/** Tool definition used to generate a tool-skill. */
export interface ToolSkillSource {
  id: string
  description: string
  parameters: Record<string, unknown>
}

// ---------------------------------------------------------------------------
// Tier 1: Catalog generation
// ---------------------------------------------------------------------------

/**
 * Generate skill catalog block for the system prompt.
 *
 * XML format (official spec):
 *   <available_skills><skill><name>...</name>...</skill></available_skills>
 *
 * Gemma 4 format (Gallery/LiteRT-LM):
 *   - name: description
 */
export function buildSkillCatalog(
  skills: SkillEntry[],
  format: SkillCatalogFormat,
): string {
  if (format === "xml") {
    return buildXmlCatalog(skills)
  }
  return buildGemma4Catalog(skills)
}

function buildXmlCatalog(skills: SkillEntry[]): string {
  if (skills.length === 0) {
    return "<available_skills>\n</available_skills>"
  }
  const entries = skills.map((s) => {
    const loc = s.location ? `\n<location>${escapeXml(s.location)}</location>` : ""
    return `<skill>\n<name>${escapeXml(s.name)}</name>\n<description>${escapeXml(s.description)}</description>${loc}\n</skill>`
  })
  return `<available_skills>\n${entries.join("\n")}\n</available_skills>`
}

function buildGemma4Catalog(skills: SkillEntry[]): string {
  return skills.map((s) => `- ${s.name}: ${s.description}`).join("\n")
}

/** Configuration for the Gemma 4 system prompt. */
export interface Gemma4PromptConfig {
  /** Role/identity line. Defaults to a generic agentic assistant. */
  role?: string
  /** Rules to include after the 3-step flow. Rendered as a bullet list. */
  rules?: string[]
  /** Additional sections inserted between the 3-step flow and rules. Each entry is a block of text. */
  sections?: string[]
}

/**
 * Build the full Gemma 4 system prompt with skill catalog.
 * Matches Gallery's mandatory 3-step flow pattern.
 *
 * The prompt structure is generic — tool-specific rules and identity
 * are supplied by the caller via `config`.
 */
export function buildGemma4SkillPrompt(
  skills: SkillEntry[],
  config?: Gemma4PromptConfig,
): string {
  const lines: string[] = []

  lines.push(config?.role ?? "You are an expert software engineer. You work autonomously on programming tasks by using your tools.")
  lines.push("")
  lines.push("For EVERY new task, you MUST execute these steps in order:")
  lines.push("")
  lines.push("1. Find the most relevant skill from the following list:")
  lines.push("")
  lines.push(buildGemma4Catalog(skills))
  lines.push("")
  lines.push("2. Use the skill tool to load the skill's full instructions.")
  lines.push("")
  lines.push("3. Follow the loaded instructions to complete the task using the appropriate tools.")

  if (config?.sections?.length) {
    for (const section of config.sections) {
      lines.push("")
      lines.push(section)
    }
  }

  if (config?.rules?.length) {
    lines.push("")
    lines.push("Rules:")
    for (const rule of config.rules) {
      lines.push(`- ${rule}`)
    }
  }

  return lines.join("\n")
}

// ---------------------------------------------------------------------------
// Tier 2: Activation content
// ---------------------------------------------------------------------------

/**
 * Generate skill activation content from a tool definition.
 * Returns formatted instructions for the skill tool response.
 */
export function generateToolSkillContent(
  tool: ToolSkillSource,
  opts?: { extraContent?: string },
): string {
  const lines: string[] = []

  lines.push(`# ${tool.id}`)
  lines.push("")
  lines.push(tool.description)
  lines.push("")

  const schema = tool.parameters as Record<string, unknown>
  const properties = schema.properties as Record<string, Record<string, unknown>> | undefined
  const required = new Set((schema.required as string[]) ?? [])

  if (properties && Object.keys(properties).length > 0) {
    lines.push("## Parameters")
    for (const [name, prop] of Object.entries(properties)) {
      const req = required.has(name) ? "required" : "optional"
      const type = (prop.type as string) ?? "any"
      const desc = (prop.description as string) ?? ""
      if (desc) {
        lines.push(`- ${name} (${type}, ${req}): ${desc}`)
      } else {
        lines.push(`- ${name} (${type}, ${req})`)
      }
    }
    lines.push("")
  }

  if (opts?.extraContent) {
    lines.push(opts.extraContent)
    lines.push("")
  }

  lines.push(`Call the ${tool.id} tool with the parameters above.`)

  return lines.join("\n")
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

/** Convert tool definitions to skill entries for the catalog. */
export function toolsToSkillEntries(tools: ToolSkillSource[]): SkillEntry[] {
  return tools.map((t) => ({
    name: t.id,
    description: truncateDescription(t.description),
    source: t.id.includes("/") ? ("mcp" as const) : ("tool" as const),
  }))
}

/** Merge file-skills and tool-skills. File-skills override tool-skills on name collision. */
export function mergeSkillEntries(
  toolSkills: SkillEntry[],
  fileSkills: SkillEntry[],
): SkillEntry[] {
  const fileNames = new Set(fileSkills.map((s) => s.name))
  return [
    ...toolSkills.filter((s) => !fileNames.has(s.name)),
    ...fileSkills,
  ]
}

function truncateDescription(desc: string): string {
  const firstSentence = desc.split(/[.\n]/)[0].trim()
  return firstSentence.length > 80 ? firstSentence.slice(0, 77) + "..." : firstSentence
}

/** HTML-escape for XML format (per spec reference implementation). */
function escapeXml(s: string): string {
  return s
    .replace(/&/g, "&amp;")
    .replace(/</g, "&lt;")
    .replace(/>/g, "&gt;")
    .replace(/"/g, "&quot;")
    .replace(/'/g, "&#39;")
}
