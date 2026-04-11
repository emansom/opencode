/**
 * Gemma 4 tool format utilities.
 *
 * - Token constants and wrapToolCall() produce native-format examples
 *   for system prompt knowledge documents (Gemma 4-specific).
 * - flattenToolSchema() / expandToolCallArgs() flatten nested tool
 *   parameters to basic types (model-independent, always active).
 *
 * This module does NOT build full prompts, format tool declarations
 * for the model, or parse model output — the Jinja template and
 * PEG parser handle those server-side.
 */

// ---------------------------------------------------------------------------
// Token constants matching LiteRT-LM's Gemma4DataProcessorConfig
// ---------------------------------------------------------------------------

export const Gemma4Tokens = {
  toolCallStart: "<|tool_call>",
  toolCallEnd: "<tool_call|>",
  quote: '<|"|>',
} as const

// ---------------------------------------------------------------------------
// Value formatting for wrapToolCall() internals
// ---------------------------------------------------------------------------

export function formatValue(value: unknown): string {
  if (value === null || value === undefined) return "null"
  if (typeof value === "string") return `${Gemma4Tokens.quote}${value}${Gemma4Tokens.quote}`
  if (typeof value === "number" || typeof value === "boolean") return String(value)
  if (Array.isArray(value)) return `[${value.map(formatValue).join(",")}]`
  if (typeof value === "object") {
    const entries = Object.entries(value as Record<string, unknown>)
    const parts = entries.map(([k, v]) => `${k}:${formatValue(v)}`)
    return `{${parts.join(",")}}`
  }
  return String(value)
}

// ---------------------------------------------------------------------------
// wrapToolCall — format a tool call example in native Gemma 4 format.
// Used by gemma4-go-ast-knowledge.ts generators to produce examples
// in system prompt knowledge documents.
// ---------------------------------------------------------------------------

export function wrapToolCall(name: string, args: Record<string, unknown>): string {
  const entries = Object.entries(args)
  const parts = entries.map(([k, v]) => `${k}:${formatValue(v)}`)
  return `${Gemma4Tokens.toolCallStart}call:${name}{${parts.join(",")}}${Gemma4Tokens.toolCallEnd}`
}

// ---------------------------------------------------------------------------
// Schema flattening — model-independent, always active.
// Converts nested object/array parameters to basic types so the server
// never sees nested arguments in tool definitions.
// ---------------------------------------------------------------------------

// eslint-disable-next-line @typescript-eslint/no-explicit-any
export type JsonSchema = Record<string, any>

export type FlattenEntry =
  | { kind: "object"; prefix: string; children: string[] }
  | { kind: "array_of_primitives"; key: string; itemType: string }
  | { kind: "array_of_objects"; key: string; childKeys: string[] }

export type FlattenMap = Map<string, FlattenEntry>

/**
 * Resolve the effective type of a JSON Schema node, handling anyOf/oneOf
 * nullable unions like `{ anyOf: [{type:"string"}, {type:"null"}] }`.
 */
function resolveSchemaType(schema: JsonSchema): string | undefined {
  if (schema.type) return schema.type as string
  const union = (schema.anyOf ?? schema.oneOf) as JsonSchema[] | undefined
  if (union) {
    const nonNull = union.filter((s) => s.type !== "null")
    if (nonNull.length === 1) return nonNull[0]!.type as string | undefined
  }
  return undefined
}

/**
 * Flatten a JSON Schema's top-level properties so every parameter is a
 * basic type (string, number, boolean). Nested objects get their
 * properties promoted with underscore-separated prefixes. Arrays become
 * comma-separated or pipe-separated strings.
 *
 * Returns the flattened schema and a FlattenMap describing how to reverse.
 */
export function flattenToolSchema(schema: JsonSchema): {
  flattenedSchema: JsonSchema
  flattenMap: FlattenMap
} {
  const properties = (schema.properties ?? {}) as Record<string, JsonSchema>
  const required = new Set((schema.required ?? []) as string[])
  const flatProps: Record<string, JsonSchema> = {}
  const flatRequired: string[] = []
  const flattenMap: FlattenMap = new Map()

  for (const [key, prop] of Object.entries(properties)) {
    const type = resolveSchemaType(prop)

    if (type === "object" && prop.properties) {
      // Promote child properties with prefix: parent_child
      const childProps = prop.properties as Record<string, JsonSchema>
      const childRequired = new Set((prop.required ?? []) as string[])
      const children: string[] = []
      for (const [ck, cv] of Object.entries(childProps)) {
        const flatKey = `${key}_${ck}`
        children.push(ck)
        flatProps[flatKey] = { ...cv }
        if (required.has(key) && childRequired.has(ck)) {
          flatRequired.push(flatKey)
        }
      }
      flattenMap.set(key, { kind: "object", prefix: key, children })
    } else if (type === "array") {
      const items = prop.items as JsonSchema | undefined
      if (!items) {
        // Unknown array — serialize as JSON string
        flatProps[key] = { type: "string", description: `${prop.description ?? key} (JSON array)` }
        flattenMap.set(key, { kind: "array_of_primitives", key, itemType: "json" })
      } else {
        const itemType = resolveSchemaType(items)
        if (itemType === "object" && items.properties) {
          // Array of objects → each child property becomes a pipe-separated string.
          // Items within each object are comma-separated. Objects are pipe-separated.
          // e.g. questions[0].options = [{label:"A",description:"a"},{label:"B",description:"b"}]
          //   → questions_options_label = "A,B"  questions_options_description = "a,b"
          // For outer array: questions[0].question="Q1" questions[1].question="Q2"
          //   → questions_question = "Q1|Q2"
          const childProps = items.properties as Record<string, JsonSchema>
          const childKeys: string[] = []
          for (const [ck, cv] of Object.entries(childProps)) {
            const flatKey = `${key}_${ck}`
            const childType = resolveSchemaType(cv as JsonSchema)
            childKeys.push(ck)
            if (childType === "array" && (cv as JsonSchema).items) {
              // Nested array inside array item (e.g. questions[].options[])
              const innerItems = (cv as JsonSchema).items as JsonSchema
              const innerType = resolveSchemaType(innerItems)
              if (innerType === "object" && innerItems.properties) {
                // Array of objects inside array of objects
                // Flatten inner object properties too
                const innerProps = innerItems.properties as Record<string, JsonSchema>
                for (const [ik, iv] of Object.entries(innerProps)) {
                  const innerFlatKey = `${key}_${ck}_${ik}`
                  flatProps[innerFlatKey] = {
                    type: "string",
                    description: `${(iv as JsonSchema).description ?? ik} (per item comma-separated, items pipe-separated)`,
                  }
                }
                // Don't add the intermediate key
                childKeys.pop()
                for (const ik of Object.keys(innerProps)) {
                  childKeys.push(`${ck}_${ik}`)
                }
              } else {
                flatProps[flatKey] = {
                  type: "string",
                  description: `${(cv as JsonSchema).description ?? ck} (per item comma-separated, items pipe-separated)`,
                }
              }
            } else if (childType === "boolean") {
              flatProps[flatKey] = {
                type: "string",
                description: `${(cv as JsonSchema).description ?? ck} (pipe-separated booleans: true|false|true)`,
              }
            } else {
              flatProps[flatKey] = {
                type: "string",
                description: `${(cv as JsonSchema).description ?? ck} (pipe-separated values)`,
              }
            }
          }
          flattenMap.set(key, { kind: "array_of_objects", key, childKeys })
          if (required.has(key)) {
            // At least the first child key is required
            flatRequired.push(`${key}_${childKeys[0]}`)
          }
        } else {
          // Array of primitives → comma-separated string
          flatProps[key] = {
            type: "string",
            description: `${prop.description ?? key} (comma-separated)`,
          }
          flattenMap.set(key, { kind: "array_of_primitives", key, itemType: itemType ?? "string" })
          if (required.has(key)) flatRequired.push(key)
        }
      }
    } else {
      // Basic type — pass through
      flatProps[key] = prop
      if (required.has(key)) flatRequired.push(key)
    }
  }

  return {
    flattenedSchema: {
      type: "object",
      properties: flatProps,
      required: flatRequired.length > 0 ? flatRequired : undefined,
      additionalProperties: schema.additionalProperties,
    },
    flattenMap,
  }
}

/**
 * Expand flat tool call arguments back to their nested structure.
 */
export function expandToolCallArgs(
  flatArgs: Record<string, unknown>,
  flattenMap: FlattenMap,
): Record<string, unknown> {
  const result: Record<string, unknown> = {}
  const consumed = new Set<string>()

  for (const [origKey, entry] of flattenMap) {
    if (entry.kind === "object") {
      const obj: Record<string, unknown> = {}
      let hasAny = false
      for (const child of entry.children) {
        const flatKey = `${entry.prefix}_${child}`
        if (flatKey in flatArgs) {
          obj[child] = flatArgs[flatKey]
          consumed.add(flatKey)
          hasAny = true
        }
      }
      if (hasAny) result[origKey] = obj
    } else if (entry.kind === "array_of_primitives") {
      const val = flatArgs[entry.key]
      consumed.add(entry.key)
      if (val === undefined || val === "") {
        result[origKey] = []
      } else if (entry.itemType === "json") {
        try {
          result[origKey] = JSON.parse(String(val))
        } catch {
          result[origKey] = [String(val)]
        }
      } else {
        const parts = String(val).split(",").map((s) => s.trim())
        result[origKey] = parts.map((p) => coerceValue(p, entry.itemType))
      }
    } else if (entry.kind === "array_of_objects") {
      // Reconstruct array of objects from pipe-separated child values
      const childValues: Record<string, string[]> = {}
      let maxLen = 0
      for (const childKey of entry.childKeys) {
        const flatKey = `${entry.key}_${childKey}`
        const val = flatArgs[flatKey]
        consumed.add(flatKey)
        // Also consume nested keys like questions_options_label
        for (const fk of Object.keys(flatArgs)) {
          if (fk.startsWith(`${entry.key}_${childKey}_`)) {
            consumed.add(fk)
          }
        }
        if (val !== undefined && val !== "") {
          const parts = String(val).split("|")
          childValues[childKey] = parts
          maxLen = Math.max(maxLen, parts.length)
        }
      }
      const arr: Record<string, unknown>[] = []
      for (let i = 0; i < maxLen; i++) {
        const obj: Record<string, unknown> = {}
        for (const childKey of entry.childKeys) {
          const parts = childValues[childKey]
          if (parts && i < parts.length) {
            // Check if this child key has nested children (e.g. options_label, options_description)
            if (childKey.includes("_")) {
              // This is a flattened nested array-of-objects property
              const [arrKey, ...rest] = childKey.split("_")
              const propName = rest.join("_")
              if (!obj[arrKey!]) obj[arrKey!] = []
              const innerArr = obj[arrKey!] as Record<string, unknown>[]
              const innerParts = parts[i]!.split(",").map((s) => s.trim())
              for (let j = 0; j < innerParts.length; j++) {
                if (!innerArr[j]) innerArr[j] = {}
                innerArr[j]![propName] = innerParts[j]
              }
            } else {
              obj[childKey] = parts[i]!.trim()
            }
          }
        }
        if (Object.keys(obj).length > 0) arr.push(obj)
      }
      result[origKey] = arr
    }
  }

  // Pass through any non-flattened args
  for (const [key, val] of Object.entries(flatArgs)) {
    if (!consumed.has(key)) {
      result[key] = val
    }
  }

  return result
}

function coerceValue(val: string, type: string): unknown {
  switch (type) {
    case "integer":
    case "number":
      return Number(val)
    case "boolean":
      return val === "true"
    default:
      return val
  }
}
