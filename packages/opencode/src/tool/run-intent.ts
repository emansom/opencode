import z from "zod"
import { Tool } from "./tool"
import { SkillRegistry } from "./registry"

const Parameters = z.object({
  intent: z.string().describe("The name of the skill to run"),
  parameters: z.string().describe("A JSON string with the skill parameters"),
})

/**
 * Tool that dispatches to any skill by name with JSON-encoded args.
 * Matches Gallery's `runIntent(intent, parameters)`.
 */
export const RunIntentTool = Tool.define("run_intent", async () => ({
  description: "Execute a skill action.",
  parameters: Parameters,
  async execute(params: z.infer<typeof Parameters>, ctx) {
    const skillDefs = await SkillRegistry.skills({
      providerID: (ctx.extra?.model as any)?.providerID ?? ("" as any),
      modelID: ((ctx.extra?.model as any)?.api?.id ?? "") as any,
      agent: { name: ctx.agent, permission: [] } as any,
    })
    const skillDef = skillDefs.find((t) => t.id === params.intent)
    if (!skillDef) {
      const available = skillDefs
        .map((t) => t.id)
        .join(", ")
      throw new Error(
        `Tool "${params.intent}" not found. Available: ${available || "none"}`,
      )
    }

    let args: Record<string, unknown>
    try {
      args = JSON.parse(params.parameters)
    } catch {
      throw new Error(
        `Invalid JSON in parameters: ${params.parameters}`,
      )
    }

    let result
    try {
      result = await skillDef.execute(args, ctx)
    } catch (err) {
      if (err instanceof Error && err.message.includes("invalid arguments")) {
        const schema = (skillDef as any).rawJsonSchema
          ?? (z.toJSONSchema(skillDef.parameters) as Record<string, unknown>)
        const props = (schema.properties as Record<string, { description?: string; type?: string }>) ?? {}
        const required = new Set((schema.required as string[]) ?? [])
        const hint = Object.entries(props)
          .map(([k, v]) => `  - ${k} (${v.type ?? "string"}${required.has(k) ? ", required" : ", optional"}): ${v.description ?? ""}`)
          .join("\n")
        throw new Error(
          `${err.message}\n\nCorrect parameters for "${params.intent}":\n${hint || "  (none)"}\n\nTip: call load_skill("${params.intent}") to see full instructions before using run_intent.`,
        )
      }
      throw err
    }
    return {
      title: result.title,
      output: result.output,
      metadata: result.metadata,
    }
  },
}))
