import z from "zod"
import { Tool } from "./tool"
import { ToolRegistry } from "./registry"

const Parameters = z.object({
  intent: z.string().describe("The name of the skill to run"),
  parameters: z.string().describe("A JSON string with the skill parameters"),
})

/**
 * Gemma 4-only tool that dispatches to any tool by name with JSON-encoded args.
 * Matches Gallery's `runIntent(intent, parameters)`.
 */
export const RunIntentTool = Tool.define("run_intent", async () => ({
  description: "Execute a skill action.",
  parameters: Parameters,
  async execute(params: z.infer<typeof Parameters>, ctx) {
    const toolDefs = await ToolRegistry.tools({
      providerID: (ctx.extra?.model as any)?.providerID ?? ("" as any),
      modelID: ((ctx.extra?.model as any)?.api?.id ?? "") as any,
      agent: { name: ctx.agent, permission: [] } as any,
    })
    const toolDef = toolDefs.find((t) => t.id === params.intent)
    if (!toolDef) {
      const available = toolDefs
        .filter((t) => t.id !== "load_skill" && t.id !== "run_intent")
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

    const result = await toolDef.execute(args, ctx)
    return {
      title: result.title,
      output: result.output,
      metadata: result.metadata,
    }
  },
}))
