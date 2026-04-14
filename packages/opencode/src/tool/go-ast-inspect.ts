import z from "zod"
import * as path from "path"
import { Tool } from "./tool"
import { Instance } from "../project/instance"
import { assertExternalDirectory } from "./external-directory"
import { execGoAst } from "./go-ast-exec"

export const GoAstInspectTool = Tool.define("go_inspect", {
  description:
    "Inspect the AST structure of a Go source file. Returns the package name, imports, types (structs with fields and tags, interfaces with methods and embeds, named types, aliases), functions (with signatures, parameters, returns, and statement-level body info), constants, variables, build constraints, and generate directives. Use this tool before editing to understand the file structure and identify correct targets.",
  parameters: z.object({
    filePath: z.string().describe("Absolute path to the .go file"),
  }),
  async execute(params, ctx) {
    const filePath = path.isAbsolute(params.filePath) ? params.filePath : path.join(Instance.directory, params.filePath)
    await assertExternalDirectory(ctx, filePath)

    if (!filePath.endsWith(".go")) {
      throw new Error("go_inspect only works on .go files")
    }

    const result = await execGoAst({ mode: "inspect", file: filePath })
    return {
      title: `Inspected ${path.basename(filePath)}`,
      metadata: {},
      output: JSON.stringify(result, null, 2),
    }
  },
})
