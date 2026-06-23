import { Box, Flex, Text, useColorModeValue } from "@chakra-ui/react"
import { Handle, Position, type Node, type NodeProps } from "@xyflow/react"
import React from "react"
import { FaKey } from "react-icons/fa"
import { FiSettings } from "react-icons/fi"

export type SecretNodeData = {
  name: string
  kind: "secret" | "config"
}

export type SecretFlowNode = Node<SecretNodeData, "secret">

function SecretNode({ data }: NodeProps<SecretFlowNode>): React.ReactElement {
  const { name, kind } = data
  const bg = useColorModeValue("gray.50", "gray.800")
  const accent = kind === "secret" ? "purple.400" : "blue.400"
  const KindIcon = kind === "secret" ? FaKey : FiSettings

  return (
    <Box bg={bg} borderWidth="1px" borderStyle="dashed" borderColor={accent} borderRadius="md" px={2.5} py={1.5} minW="150px">
      <Handle type="target" position={Position.Left} />
      <Flex align="center" gap={2}>
        <Box as="span" color={accent} display="inline-flex" aria-label={kind}>
          <KindIcon />
        </Box>
        <Text fontSize="xs" fontWeight="semibold" noOfLines={1}>
          {name}
        </Text>
      </Flex>
      <Text fontSize="0.65rem" color="gray.500">
        {kind}
      </Text>
    </Box>
  )
}

export default SecretNode
