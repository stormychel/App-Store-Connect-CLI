import { render, screen } from "@testing-library/react";
import { afterEach, beforeEach, vi } from "vitest";

const {
  mockListApps,
  mockCheckAuthStatus,
  mockBootstrap,
  mockGetSettings,
  mockSaveSettings,
} = vi.hoisted(() => ({
  mockListApps: vi.fn(),
  mockCheckAuthStatus: vi.fn(),
  mockBootstrap: vi.fn(),
  mockGetSettings: vi.fn(),
  mockSaveSettings: vi.fn(),
}));

vi.mock("../wailsjs/go/main/App", () => ({
  ListApps: mockListApps,
  CheckAuthStatus: mockCheckAuthStatus,
  Bootstrap: mockBootstrap,
  GetSettings: mockGetSettings,
  SaveSettings: mockSaveSettings,
  RunASCCommand: vi.fn().mockResolvedValue({ error: "", data: "{\"data\":[]}" }),
  GetAppDetail: vi.fn(),
  GetVersionMetadata: vi.fn(),
  GetScreenshots: vi.fn(),
  GetTestFlight: vi.fn(),
  GetTestFlightTesters: vi.fn(),
  GetPricingOverview: vi.fn(),
  GetSubscriptions: vi.fn(),
  GetFinanceRegions: vi.fn(),
  GetOfferCodes: vi.fn(),
  GetFeedback: vi.fn(),
}));

vi.mock("../wailsjs/go/models", () => ({
  environment: { Snapshot: class {} },
  settings: {
    StudioSettings: class {
      constructor(source: Record<string, unknown> = {}) {
        Object.assign(this, source);
      }
    },
    ProviderPreset: class {},
  },
}));

vi.mock("./components/Sidebar", () => ({
  Sidebar: function MockSidebar() {
    throw new Error("sidebar exploded");
  },
}));

import App from "./App";

describe("App shell error boundary", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    vi.spyOn(console, "error").mockImplementation(() => {});

    mockListApps.mockResolvedValue({ apps: [] });
    mockCheckAuthStatus.mockResolvedValue({
      authenticated: true,
      storage: "System Keychain",
      profile: "default",
      rawOutput: "",
    });
    mockBootstrap.mockResolvedValue({
      appName: "ASC Studio",
      environment: {
        configPath: "/Users/test/.asc/config.json",
        configPresent: true,
        defaultAppId: "",
        keychainAvailable: true,
        keychainBypassed: false,
        workflowPath: "",
      },
      settings: {
        preferredPreset: "codex",
        agentCommand: "",
        agentArgs: [],
        agentEnv: {},
        preferBundledASC: true,
        systemASCPath: "",
        workspaceRoot: "",
        theme: "glass-light",
        windowMaterial: "translucent",
        showCommandPreviews: true,
      },
      presets: [],
      threads: [],
      approvals: [],
    });
    mockGetSettings.mockResolvedValue({});
    mockSaveSettings.mockResolvedValue({});
  });

  afterEach(() => {
    vi.restoreAllMocks();
  });

  it("shows a shell-level fallback when navigation rendering throws", async () => {
    render(<App />);

    expect(await screen.findByText("Something went wrong")).toBeInTheDocument();
    expect(screen.getByText("sidebar exploded")).toBeInTheDocument();
  });
});
