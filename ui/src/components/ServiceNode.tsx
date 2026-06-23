import { Badge, Box, Flex, Text, useColorModeValue } from "@chakra-ui/react"
import { Handle, Position, type Node, type NodeProps } from "@xyflow/react"
import React from "react"
import { ServiceStatus } from "../hooks/useFetchStackServices"
import { healthColor, healthIcon } from "./health"

export type ServiceNodeData = {
  service: ServiceStatus
}

export type ServiceFlowNode = Node<ServiceNodeData, "service">

function ServiceNode({ data }: NodeProps<ServiceFlowNode>): React.ReactElement {
  const { service } = data
  const bg = useColorModeValue("white", "gray.700")
  const color = healthColor(service.Health)
  const HealthIcon = healthIcon(service.Health)

  return (
    <Box bg={bg} borderWidth="1px" borderRadius="md" borderLeftWidth="4px" borderLeftColor={color} boxShadow="md" px={3} py={2} minW="210px">
      <Handle type="target" position={Position.Left} />
      <Handle type="source" position={Position.Right} />
      <Flex align="center" gap={2}>
        <Box as="span" title={service.Health} aria-label={service.Health} color={color} display="inline-flex" fontSize="md">
          <HealthIcon />
        </Box>
        <Text fontWeight="bold" fontSize="sm" noOfLines={1}>
          {service.Name}
        </Text>
      </Flex>
      <Text fontSize="xs" color="gray.500" noOfLines={1} mt={1}>
        {service.Image}
      </Text>
      <Flex align="center" gap={2} mt={1} wrap="wrap">
        <Badge textTransform="lowercase">{service.Mode}</Badge>
        <Text fontSize="xs">
          {service.RunningTasks}/{service.DesiredTasks} replicas
        </Text>
        {service.FailedTasks > 0 && (
          <Text fontSize="xs" color="red.500" fontWeight="semibold">
            {service.FailedTasks} failed
          </Text>
        )}
      </Flex>
    </Box>
  )
}

export default ServiceNode
