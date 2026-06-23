import { ChakraProvider } from "@chakra-ui/react"
import { render, screen } from "@testing-library/react"
import type { NodeProps } from "@xyflow/react"
import ServiceNode, { type ServiceFlowNode } from "../../src/components/ServiceNode"
import { ServiceStatus } from "../../src/hooks/useFetchStackServices"

// React Flow's Handle needs a provider context; stub it out for unit testing
// the node's presentation.
vi.mock("@xyflow/react", () => ({
  Handle: () => null,
  Position: { Left: "left", Right: "right", Top: "top", Bottom: "bottom" }
}))

const baseService: ServiceStatus = {
  ID: "svc-1",
  Name: "web_nginx",
  Image: "nginx:1.27",
  Mode: "replicated",
  RunningTasks: 3,
  DesiredTasks: 3,
  FailedTasks: 0,
  Health: "healthy",
  Tasks: [],
  Secrets: [],
  Configs: []
}

function nodeProps(service: ServiceStatus): NodeProps<ServiceFlowNode> {
  return { data: { service } } as unknown as NodeProps<ServiceFlowNode>
}

describe("ServiceNode", () => {
  it("renders service name, image, mode and replica counts", () => {
    render(
      <ChakraProvider>
        <ServiceNode {...nodeProps(baseService)} />
      </ChakraProvider>
    )

    expect(screen.getByText("web_nginx")).toBeInTheDocument()
    expect(screen.getByText("nginx:1.27")).toBeInTheDocument()
    expect(screen.getByText("replicated")).toBeInTheDocument()
    expect(screen.getByText("3/3 replicas")).toBeInTheDocument()
  })

  it.each(["healthy", "progressing", "degraded"])("exposes the %s health via the indicator title", health => {
    render(
      <ChakraProvider>
        <ServiceNode {...nodeProps({ ...baseService, Health: health })} />
      </ChakraProvider>
    )

    expect(screen.getByTitle(health)).toBeInTheDocument()
  })

  it("shows the failed-task count when FailedTasks > 0", () => {
    render(
      <ChakraProvider>
        <ServiceNode {...nodeProps({ ...baseService, FailedTasks: 2 })} />
      </ChakraProvider>
    )

    expect(screen.getByText(/2 failed/)).toBeInTheDocument()
  })
})
