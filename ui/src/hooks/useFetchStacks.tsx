import { useEffect } from "react";

// TODO move somewhere? A place for data types
interface StackStatus {
  Name: string;
  Error: string;
  Revision: string;
  RepoURL: string;
}

const devData: StackStatus[] = [
  {
    Name: "Project Alpha",
    Error: "",
    Revision: "v1.0.1",
    RepoURL: "https://github.com/user/project-alpha",
  },
  {
    Name: "Project Beta",
    Error: "Failed to build",
    Revision: "v2.3.4",
    RepoURL: "https://github.com/user/project-beta",
  },
  {
    Name: "Project Gamma",
    Error: "",
    Revision: "v0.9.8",
    RepoURL: "https://github.com/user/project-gamma",
  },
];

async function requestStacks(): Promise<StackStatus[]> {
  const response = await fetch("/stacks", {
    method: "GET",
    headers: {
      Accept: "application/json",
    },
  });
  const json = await response.json();
  return json as StackStatus[];
}

export default function useFetchStack(
  setStacks: (stacks: StackStatus[]) => void
): void {
  useEffect(() => {
    if (import.meta.env.MODE === "development") {
      setStacks(devData);
    } else {
      const interval = setInterval(() => {
        requestStacks()
          .then((stacks) => setStacks(stacks))
          .catch((err) => {
            console.error(err);
          });
      }, 5000);

      return () => {
        clearInterval(interval);
      };
    }
  });
}
