import { Box, Grid, Link, Text, TextProps } from "@chakra-ui/react"
import React from "react"

function StatusCard({
  name,
  error,
  revision,
  repoURL,
  templated
}: Readonly<{
  name: string
  error: string
  revision: string
  repoURL: string
  templated: boolean
}>): React.ReactElement {
  return (
    <Box borderWidth="1px" borderRadius="sm" overflow="hidden" p={4} boxShadow="lg">
      <Grid templateColumns="auto 1fr" gap={2}>
        <KeyText>Name:</KeyText>
        <Text>{name}</Text>

        {error !== "" && (
          <>
            <KeyText>Error:</KeyText>
            <Text color="red.500">{error}</Text>
          </>
        )}

        <KeyText>Revision:</KeyText>
        <Text>{revision}</Text>

        <KeyText>Repo URL:</KeyText>
        <Link color="teal.500" href={repoURL} isExternal>
          {repoURL}
        </Link>

        <KeyText>Compose file:</KeyText>
        <Link color="teal.500" href={`/stacks/${name}/compose.yaml`} isExternal>
          compose.yaml
        </Link>
        {templated && (
          <>
            <KeyText>Rendered file:</KeyText>
            <Link color="teal.500" href={`/stacks/${name}/rendered.yaml`} isExternal>
              rendered.yaml
            </Link>
          </>
	)}
      </Grid>
    </Box>
  )
}

function KeyText({ children, ...props }: Readonly<TextProps>): React.ReactElement {
  return (
    <Text fontWeight="bold" {...props}>
      {children}
    </Text>
  )
}

export default StatusCard
