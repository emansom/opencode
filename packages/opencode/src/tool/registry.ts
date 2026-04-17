import { PlanExitTool } from "./plan"
import { QuestionTool } from "./question"
import { BashTool } from "./bash"
import { EditTool } from "./edit"
import { GlobTool } from "./glob"
import { GrepTool } from "./grep"
import { ReadTool } from "./read"
import { TaskDescription, TaskTool } from "./task"
import { TodoWriteTool } from "./todo"
import { WebFetchTool } from "./webfetch"
import { WriteTool } from "./write"
import { LoadSkillTool } from "./load-skill"
import { RunIntentTool } from "./run-intent"
import { Tool } from "./tool"
import { Config } from "../config/config"
import { type ToolContext as PluginToolContext, type ToolDefinition } from "@opencode-ai/plugin"
import z from "zod"
import { Plugin } from "../plugin"
import { ProviderID, type ModelID } from "../provider/schema"
import { WebSearchTool } from "./websearch"
import { CodeSearchTool } from "./codesearch"
import { Flag } from "@/flag/flag"
import { Log } from "@/util/log"
import { LspTool } from "./lsp"
import { Truncate } from "./truncate"
import { ApplyPatchTool } from "./apply_patch"
import { loadGoAstEditTools } from "./go-ast-edit"
import { GoAstInspectTool } from "./go-ast-inspect"
import { GoFixTool } from "./go-fix"
import { Glob } from "../util/glob"
import path from "path"
import { pathToFileURL } from "url"
import { Effect, Layer, ServiceMap } from "effect"
import { InstanceState } from "@/effect/instance-state"
import { makeRuntime } from "@/effect/run-service"
import { Env } from "../env"
import { Question } from "../question"
import { Todo } from "../session/todo"
import { LSP } from "../lsp"
import { FileTime } from "../file/time"
import { Instruction } from "../session/instruction"
import { AppFileSystem } from "../filesystem"
import { Agent } from "../agent/agent"

// Shared state for both ToolRegistry and SkillRegistry
type State = {
  custom: Tool.Def[]
  tools: Tool.Def[]   // Only load_skill, run_intent (FC-declared)
  skills: Tool.Def[]  // Everything else (dispatch targets for run_intent)
}

type ModelInput = {
  providerID: ProviderID
  modelID: ModelID
  agent: Agent.Info
}

interface SharedInterface {
  readonly toolIds: () => Effect.Effect<string[]>
  readonly toolAll: () => Effect.Effect<Tool.Def[]>
  readonly tools: (model: ModelInput) => Effect.Effect<Tool.Def[]>
  readonly skills: (model: ModelInput) => Effect.Effect<Tool.Def[]>
  readonly getSkill: (name: string, model: ModelInput) => Effect.Effect<Tool.Def | undefined>
}

class SharedService extends ServiceMap.Service<SharedService, SharedInterface>()("@opencode/ToolRegistry") {}

const log = Log.create({ service: "tool.registry" })

const sharedLayer: Layer.Layer<
  SharedService,
  never,
  | Config.Service
  | Plugin.Service
  | Question.Service
  | Todo.Service
  | Agent.Service
  | LSP.Service
  | FileTime.Service
  | Instruction.Service
  | AppFileSystem.Service
> = Layer.effect(
  SharedService,
  Effect.gen(function* () {
    const config = yield* Config.Service
    const plugin = yield* Plugin.Service

    const task = yield* TaskTool
    const read = yield* ReadTool
    const question = yield* QuestionTool
    const todo = yield* TodoWriteTool

    const state = yield* InstanceState.make<State>(
      Effect.fn("ToolRegistry.state")(function* (ctx) {
        const custom: Tool.Def[] = []

        function fromPlugin(id: string, def: ToolDefinition): Tool.Def {
          return {
            id,
            parameters: z.object(def.args),
            description: def.description,
            execute: async (args, toolCtx) => {
              const pluginCtx: PluginToolContext = {
                ...toolCtx,
                directory: ctx.directory,
                worktree: ctx.worktree,
              }
              const result = await def.execute(args as any, pluginCtx)
              const out = await Truncate.output(result, {}, await Agent.get(toolCtx.agent))
              return {
                title: "",
                output: out.truncated ? out.content : result,
                metadata: {
                  truncated: out.truncated,
                  outputPath: out.truncated ? out.outputPath : undefined,
                },
              }
            },
          }
        }

        const dirs = yield* config.directories()
        const matches = dirs.flatMap((dir) =>
          Glob.scanSync("{tool,tools}/*.{js,ts}", { cwd: dir, absolute: true, dot: true, symlink: true }),
        )
        if (matches.length) yield* config.waitForDependencies()
        for (const match of matches) {
          const namespace = path.basename(match, path.extname(match))
          const mod = yield* Effect.promise(
            () => import(process.platform === "win32" ? match : pathToFileURL(match).href),
          )
          for (const [id, def] of Object.entries<ToolDefinition>(mod)) {
            custom.push(fromPlugin(id === "default" ? namespace : `${namespace}_${id}`, def))
          }
        }

        const plugins = yield* plugin.list()
        for (const p of plugins) {
          for (const [id, def] of Object.entries(p.tool ?? {})) {
            custom.push(fromPlugin(id, def))
          }
        }

        const cfg = yield* config.get()
        const questionEnabled =
          ["app", "cli", "desktop"].includes(Flag.OPENCODE_CLIENT) || Flag.OPENCODE_ENABLE_QUESTION_TOOL

        const tool = yield* Effect.all({
          bash: Tool.init(BashTool),
          read: Tool.init(read),
          glob: Tool.init(GlobTool),
          grep: Tool.init(GrepTool),
          edit: Tool.init(EditTool),
          write: Tool.init(WriteTool),
          task: Tool.init(task),
          fetch: Tool.init(WebFetchTool),
          todo: Tool.init(todo),
          search: Tool.init(WebSearchTool),
          code: Tool.init(CodeSearchTool),
          loadSkill: Tool.init(LoadSkillTool),
          runIntent: Tool.init(RunIntentTool),
          patch: Tool.init(ApplyPatchTool),
          goAstInspect: Tool.init(GoAstInspectTool),
          goFix: Tool.init(GoFixTool),
          question: Tool.init(question),
          lsp: Tool.init(LspTool),
          plan: Tool.init(PlanExitTool),
        })

        // Load individual Go AST edit tools from the Go helper's operation registry.
        const goAstEditDefs = yield* Effect.promise(() => loadGoAstEditTools())
        const goAstEditTools = yield* Effect.all(
          goAstEditDefs.map((def) => Tool.init(def)),
        )

        return {
          custom,
          tools: [
            tool.loadSkill,
            tool.runIntent,
          ],
          skills: [
            ...(questionEnabled ? [tool.question] : []),
            tool.bash,
            tool.read,
            tool.glob,
            tool.grep,
            tool.edit,
            tool.write,
            tool.task,
            tool.fetch,
            tool.todo,
            tool.search,
            tool.code,
            tool.patch,
            ...goAstEditTools,
            tool.goAstInspect,
            tool.goFix,
            ...(Flag.OPENCODE_EXPERIMENTAL_LSP_TOOL ? [tool.lsp] : []),
            ...(Flag.OPENCODE_EXPERIMENTAL_PLAN_MODE && Flag.OPENCODE_CLIENT === "cli" ? [tool.plan] : []),
          ],
        }
      }),
    )

    // ToolRegistry methods: only load_skill and run_intent
    const toolAll: SharedInterface["toolAll"] = Effect.fn("ToolRegistry.all")(function* () {
      const s = yield* InstanceState.get(state)
      return s.tools as Tool.Def[]
    })

    const toolIds: SharedInterface["toolIds"] = Effect.fn("ToolRegistry.ids")(function* () {
      return (yield* toolAll()).map((t) => t.id)
    })

    const tools: SharedInterface["tools"] = Effect.fn("ToolRegistry.tools")(function* (_input) {
      const s = yield* InstanceState.get(state)
      return yield* Effect.forEach(
        s.tools,
        Effect.fnUntraced(function* (tool: Tool.Def) {
          const output = { description: tool.description, parameters: tool.parameters }
          yield* plugin.trigger("tool.definition", { toolID: tool.id }, output)
          return {
            id: tool.id,
            description: output.description,
            parameters: output.parameters,
            execute: tool.execute,
            formatValidationError: tool.formatValidationError,
          }
        }),
        { concurrency: "unbounded" },
      )
    })

    // SkillRegistry methods: all skill implementations
    const skills: SharedInterface["skills"] = Effect.fn("SkillRegistry.skills")(function* (input) {
      const s = yield* InstanceState.get(state)
      const allSkills = [...s.skills, ...s.custom]

      const filtered = allSkills.filter((tool) => {
        if (tool.id === CodeSearchTool.id || tool.id === WebSearchTool.id) {
          return input.providerID === ProviderID.opencode || Flag.OPENCODE_ENABLE_EXA
        }
        const usePatch =
          !!Env.get("OPENCODE_E2E_LLM_URL") ||
          (input.modelID.includes("gpt-") && !input.modelID.includes("oss") && !input.modelID.includes("gpt-4"))
        if (tool.id === ApplyPatchTool.id) return usePatch
        if (tool.id === EditTool.id || tool.id === WriteTool.id) return !usePatch
        return true
      })

      return yield* Effect.forEach(
        filtered,
        Effect.fnUntraced(function* (tool: Tool.Def) {
          using _ = log.time(tool.id)
          const output = {
            description: tool.description,
            parameters: tool.parameters,
          }
          yield* plugin.trigger("tool.definition", { toolID: tool.id }, output)
          return {
            id: tool.id,
            description: [
              output.description,
              tool.id === TaskTool.id ? yield* TaskDescription(input.agent) : undefined,
            ]
              .filter(Boolean)
              .join("\n"),
            parameters: output.parameters,
            execute: tool.execute,
            formatValidationError: tool.formatValidationError,
          }
        }),
        { concurrency: "unbounded" },
      )
    })

    const getSkill: SharedInterface["getSkill"] = Effect.fn("SkillRegistry.get")(function* (name, _input) {
      const s = yield* InstanceState.get(state)
      return [...s.skills, ...s.custom].find((t) => t.id === name)
    })

    return SharedService.of({ toolIds, toolAll, tools, skills, getSkill })
  }),
)

// =============================================================================
// ToolRegistry — only load_skill and run_intent
// =============================================================================

export namespace ToolRegistry {
  export const Service = SharedService
  export const layer = sharedLayer

  export const defaultLayer = Layer.unwrap(
    Effect.sync(() =>
      sharedLayer.pipe(
        Layer.provide(Config.defaultLayer),
        Layer.provide(Plugin.defaultLayer),
        Layer.provide(Question.defaultLayer),
        Layer.provide(Todo.defaultLayer),
        Layer.provide(Agent.defaultLayer),
        Layer.provide(LSP.defaultLayer),
        Layer.provide(FileTime.defaultLayer),
        Layer.provide(Instruction.defaultLayer),
        Layer.provide(AppFileSystem.defaultLayer),
      ),
    ),
  )

  const { runPromise } = makeRuntime(SharedService, defaultLayer)

  export async function ids() {
    return runPromise((svc) => svc.toolIds())
  }

  export async function tools(input: ModelInput): Promise<(Tool.Def & { id: string })[]> {
    return runPromise((svc) => svc.tools(input))
  }

  export async function skills(input: ModelInput): Promise<(Tool.Def & { id: string })[]> {
    return runPromise((svc) => svc.skills(input))
  }

  export async function get(name: string, input: ModelInput): Promise<(Tool.Def & { id: string }) | undefined> {
    return runPromise((svc) => svc.getSkill(name, input))
  }
}

// =============================================================================
// SkillRegistry — convenience alias for skill access
// =============================================================================

export namespace SkillRegistry {
  export async function skills(input: ModelInput): Promise<(Tool.Def & { id: string })[]> {
    return ToolRegistry.skills(input)
  }

  export async function get(name: string, input: ModelInput): Promise<(Tool.Def & { id: string }) | undefined> {
    return ToolRegistry.get(name, input)
  }
}
