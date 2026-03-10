import { describe, it, expect } from "vitest";

describe("Home page module", () => {
  it("exports a default component", async () => {
    const mod = await import("./page");
    expect(mod.default).toBeDefined();
    expect(typeof mod.default).toBe("function");
  });
});
