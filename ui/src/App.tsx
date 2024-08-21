import { Container, Text } from "@chakra-ui/react"
import React, { useState } from "react"
import HeaderBar from "./components/HeaderBar"
import StatusCardList from "./components/StatusCardList"
import useFetchStatuses from "./hooks/useFetchStatuses"

function App(): React.ReactElement {
  const { stacks, error } = useFetchStatuses()
  const [searchQuery, setSearchQuery] = useState("")

  return (
    <>
      <Container maxW="container.lg" mt={4}>
        <HeaderBar onQueryChange={query => setSearchQuery(query)} />
        {error !== null ? (
          <Text fontSize="xl" align="center">
            {error}
          </Text>
        ) : (
          <StatusCardList statuses={stacks} query={searchQuery} />
        )}
      </Container>
    </>
  )
}

export default App
