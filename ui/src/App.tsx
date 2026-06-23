import {
  Box,
  Container,
  Flex,
  Modal,
  ModalBody,
  ModalCloseButton,
  ModalContent,
  ModalHeader,
  ModalOverlay,
  Spinner,
  Text,
  useDisclosure
} from "@chakra-ui/react"
import React, { Suspense, lazy, useState } from "react"
import HeaderBar from "./components/HeaderBar"
import { HEALTH_LABELS, HEALTH_ORDER, healthColor } from "./components/health"
import StatusCardList from "./components/StatusCardList"
import useFetchStackServices from "./hooks/useFetchStackServices"
import useFetchStatuses from "./hooks/useFetchStatuses"

// Lazy-loaded so React Flow (and its CSS) is code-split out of the initial
// bundle — it only loads when a user opens a stack's service graph.
const StackGraph = lazy(() => import("./components/StackGraph"))

function HealthLegend(): React.ReactElement {
  return (
    <Flex gap={4} mb={3} fontSize="sm" align="center" wrap="wrap">
      {HEALTH_ORDER.map(level => (
        <Flex key={level} align="center" gap={1}>
          <Box w={2.5} h={2.5} borderRadius="full" bg={healthColor(level)} />
          <Text>{HEALTH_LABELS[level]}</Text>
        </Flex>
      ))}
    </Flex>
  )
}

function App(): React.ReactElement {
  const { statuses, error } = useFetchStatuses()
  const [searchQuery, setSearchQuery] = useState("")
  const [selectedStack, setSelectedStack] = useState<string | null>(null)
  const { services, error: servicesError, loading } = useFetchStackServices(selectedStack)
  const { isOpen, onOpen, onClose } = useDisclosure()

  const handleSelect = (name: string): void => {
    setSelectedStack(name)
    onOpen()
  }

  const handleClose = (): void => {
    onClose()
    setSelectedStack(null)
  }

  return (
    <Container maxW="container.lg" mt={4}>
      <HeaderBar onQueryChange={query => setSearchQuery(query)} error={error !== null} />
      {error === null ? (
        <StatusCardList statuses={statuses} query={searchQuery} onSelect={handleSelect} />
      ) : (
        <Text fontSize="xl" align="center" color="red.500">
          {error}
        </Text>
      )}

      <Modal isOpen={isOpen} onClose={handleClose} size="6xl" scrollBehavior="inside">
        <ModalOverlay />
        <ModalContent>
          <ModalHeader>{selectedStack !== null ? `${selectedStack} — services` : "Services"}</ModalHeader>
          <ModalCloseButton />
          <ModalBody pb={6}>
            <HealthLegend />
            {loading && services.length === 0 ? (
              <Flex justify="center" py={8}>
                <Spinner />
              </Flex>
            ) : servicesError !== null ? (
              <Text color="red.500" align="center" py={8}>
                {servicesError}
              </Text>
            ) : selectedStack !== null ? (
              <Suspense
                fallback={
                  <Flex justify="center" py={8}>
                    <Spinner />
                  </Flex>
                }
              >
                <StackGraph stackName={selectedStack} services={services} />
              </Suspense>
            ) : null}
          </ModalBody>
        </ModalContent>
      </Modal>
    </Container>
  )
}

export default App
