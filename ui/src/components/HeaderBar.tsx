import { Box, Flex, HStack, IconButton, Input, Link, Text, useColorModeValue } from "@chakra-ui/react"
import React from "react"
import { FaGithub } from "react-icons/fa"
import ColorToggleButton from "./ColorToggleButton"

function HeaderBar({
  onQueryChange,
  error
}: Readonly<{
  onQueryChange: (query: string) => void
  error: boolean
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
        <HeaderLinks />
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

function HeaderLinks(): React.ReactElement {
  return (
    <HStack>
      <Link href="https://github.com/m-adawi/swarm-cd/" isExternal>
        <IconButton aria-label="GitHub" icon={<FaGithub />} variant="ghost" isRound size="lg" />
      </Link>
      <ColorToggleButton variant="ghost" isRound size="lg" />
    </HStack>
  )
}

export default HeaderBar
