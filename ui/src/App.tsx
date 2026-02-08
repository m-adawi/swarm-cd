import { Container, Text } from "@chakra-ui/react"
import React, { useState } from "react"
import HeaderBar from "./components/HeaderBar"
import StatusCardList from "./components/StatusCardList"
import useFetchStatuses from "./hooks/useFetchStatuses"

function App(): React.ReactElement {
  const { statuses, error, hasUpdate, checkForUpdate, applyUpdate, isChecking } = useFetchStatuses()
  const [searchQuery, setSearchQuery] = useState("")

  return (
    <Container maxW="container.lg" mt={4}>
      <HeaderBar
        onQueryChange={query => setSearchQuery(query)}
        error={error !== null}
        hasUpdate={hasUpdate}
        checkForUpdate={checkForUpdate}
        applyUpdate={applyUpdate}
        isChecking={isChecking}
      />
      {error === null ? (
        <StatusCardList statuses={statuses} query={searchQuery} />
      ) : (
        <Text fontSize="xl" align="center" color="red.500">
          {error}
        </Text>
      )}
    </Container>
  )
}

export default App
