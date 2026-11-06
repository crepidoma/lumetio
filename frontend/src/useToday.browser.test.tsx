import { afterEach, expect, test, vi } from "vitest";
import { renderHook } from "vitest-browser-react";
import { Service } from "../bindings/github.com/crepidoma/lumetio/internal/gui";
import { useToday } from "./useToday";

vi.mock("../bindings/github.com/crepidoma/lumetio/internal/gui", () => ({
	Service: { Today: vi.fn() },
}));

afterEach(() => {
	vi.useRealTimers();
});

test("起動したまま日付が変わったら、今日は新しい日になる", async () => {
	vi.useFakeTimers({ toFake: ["setInterval", "clearInterval"] });
	vi.mocked(Service.Today).mockResolvedValue("2026-11-05");
	const { result } = await renderHook(() => useToday());
	await vi.waitFor(() => expect(result.current).toBe("2026-11-05"));

	vi.mocked(Service.Today).mockResolvedValue("2026-11-06");
	await vi.advanceTimersByTimeAsync(60_000);
	await vi.waitFor(() => expect(result.current).toBe("2026-11-06"));
});
