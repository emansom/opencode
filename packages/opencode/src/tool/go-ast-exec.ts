import { spawn } from "child_process"
import * as path from "path"

const GOAST_DIR = path.resolve(import.meta.dirname, "../../goast")

export interface GoAstInput {
  mode: "inspect" | "edit" | "fix" | "prompt"
  file?: string
  op?: string
  [key: string]: unknown
}

export async function execGoAst(input: GoAstInput): Promise<any> {
  return new Promise((resolve, reject) => {
    const child = spawn("go", ["run", "."], {
      cwd: GOAST_DIR,
      stdio: ["pipe", "pipe", "pipe"],
    })

    let stdout = ""
    let stderr = ""

    child.stdout.on("data", (data: Buffer) => {
      stdout += data.toString()
    })

    child.stderr.on("data", (data: Buffer) => {
      stderr += data.toString()
    })

    child.on("close", (code: number | null) => {
      if (code !== 0 && !stdout) {
        reject(new Error(`Go AST helper exited with code ${code}: ${stderr}`))
        return
      }
      try {
        const result = JSON.parse(stdout)
        if (result.success === false && result.errors?.length) {
          const errMsg = result.errors.join(", ")
          // Detect parse errors — the file has syntax errors that prevent AST parsing.
          // Guide the model to use go_fix which auto-detects and repairs syntax errors.
          if (errMsg.includes("parse file:") || errMsg.includes("expected ") || errMsg.includes("syntax error")) {
            reject(new Error(
              `Go AST error: ${errMsg}\n\n` +
              `The file has syntax errors that prevent AST parsing. ` +
              `Call the go_fix tool with just the filePath — it will automatically detect and fix the syntax errors. ` +
              `Then retry the edit operation.`
            ))
            return
          }
          reject(new Error(`Go AST error: ${errMsg}`))
          return
        }
        resolve(result)
      } catch {
        reject(new Error(`Failed to parse Go AST output: ${stdout}\nstderr: ${stderr}`))
      }
    })

    child.on("error", (err: Error) => {
      reject(new Error(`Failed to spawn Go AST helper: ${err.message}`))
    })

    child.stdin.write(JSON.stringify(input))
    child.stdin.end()
  })
}
