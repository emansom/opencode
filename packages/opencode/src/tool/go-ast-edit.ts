// Go AST editing tools — one tool per operation, generated from the Go helper's
// operation registry at runtime.
//
// Gemma 4 was trained on Google's FC (Function Calling) format with flat tool
// calling structures: one tool per action, each with only its own human-readable
// parameters. Each tool name uniquely identifies one AST operation. Parameters
// are only the ones relevant to that specific operation — no dispatcher "op"
// field, no parameter disambiguation needed.
//
// Tool definitions and system prompt are sourced from the Go helper's "prompt"
// mode — a single source of truth that prevents divergence between what the
// tool supports and what the model is told.

import z from "zod"
import * as path from "path"
import { Tool } from "./tool"
import { LSP } from "../lsp"
import { Bus } from "../bus"
import { File } from "../file"
import { FileWatcher } from "../file/watcher"
import { FileTime } from "../file/time"
import { Filesystem } from "../util/filesystem"
import { Instance } from "../project/instance"
import { assertExternalDirectory } from "./external-directory"
import { execGoAst } from "./go-ast-exec"

// Types matching the Go helper's prompt.go output.
interface ToolParam {
  name: string
  type: string
  description: string
  required: boolean
  enum?: string[]
}

interface ToolInfo {
  name: string
  op: string
  description: string
  params: ToolParam[]
}

interface PromptResult {
  systemPrompt: string
  tools: ToolInfo[]
}

// Cached registry from the Go helper.
let cachedRegistry: PromptResult | null = null

async function getRegistry(): Promise<PromptResult> {
  if (cachedRegistry) return cachedRegistry
  cachedRegistry = await execGoAst({ mode: "prompt" }) as PromptResult
  return cachedRegistry
}

/** Build a Zod schema from the ToolParam array. Always includes filePath. */
function buildSchema(params: ToolParam[]): z.ZodObject<any> {
  const shape: Record<string, z.ZodTypeAny> = {
    filePath: z.string().describe("Absolute path to the .go file"),
  }
  for (const p of params) {
    let field: z.ZodTypeAny
    if (p.type === "number") {
      field = z.number()
    } else if (p.type === "boolean") {
      field = z.boolean()
    } else if (p.enum && p.enum.length > 0) {
      field = z.enum(p.enum as [string, ...string[]])
    } else {
      field = z.string()
    }
    field = field.describe(p.description)
    if (!p.required) {
      field = field.optional()
    }
    shape[p.name] = field
  }
  return z.object(shape)
}

/** Create a Tool.define() for one operation from the registry. */
function defineGoOp(info: ToolInfo) {
  return Tool.define(info.name, {
    description: info.description,
    shortDescription: info.description.split(".")[0],
    parameters: buildSchema(info.params),
    async execute(params: Record<string, unknown>, ctx: any) {
      const filePath = path.isAbsolute(params.filePath as string)
        ? (params.filePath as string)
        : path.join(Instance.directory, params.filePath as string)
      await assertExternalDirectory(ctx, filePath)

      if (!filePath.endsWith(".go")) {
        throw new Error(`${info.name} only works on .go files`)
      }

      let diff = ""
      let diagnosticOutput = ""

      await FileTime.withLock(filePath, async () => {
        await FileTime.assert(ctx.sessionID, filePath)

        const contentOld = await Filesystem.readText(filePath)

        const input: Record<string, unknown> = {
          mode: "edit",
          file: filePath,
          op: info.op,
        }
        for (const [key, val] of Object.entries(params)) {
          if (key !== "filePath" && val !== undefined) {
            input[key] = val
          }
        }

        const result = await execGoAst(input as any)

        diff = result.diff || ""
        const content = result.content || contentOld

        if (content === contentOld) return

        await ctx.ask({
          permission: "edit",
          patterns: [path.relative(Instance.worktree, filePath)],
          always: ["*"],
          metadata: {
            filepath: filePath,
            diff,
          },
        })

        await Filesystem.write(filePath, content)
        Bus.publish(File.Event.Edited, { file: filePath })
        await Bus.publish(FileWatcher.Event.Updated, {
          file: filePath,
          event: "change",
        })
        await FileTime.read(ctx.sessionID, filePath)
      })

      if (!diff) {
        return {
          title: `No changes to ${path.basename(filePath)}`,
          metadata: { diff: "" },
          output: "No changes were made.",
        }
      }

      try {
        await LSP.touchFile(filePath, true)
        const diagnostics = await LSP.diagnostics()
        const normalizedPath = Filesystem.normalizePath(filePath)
        const issues = diagnostics[normalizedPath] ?? []
        const errors = issues.filter((d) => d.severity === 1)
        if (errors.length > 0) {
          diagnosticOutput =
            "\n\nLSP errors detected, please fix:\n" +
            errors
              .slice(0, 20)
              .map(LSP.Diagnostic.pretty)
              .join("\n")
        }
      } catch {
        // LSP not available
      }

      return {
        title: `Edited ${path.basename(filePath)} (${info.op})`,
        metadata: { diff },
        output: diff + diagnosticOutput,
      }
    },
  })
}

/**
 * Load tool definitions from the Go helper's operation registry and return
 * them as Tool.define() results ready for Tool.init().
 */
export async function loadGoAstEditTools(): Promise<ReturnType<typeof Tool.define>[]> {
  const registry = await getRegistry()
  return registry.tools.map(defineGoOp)
}

/** Load the system prompt from the Go helper's operation registry. */
export async function loadGoAstSystemPrompt(): Promise<string> {
  const registry = await getRegistry()
  return registry.systemPrompt
}
