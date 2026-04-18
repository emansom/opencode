import path from "path"
import { pathToFileURL } from "url"
import z from "zod"
import { Tool } from "./tool"
import { SkillRegistry } from "./registry"
import { generateToolSkillContent, type ToolSkillSource } from "../skill/prompt"
import { Skill } from "../skill"
import { Ripgrep } from "../file/ripgrep"
import { iife } from "@/util/iife"

const Parameters = z.object({
  name: z.string().describe("The name of the skill to load"),
})

/**
 * Tool that loads skill instructions from tool definitions at runtime.
 * Matches Gallery's `loadSkill` — returns YAML frontmatter + ## Instructions body.
 *
 * Checks file-based SKILL.md first, then falls back to runtime generation
 * from SkillRegistry.
 */
export const LoadSkillTool = Tool.define("load_skill", async () => ({
  description: "Load a skill's instructions.",
  parameters: Parameters,
  async execute(params: z.infer<typeof Parameters>, ctx) {
    // 1. Check file-based SKILL.md first
    const fileSkill = await Skill.get(params.name)
    if (fileSkill) {
      await ctx.ask({
        permission: "skill",
        patterns: [params.name],
        always: [params.name],
        metadata: {},
      })

      const dir = path.dirname(fileSkill.location)
      const base = pathToFileURL(dir).href

      const limit = 10
      const files = await iife(async () => {
        const arr = []
        for await (const file of Ripgrep.files({
          cwd: dir,
          follow: false,
          hidden: true,
          signal: ctx.abort,
        })) {
          if (file.includes("SKILL.md")) {
            continue
          }
          arr.push(path.resolve(dir, file))
          if (arr.length >= limit) {
            break
          }
        }
        return arr
      }).then((f) => f.map((file) => `<file>${file}</file>`).join("\n"))

      return {
        title: `Loaded skill: ${fileSkill.name}`,
        output: [
          `<skill_content name="${fileSkill.name}">`,
          `# Skill: ${fileSkill.name}`,
          "",
          fileSkill.content.trim(),
          "",
          `Base directory for this skill: ${base}`,
          "Relative paths in this skill (e.g., scripts/, reference/) are relative to this base directory.",
          "Note: file list is sampled.",
          "",
          "<skill_files>",
          files,
          "</skill_files>",
          "</skill_content>",
        ].join("\n"),
        metadata: {
          name: fileSkill.name,
          dir,
        },
      }
    }

    // 2. Fall back to runtime generation from SkillRegistry
    const skillDefs = await SkillRegistry.skills({
      providerID: (ctx.extra?.model as any)?.providerID ?? ("" as any),
      modelID: ((ctx.extra?.model as any)?.api?.id ?? "") as any,
      agent: { name: ctx.agent, permission: [] } as any,
    })
    const skillDef = skillDefs.find((t) => t.id === params.name)
    if (!skillDef) {
      const available = skillDefs
        .map((t) => t.id)
        .join(", ")
      throw new Error(
        `Skill "${params.name}" not found. Available skills: ${available || "none"}`,
      )
    }

    const schema = skillDef.rawJsonSchema ?? (z.toJSONSchema(skillDef.parameters) as Record<string, unknown>)
    const source: ToolSkillSource = {
      id: skillDef.id,
      description: skillDef.description,
      parameters: schema,
    }

    // For Go AST tools, include the operation reference
    let extraContent: string | undefined
    if (skillDef.id.startsWith("go_")) {
      try {
        const { generateGoAstSkillContent } = await import(
          "../provider/gemma4-go-ast-knowledge"
        )
        extraContent = generateGoAstSkillContent(skillDef.id)
      } catch {
        // Go AST knowledge not available — skip
      }
    }

    const content = generateToolSkillContent(source, { extraContent })
    return {
      title: `Loaded skill: ${params.name}`,
      output: content,
      metadata: { name: params.name, dir: process.cwd() },
    }
  },
}))
