import z from "zod"
import { Tool } from "./tool"
import { ToolRegistry } from "./registry"
import { generateToolSkillContent, type ToolSkillSource } from "../skill/prompt"

const Parameters = z.object({
  name: z.string().describe("The name of the skill to load"),
})

/**
 * Gemma 4-only tool that loads skill instructions from tool definitions at runtime.
 * Matches Gallery's `loadSkill` — returns YAML frontmatter + ## Instructions body.
 * All skills are generated dynamically from the tool registry; no file-based SKILL.md.
 */
export const LoadSkillTool = Tool.define("load_skill", async () => ({
  description: "Load a skill's instructions.",
  parameters: Parameters,
  async execute(params: z.infer<typeof Parameters>, ctx) {
    const toolDefs = await ToolRegistry.tools({
      providerID: (ctx.extra?.model as any)?.providerID ?? ("" as any),
      modelID: ((ctx.extra?.model as any)?.api?.id ?? "") as any,
      agent: { name: ctx.agent, permission: [] } as any,
    })
    const toolDef = toolDefs.find((t) => t.id === params.name)
    if (!toolDef) {
      const available = toolDefs
        .filter((t) => t.id !== "load_skill" && t.id !== "run_intent")
        .map((t) => t.id)
        .join(", ")
      throw new Error(
        `Skill "${params.name}" not found. Available skills: ${available || "none"}`,
      )
    }

    const schema = z.toJSONSchema(toolDef.parameters)
    const source: ToolSkillSource = {
      id: toolDef.id,
      description: toolDef.description,
      parameters: schema as Record<string, unknown>,
    }

    // For Go AST tools, include the operation reference
    let extraContent: string | undefined
    if (toolDef.id.startsWith("go_")) {
      try {
        const { generateGoAstSkillContent } = await import(
          "../provider/gemma4-go-ast-knowledge"
        )
        extraContent = generateGoAstSkillContent(toolDef.id)
      } catch {
        // Go AST knowledge not available — skip
      }
    }

    const content = generateToolSkillContent(source, { extraContent })
    return {
      title: `Loaded skill: ${params.name}`,
      output: content,
      metadata: { name: params.name },
    }
  },
}))
