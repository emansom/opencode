/**
 * Skill Prompt Library
 *
 * Skill Prompt Library
 *
 * Builds skill catalog prompts and activation content for the universal
 * skill-only architecture (load_skill + run_intent).
 *
 * Tier 1 — Catalog: brief name:description in system prompt
 * Tier 2 — Activation: full instructions loaded on-demand via load_skill
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
export type SkillCatalogFormat = "xml" | "plain"

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
 * Plain format:
 *   - name: description
 */
export function buildSkillCatalog(
  skills: SkillEntry[],
  format: SkillCatalogFormat,
): string {
  if (format === "xml") {
    return buildXmlCatalog(skills)
  }
  return buildPlainCatalog(skills)
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

function buildPlainCatalog(skills: SkillEntry[]): string {
  return skills.map((s) => `- ${s.name}: ${s.description}`).join("\n")
}

/** Configuration for the skill-based system prompt. */
export interface SkillPromptConfig {
  /** Role/identity line. Defaults to a generic agentic assistant. */
  role?: string
  /** Rules to include after the 3-step flow. Rendered as a bullet list. */
  rules?: string[]
  /** Additional sections inserted between the 3-step flow and rules. Each entry is a block of text. */
  sections?: string[]
}

/**
 * Build the system prompt with skill catalog and 3-step flow.
 * The model must: find skill → load_skill → run_intent.
 */
export function buildSkillPrompt(
  skills: SkillEntry[],
  config?: SkillPromptConfig,
): string {
  const lines: string[] = []

  lines.push(config?.role ?? "You are an expert software engineer. You work autonomously on programming tasks by using your skills.")
  lines.push("")
  lines.push("You have two direct tools: `load_skill` and `run_intent`. Everything else is a skill — accessed only through these two tools.")
  lines.push("")
  lines.push("For EVERY new task, you MUST execute the following steps in exact order:")
  lines.push("")
  lines.push("1. First, find the most relevant skill from the following list:")
  lines.push("")
  lines.push(buildPlainCatalog(skills))
  lines.push("")
  lines.push("After this step you MUST go to next step. You MUST NOT use `run_intent` at this step.")
  lines.push("")
  lines.push("2. Use the `load_skill` tool to load the skill's full instructions. You MUST NOT use `run_intent` at this step.")
  lines.push("")
  lines.push("3. Follow the skill's instructions exactly to complete the task.")
  lines.push("")
  lines.push("Do NOT call run_intent without loading the skill first.")

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
 * Matches Gallery's loadSkill return format: YAML frontmatter + instructions body.
 *
 * Gallery reconstructs: "---\nname: ${name}\ndescription: ${desc}\n---\n\n${instructions}"
 */
export function generateToolSkillContent(
  tool: ToolSkillSource,
  opts?: { extraContent?: string },
): string {
  const lines: string[] = []

  // YAML frontmatter (matches Gallery's loadSkill return format)
  lines.push("---")
  lines.push(`name: ${tool.id}`)
  lines.push(`description: ${truncateDescription(tool.description)}`)
  lines.push("---")
  lines.push("")

  // Title + description
  lines.push(`# ${tool.id}`)
  lines.push("")
  lines.push(tool.description)
  lines.push("")

  // Instructions section (Gallery SKILL.md format)
  lines.push("## Instructions")
  lines.push("")
  lines.push("Call the `run_intent` tool with the following exact parameters:")
  lines.push("")
  lines.push(`- intent: ${tool.id}`)

  // Parameter fields from schema
  const schema = tool.parameters as Record<string, unknown>
  const properties = schema.properties as Record<string, Record<string, unknown>> | undefined
  const required = new Set((schema.required as string[]) ?? [])

  if (properties && Object.keys(properties).length > 0) {
    lines.push("- parameters: A JSON string with the following fields:")
    for (const [name, prop] of Object.entries(properties)) {
      const req = required.has(name) ? "Required" : "Optional"
      const type = (prop.type as string) ?? "any"
      const desc = (prop.description as string) ?? ""
      lines.push(`  - ${name}: ${desc ? desc + " " : ""}${capitalize(type)}. ${req}.`)
    }
  }

  // Extra content (e.g. Go AST reference)
  if (opts?.extraContent) {
    lines.push("")
    lines.push("## Reference")
    lines.push("")
    lines.push(opts.extraContent)
  }

  return lines.join("\n")
}

function capitalize(s: string): string {
  return s.charAt(0).toUpperCase() + s.slice(1)
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
