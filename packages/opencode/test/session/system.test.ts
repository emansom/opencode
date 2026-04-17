import { describe, expect, test } from "bun:test"
import { SystemPrompt } from "../../src/session/system"

describe("session.system", () => {
  test("skills returns undefined — catalog is built in llm.ts from SkillRegistry", async () => {
    const result = await SystemPrompt.skills({ name: "build", mode: "default", permission: [] } as any)
    expect(result).toBeUndefined()
  })
})
