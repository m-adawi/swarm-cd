import { ChakraProvider, Container } from "@chakra-ui/react";
import { useState } from "react";
import StatusCard from "./components/StatusCard";
import useFetchStack from "./hooks/useFetchStacks";

interface StackStatus {
  Name: string;
  Error: string;
  Revision: string;
  RepoURL: string;
}

function App(): React.ReactElement {
  const [stacks, setStacks] = useState<StackStatus[]>([]);

  useFetchStack(setStacks);

  return (
    <ChakraProvider>
      <Container maxW="container.md" mt={4}>
        {stacks.map((item, index) => (
          <StatusCard
            key={index}
            name={item.Name}
            error={item.Error}
            revision={item.Revision}
            repoURL={item.RepoURL}
          />
        ))}
      </Container>
    </ChakraProvider>
  );
}

export default App;
