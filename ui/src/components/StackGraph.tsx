import { Box, Text } from "@chakra-ui/react"
import {
  Background,
  Controls,
  Position,
  ReactFlow,
  useEdgesState,
  useNodesState,
  type Edge,
  type Node,
  type NodeTypes
} from "@xyflow/react"
import "@xyflow/react/dist/style.css"
import React, { useEffect, useMemo, useRef } from "react"
import { ServiceStatus } from "../hooks/useFetchStackServices"
import SecretNode from "./SecretNode"
import ServiceNode from "./ServiceNode"
import TaskNode from "./TaskNode"

// Registered once at module scope so React Flow does not warn about a new
// nodeTypes object on every render.
const nodeTypes: NodeTypes = { service: ServiceNode, task: TaskNode, secret: SecretNode }

const COL_STACK = 0
const COL_SERVICE = 260
const COL_TASK = 560
const COL_SECRET = 840
const ROW = 72
const DASHED_EDGE = { strokeDasharray: "6 4" }

type BuiltGraph = { nodes: Node[]; edges: Edge[] }

// buildGraph lays out an ArgoCD-style resource tree: stack(root) → service →
// task, plus secret/config nodes linked to their service with dashed edges.
// Layout is deterministic (column by depth, row by running index); React Flow
// makes the nodes draggable afterwards.
function buildGraph(stackName: string, services: ServiceStatus[]): BuiltGraph {
  const nodes: Node[] = []
  const edges: Edge[] = []
  const rootId = "stack-root"
  const secretNodeIds = new Map<string, string>()
  let cursorY = 0

  services.forEach((service, serviceIndex) => {
    const serviceNodeId = service.ID || service.Name || `service-${serviceIndex}`
    const childCount = Math.max(service.Tasks.length + service.Secrets.length + service.Configs.length, 1)
    const blockStart = cursorY
    const blockHeight = childCount * ROW

    nodes.push({
      id: serviceNodeId,
      type: "service",
      position: { x: COL_SERVICE, y: blockStart + (blockHeight - ROW) / 2 },
      data: { service }
    })
    edges.push({ id: `edge-${rootId}-${serviceNodeId}`, source: rootId, target: serviceNodeId })

    let childRow = 0
    service.Tasks.forEach((task, taskIndex) => {
      const taskNodeId = `${serviceNodeId}-task-${task.ID || taskIndex}`
      nodes.push({
        id: taskNodeId,
        type: "task",
        position: { x: COL_TASK, y: blockStart + childRow * ROW },
        data: { task }
      })
      edges.push({ id: `edge-${serviceNodeId}-${taskNodeId}`, source: serviceNodeId, target: taskNodeId })
      childRow += 1
    })

    const refs = [
      ...service.Secrets.map(name => ({ name, kind: "secret" as const })),
      ...service.Configs.map(name => ({ name, kind: "config" as const }))
    ]
    refs.forEach(ref => {
      const key = `${ref.kind}:${ref.name}`
      let secretNodeId = secretNodeIds.get(key)
      if (secretNodeId === undefined) {
        secretNodeId = `secret-${ref.kind}-${ref.name}`
        secretNodeIds.set(key, secretNodeId)
        nodes.push({
          id: secretNodeId,
          type: "secret",
          position: { x: COL_SECRET, y: blockStart + childRow * ROW },
          data: { name: ref.name, kind: ref.kind }
        })
        childRow += 1
      }
      edges.push({
        id: `edge-${serviceNodeId}-${secretNodeId}`,
        source: serviceNodeId,
        target: secretNodeId,
        style: DASHED_EDGE
      })
    })

    cursorY = blockStart + blockHeight + ROW / 2
  })

  nodes.unshift({
    id: rootId,
    type: "input",
    position: { x: COL_STACK, y: Math.max(0, (cursorY - ROW) / 2) },
    data: { label: stackName },
    sourcePosition: Position.Right
  })

  return { nodes, edges }
}

function nodeSignature(nodes: Node[]): string {
  return nodes
    .map(node => node.id)
    .sort()
    .join("|")
}

function StackGraph({ stackName, services }: Readonly<{ stackName: string; services: ServiceStatus[] }>): React.ReactElement {
  const built = useMemo(() => buildGraph(stackName, services), [stackName, services])
  const [nodes, setNodes, onNodesChange] = useNodesState<Node>(built.nodes)
  const [edges, setEdges, onEdgesChange] = useEdgesState<Edge>(built.edges)
  const lastSignature = useRef<string>(nodeSignature(built.nodes))

  useEffect(() => {
    const signature = nodeSignature(built.nodes)
    if (signature !== lastSignature.current) {
      // Structure changed (services/tasks/secrets added or removed) → rebuild layout.
      lastSignature.current = signature
      console.debug("[StackGraph] rebuild", stackName, "nodes", built.nodes.length, "edges", built.edges.length)
      setNodes(built.nodes)
      setEdges(built.edges)
      return
    }
    // Same node set → patch data only, preserving user-dragged positions across polls.
    const dataById = new Map(built.nodes.map(node => [node.id, node.data]))
    setNodes(current =>
      current.map(node => {
        const data = dataById.get(node.id)
        return data ? { ...node, data } : node
      })
    )
  }, [built, stackName, setNodes, setEdges])

  if (services.length === 0) {
    return (
      <Box textAlign="center" py={8}>
        <Text>No services found for this stack.</Text>
      </Box>
    )
  }

  return (
    <Box h="65vh" w="100%">
      <ReactFlow
        nodes={nodes}
        edges={edges}
        onNodesChange={onNodesChange}
        onEdgesChange={onEdgesChange}
        nodeTypes={nodeTypes}
        nodesDraggable
        fitView
      >
        <Background />
        <Controls />
      </ReactFlow>
    </Box>
  )
}

export default StackGraph
