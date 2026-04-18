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

export const GoFixTool = Tool.define("go_fix", {
  description:
    "Automatically fix syntax errors in a Go source file that prevent AST parsing. " +
    "Analyzes parse errors and applies targeted fixes: removes double semicolons, " +
    "splits statements merged onto one line, removes trailing semicolons, fixes " +
    "missing commas, and repairs other common syntax issues. " +
    "Use this ONLY when an edit tool fails with parse errors. Just pass the file path — " +
    "the tool figures out what to fix automatically. After fixing, retry the edit.",
  parameters: z.object({
    filePath: z.string().describe("Absolute path to the .go file with syntax errors"),
  }),
  async execute(params, ctx) {
    const filePath = path.isAbsolute(params.filePath)
      ? params.filePath
      : path.join(Instance.directory, params.filePath)
    await assertExternalDirectory(ctx, filePath)

    if (!filePath.endsWith(".go")) {
      throw new Error("go_fix only works on .go files")
    }

    const exists = await Filesystem.exists(filePath)
    if (!exists) {
      throw new Error(`File not found: ${filePath}`)
    }

    await FileTime.read(ctx.sessionID, filePath)
    await FileTime.assert(ctx.sessionID, filePath)

    const result = await execGoAst({ mode: "fix", file: filePath })

    if (!result.fixesApplied?.length && !result.success) {
      const remaining = result.remainingErrors?.join("\n") ?? "unknown errors"
      throw new Error(
        `go_fix could not automatically fix the syntax errors:\n${remaining}\n\n` +
          "The errors are too complex for automatic repair.",
      )
    }

    const diff = result.diff ?? ""

    if (diff) {
      await ctx.ask({
        permission: "edit",
        patterns: [path.relative(Instance.worktree, filePath)],
        always: ["*"],
        metadata: {
          filepath: filePath,
          diff,
        },
      })
    }

    Bus.publish(File.Event.Edited, { file: filePath })
    await Bus.publish(FileWatcher.Event.Updated, {
      file: filePath,
      event: "change",
    })
    await FileTime.read(ctx.sessionID, filePath)

    await LSP.touchFile(filePath, true)
    const diagnostics = await LSP.diagnostics()

    const fixList = (result.fixesApplied ?? []).map((f: string) => `  - ${f}`).join("\n")
    let output = `Applied ${result.fixesApplied?.length ?? 0} fix(es):\n${fixList}`

    if (result.success) {
      output += "\n\nAll syntax errors fixed. You can now retry the edit operation."
    } else {
      const remaining = (result.remainingErrors ?? []).map((e: string) => `  - ${e}`).join("\n")
      output += `\n\nRemaining parse errors (could not auto-fix):\n${remaining}`
      output += "\n\nCall go_fix again — some fixes may enable further automatic repairs."
    }

    const normalized = Filesystem.normalizePath(filePath)
    const issues = diagnostics[normalized] ?? []
    const errors = issues.filter((item: any) => item.severity === 1)
    if (errors.length > 0) {
      const limited = errors.slice(0, 20)
      const suffix = errors.length > 20 ? `\n... and ${errors.length - 20} more` : ""
      output += `\n\nLSP errors in ${path.relative(Instance.worktree, filePath)}:\n<diagnostics file="${filePath}">\n${limited.map(LSP.Diagnostic.pretty).join("\n")}${suffix}\n</diagnostics>`
    }

    return {
      title: `Fixed ${path.relative(Instance.worktree, filePath)}`,
      metadata: {
        diff,
        diagnostics,
      },
      output,
    }
  },
})
