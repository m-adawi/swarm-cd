import { Container, Text } from "@chakra-ui/react"
import React, { useState } from "react"
import HeaderBar from "./components/HeaderBar"
import HealthFooter from "./components/HealthFooter"
import StatusCardList from "./components/StatusCardList"
import WarningBanner from "./components/WarningBanner"
import useFetchHealth from "./hooks/useFetchHealth"
import useFetchStatuses from "./hooks/useFetchStatuses"

function App(): React.ReactElement {
  const { statuses, error } = useFetchStatuses()
  const { health } = useFetchHealth()
  const [searchQuery, setSearchQuery] = useState("")

  return (
    <Container maxW="container.lg" mt={4}>
      <HeaderBar onQueryChange={query => setSearchQuery(query)} error={error !== null} />
      {health?.config_warnings && health.config_warnings.length > 0 && (
        <WarningBanner warnings={health.config_warnings} />
      )}
      {error === null ? (
        <StatusCardList statuses={statuses} query={searchQuery} />
      ) : (
        <Text fontSize="xl" align="center" color="red.500">
          {error}
        </Text>
      )}
      {health && <HealthFooter health={health} />}
    </Container>
  )
}

export default App
