import { Box, Flex, Text, useColorModeValue } from "@chakra-ui/react"
import { Handle, Position, type Node, type NodeProps } from "@xyflow/react"
import React from "react"
import { TaskStatus } from "../hooks/useFetchStackServices"
import { taskStateColor } from "./health"

export type TaskNodeData = {
  task: TaskStatus
}

export type TaskFlowNode = Node<TaskNodeData, "task">

function TaskNode({ data }: NodeProps<TaskFlowNode>): React.ReactElement {
  const { task } = data
  const bg = useColorModeValue("white", "gray.800")
  const color = taskStateColor(task.State)
  const label = task.Slot > 0 ? `Slot ${task.Slot}` : task.NodeID || task.ID

  return (
    <Box bg={bg} borderWidth="1px" borderRadius="md" boxShadow="sm" px={2.5} py={1.5} minW="150px" title={task.Error !== "" ? task.Error : undefined}>
      <Handle type="target" position={Position.Left} />
      <Flex align="center" gap={2}>
        <Box w={2} h={2} borderRadius="full" bg={color} title={task.State} />
        <Text fontSize="xs" fontWeight="semibold" noOfLines={1}>
          {label}
        </Text>
      </Flex>
      <Text fontSize="0.65rem" color="gray.500" noOfLines={1}>
        {task.State}
        {task.NodeID !== "" && task.Slot > 0 ? ` · ${task.NodeID}` : ""}
      </Text>
      {task.Error !== "" && (
        <Text fontSize="0.65rem" color="red.500" noOfLines={1}>
          {task.Error}
        </Text>
      )}
    </Box>
  )
}

export default TaskNode
