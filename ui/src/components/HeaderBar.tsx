import { Box, Flex, HStack, IconButton, Input, Link, Text, useColorModeValue } from "@chakra-ui/react"
import React from "react"
import { FaGithub, FaSync } from "react-icons/fa"
import ColorToggleButton from "./ColorToggleButton"

function HeaderBar({
  onQueryChange,
  error,
  hasUpdate,
  checkForUpdate,
  applyUpdate,
  isChecking
}: Readonly<{
  onQueryChange: (query: string) => void
  error: boolean
  hasUpdate: boolean
  checkForUpdate: () => Promise<void>
  applyUpdate: () => void
  isChecking: boolean
}>): React.ReactElement {
  return (
    <Box
      as="header"
      position="sticky"
      top="0"
      zIndex="1000"
      bg={useColorModeValue("gray.100", "gray.900")}
      boxShadow="sm"
      padding={4}
      mb={1}
    >
      <Flex justifyContent="space-between" alignItems="center">
        <Title />
        <SearchBar onQueryChange={onQueryChange} error={error} />
        <HeaderLinks
          hasUpdate={hasUpdate}
          checkForUpdate={checkForUpdate}
          applyUpdate={applyUpdate}
          isChecking={isChecking}
        />
      </Flex>
    </Box>
  )
}

function Title(): React.ReactElement {
  return (
    <HStack>
      <Text fontSize="xl" fontWeight="bold">
        SwarmCD
      </Text>
    </HStack>
  )
}

function SearchBar({
  onQueryChange,
  error
}: Readonly<{
  onQueryChange: (query: string) => void
  error: boolean
}>): React.ReactElement {
  return (
    <Box flex="1" mx={6}>
      <Input
        placeholder="Search..."
        onChange={event => onQueryChange(event.target.value)}
        size="lg"
        variant="filled"
        bg={useColorModeValue("gray.200", "gray.800")}
        disabled={error}
      />
    </Box>
  )
}

function RefreshButton({
  hasUpdate,
  checkForUpdate,
  applyUpdate,
  isChecking
}: Readonly<{
  hasUpdate: boolean
  checkForUpdate: () => Promise<void>
  applyUpdate: () => void
  isChecking: boolean
}>): React.ReactElement {
  const handleClick = (): void => {
    if (hasUpdate) {
      applyUpdate()
    } else {
      void checkForUpdate()
    }
  }

  const badgeBg = useColorModeValue("green.500", "green.400")

  return (
    <Box position="relative">
      <IconButton
        aria-label={hasUpdate ? "Apply update" : "Check for updates"}
        icon={<FaSync />}
        variant="ghost"
        isRound
        size="lg"
        onClick={handleClick}
        isLoading={isChecking}
        color={hasUpdate ? "green.500" : undefined}
        _hover={{
          transform: hasUpdate ? "scale(1.1)" : undefined
        }}
      />
      {hasUpdate && (
        <Box
          position="absolute"
          top="1"
          right="1"
          width="10px"
          height="10px"
          bg={badgeBg}
          borderRadius="full"
          animation="pulse 2s infinite"
          sx={{
            "@keyframes pulse": {
              "0%": { opacity: 1 },
              "50%": { opacity: 0.5 },
              "100%": { opacity: 1 }
            }
          }}
        />
      )}
    </Box>
  )
}

function HeaderLinks({
  hasUpdate,
  checkForUpdate,
  applyUpdate,
  isChecking
}: Readonly<{
  hasUpdate: boolean
  checkForUpdate: () => Promise<void>
  applyUpdate: () => void
  isChecking: boolean
}>): React.ReactElement {
  return (
    <HStack>
      <RefreshButton
        hasUpdate={hasUpdate}
        checkForUpdate={checkForUpdate}
        applyUpdate={applyUpdate}
        isChecking={isChecking}
      />
      <Link href="https://github.com/m-adawi/swarm-cd/" isExternal>
        <IconButton aria-label="GitHub" icon={<FaGithub />} variant="ghost" isRound size="lg" />
      </Link>
      <ColorToggleButton variant="ghost" isRound size="lg" />
    </HStack>
  )
}

export default HeaderBar
