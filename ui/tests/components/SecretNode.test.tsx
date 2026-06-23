import { ChakraProvider } from "@chakra-ui/react"
import { render, screen } from "@testing-library/react"
import type { NodeProps } from "@xyflow/react"
import SecretNode, { type SecretFlowNode } from "../../src/components/SecretNode"

vi.mock("@xyflow/react", () => ({
  Handle: () => null,
  Position: { Left: "left", Right: "right", Top: "top", Bottom: "bottom" }
}))

function nodeProps(name: string, kind: "secret" | "config"): NodeProps<SecretFlowNode> {
  return { data: { name, kind } } as unknown as NodeProps<SecretFlowNode>
}

describe("SecretNode", () => {
  it("renders a secret with its name and kind", () => {
    render(
      <ChakraProvider>
        <SecretNode {...nodeProps("tls-cert", "secret")} />
      </ChakraProvider>
    )

    expect(screen.getByText("tls-cert")).toBeInTheDocument()
    expect(screen.getByText("secret")).toBeInTheDocument()
    expect(screen.getByLabelText("secret")).toBeInTheDocument()
  })

  it("renders a config with its name and kind", () => {
    render(
      <ChakraProvider>
        <SecretNode {...nodeProps("app-config", "config")} />
      </ChakraProvider>
    )

    expect(screen.getByText("app-config")).toBeInTheDocument()
    expect(screen.getByText("config")).toBeInTheDocument()
  })
})
