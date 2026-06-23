import { render, screen } from "@testing-library/react"
import StackGraph from "../../src/components/StackGraph"
import { ServiceStatus, TaskStatus } from "../../src/hooks/useFetchStackServices"

// Stub React Flow so the graph can be asserted deterministically in happy-dom.
// The stub renders one element per node (tagged with its type) and per edge
// (tagged dashed/solid) so the test can verify the resource tree shape.
vi.mock("@xyflow/react", () => ({
  ReactFlow: ({
    nodes,
    edges
  }: {
    nodes: ReadonlyArray<{ id: string; type?: string }>
    edges: ReadonlyArray<{ id: string; style?: { strokeDasharray?: string } }>
  }) => (
    <div data-testid="reactflow">
      {nodes.map(node => (
        <div key={node.id} data-testid="rf-node" data-type={node.type}>
          {node.id}
        </div>
      ))}
      {edges.map(edge => (
        <div key={edge.id} data-testid="rf-edge" data-dashed={edge.style?.strokeDasharray !== undefined ? "true" : "false"}>
          {edge.id}
        </div>
      ))}
    </div>
  ),
  Background: () => null,
  Controls: () => null,
  Handle: () => null,
  useNodesState: (initial: unknown) => [initial, () => {}, () => {}],
  useEdgesState: (initial: unknown) => [initial, () => {}, () => {}],
  Position: { Left: "left", Right: "right", Top: "top", Bottom: "bottom" }
}))

function task(id: string): TaskStatus {
  return { ID: id, Slot: 1, NodeID: "node-1", State: "running", DesiredState: "running", Error: "", ContainerID: "c1" }
}

const services: ServiceStatus[] = [
  {
    ID: "s1", Name: "web", Image: "nginx", Mode: "replicated",
    RunningTasks: 2, DesiredTasks: 2, FailedTasks: 0, Health: "healthy",
    Tasks: [task("t1"), task("t2")], Secrets: ["tls"], Configs: []
  },
  {
    ID: "s2", Name: "db", Image: "postgres", Mode: "replicated",
    RunningTasks: 1, DesiredTasks: 1, FailedTasks: 0, Health: "healthy",
    Tasks: [task("t3")], Secrets: [], Configs: ["dbcfg"]
  }
]

function nodesByType(type: string): HTMLElement[] {
  return screen.getAllByTestId("rf-node").filter(node => node.getAttribute("data-type") === type)
}

describe("StackGraph", () => {
  it("renders stack, service, task and secret/config nodes", () => {
    render(<StackGraph stackName="demo" services={services} />)

    // root(1) + services(2) + tasks(3) + secret(1) + config(1) = 8
    expect(screen.getAllByTestId("rf-node")).toHaveLength(8)
    expect(nodesByType("service")).toHaveLength(2)
    expect(nodesByType("task")).toHaveLength(3)
    expect(nodesByType("secret")).toHaveLength(2)
  })

  it("draws dashed edges to secret/config nodes", () => {
    render(<StackGraph stackName="demo" services={services} />)

    const dashed = screen.getAllByTestId("rf-edge").filter(edge => edge.getAttribute("data-dashed") === "true")
    expect(dashed).toHaveLength(2)
  })

  it("shows an empty state when there are no services", () => {
    render(<StackGraph stackName="demo" services={[]} />)
    expect(screen.getByText(/no services/i)).toBeInTheDocument()
  })
})
