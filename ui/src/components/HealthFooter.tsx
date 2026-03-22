import { Box, HStack, Text, useColorModeValue } from "@chakra-ui/react"
import React from "react"
import { HealthData } from "../hooks/useFetchHealth"

function formatUptime(seconds: number): string {
  const days = Math.floor(seconds / 86400)
  const hours = Math.floor((seconds % 86400) / 3600)
  const minutes = Math.floor((seconds % 3600) / 60)

  if (days > 0) return `${days}d ${hours}h`
  if (hours > 0) return `${hours}h ${minutes}m`
  return `${minutes}m`
}

function HealthFooter({ health }: Readonly<{ health: HealthData }>): React.ReactElement {
  const bg = useColorModeValue("gray.50", "gray.800")
  const color = useColorModeValue("gray.500", "gray.400")

  return (
    <Box as="footer" textAlign="center" py={4} mt={6} bg={bg} borderRadius="md">
      <HStack justify="center" spacing={4}>
        <Text fontSize="xs" color={color}>
          SwarmCD {health.version}
        </Text>
        <Text fontSize="xs" color={color}>
          Uptime: {formatUptime(health.uptime_seconds)}
        </Text>
        <Text fontSize="xs" color={color}>
          Poll: {health.update_interval_seconds}s
        </Text>
        <Text fontSize="xs" color={color}>
          {health.stacks_managed} stack{health.stacks_managed !== 1 ? "s" : ""} managed
        </Text>
      </HStack>
    </Box>
  )
}

export default HealthFooter
