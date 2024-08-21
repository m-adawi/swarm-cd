import { ChakraProvider, extendTheme, ThemeConfig } from "@chakra-ui/react"
import React from "react"
import ReactDOM from "react-dom/client"
import App from "./App.tsx"

const config: ThemeConfig = {
  initialColorMode: "system",
  useSystemColorMode: true
}

const theme = extendTheme({ config })

ReactDOM.createRoot(document.getElementById("root")!).render(
  <React.StrictMode>
    <ChakraProvider theme={theme}>
      <App />
    </ChakraProvider>
  </React.StrictMode>
)
