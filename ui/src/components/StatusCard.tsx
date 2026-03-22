import { Box, Grid, Link, Text, TextProps } from "@chakra-ui/react"
import React from "react"

function StatusCard({
  name,
  error,
  revision,
  repoURL,
  gitRef
}: Readonly<{
  name: string
  error: string
  revision: string
  repoURL: string
  gitRef: string
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

        <KeyText>Watching:</KeyText>
        <Text>{gitRef}</Text>

        <KeyText>Revision:</KeyText>
        <Text>{revision}</Text>

        <KeyText>Repo URL:</KeyText>
        <Link color="teal.500" href={repoURL} isExternal>
          {repoURL}
        </Link>
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
