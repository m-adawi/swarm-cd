import { Box, Grid, Link, Text } from "@chakra-ui/react"
import React from "react"

function StatusCard({
  name,
  error,
  revision,
  repoURL
}: Readonly<{
  name: string
  error: string
  revision: string
  repoURL: string
}>): React.ReactElement {
  return (
    <Box borderWidth="1px" borderRadius="sm" overflow="hidden" p={4} boxShadow="md">
      <Grid templateColumns="auto 1fr" gap={2}>
        <Text fontWeight="bold">Name:</Text>
        <Text>{name}</Text>

        {error !== "" && (
          <>
            <Text fontWeight="bold" color="red.500">
              Error:
            </Text>
            <Text color="red.500">{error}</Text>
          </>
        )}

        <Text fontWeight="bold">Revision:</Text>
        <Text>{revision}</Text>

        <Text fontWeight="bold">Repo URL:</Text>
        <Link color="teal.500" href={repoURL} isExternal>
          {repoURL}
        </Link>
      </Grid>
    </Box>
  )
}

export default StatusCard
