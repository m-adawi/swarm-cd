import { Alert, AlertIcon, Box, CloseButton, Text } from "@chakra-ui/react"
import React, { useEffect, useState } from "react"

function WarningBanner({ warnings }: Readonly<{ warnings: string[] }>): React.ReactElement | null {
  const [dismissed, setDismissed] = useState<Set<string>>(new Set())

  useEffect(() => {
    setDismissed(prev => {
      const updated = new Set(prev)
      for (const key of prev) {
        if (!warnings.includes(key)) updated.delete(key)
      }
      return updated.size !== prev.size ? updated : prev
    })
  }, [warnings])

  const visible = warnings.filter(w => !dismissed.has(w))
  if (visible.length === 0) return null

  return (
    <Box mb={3}>
      {visible.map(warning => (
        <Alert key={warning} status="warning" borderRadius="md" mb={1}>
          <AlertIcon />
          <Text fontSize="sm" flex="1">{warning}</Text>
          <CloseButton size="sm" onClick={() => setDismissed(prev => new Set(prev).add(warning))} />
        </Alert>
      ))}
    </Box>
  )
}

export default WarningBanner
