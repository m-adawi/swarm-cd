import { MoonIcon, SunIcon } from "@chakra-ui/icons"
import { IconButton, IconButtonProps, useColorMode } from "@chakra-ui/react"
import React from "react"

function ColorToggleButton(props: Readonly<Partial<IconButtonProps>>): React.ReactElement {
  const { colorMode, toggleColorMode } = useColorMode()

  const icon = colorMode === "light" ? <MoonIcon /> : <SunIcon />
  return <IconButton onClick={toggleColorMode} aria-label="Toggle color mode" icon={icon} {...props} />
}

export default ColorToggleButton
