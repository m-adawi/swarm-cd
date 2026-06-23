import { ChakraProvider } from "@chakra-ui/react"
import { render, screen } from "@testing-library/react"
import type { NodeProps } from "@xyflow/react"
import TaskNode, { type TaskFlowNode } from "../../src/components/TaskNode"
import { TaskStatus } from "../../src/hooks/useFetchStackServices"

vi.mock("@xyflow/react", () => ({
  Handle: () => null,
  Position: { Left: "left", Right: "right", Top: "top", Bottom: "bottom" }
}))

function nodeProps(task: TaskStatus): NodeProps<TaskFlowNode> {
  return { data: { task } } as unknown as NodeProps<TaskFlowNode>
}

const runningTask: TaskStatus = {
  ID: "t-1",
  Slot: 1,
  NodeID: "node-1",
  State: "running",
  DesiredState: "running",
  Error: "",
  ContainerID: "c0ffee"
}

describe("TaskNode", () => {
  it("renders slot label and running state", () => {
    render(
      <ChakraProvider>
        <TaskNode {...nodeProps(runningTask)} />
      </ChakraProvider>
    )

    expect(screen.getByText("Slot 1")).toBeInTheDocument()
    expect(screen.getByTitle("running")).toBeInTheDocument()
  })

  it("renders the error text for a failed task", () => {
    render(
      <ChakraProvider>
        <TaskNode {...nodeProps({ ...runningTask, State: "failed", Error: "task: non-zero exit (1)" })} />
      </ChakraProvider>
    )

    expect(screen.getByText("task: non-zero exit (1)")).toBeInTheDocument()
    expect(screen.getByTitle("failed")).toBeInTheDocument()
  })
})
