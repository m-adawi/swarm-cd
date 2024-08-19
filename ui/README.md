# SwarmCD UI

This repository contains the code for the Swarm-CD UI, a web-based user interface for checking the deployment status. The UI is built using React and bundled by Vite for fast and efficient development and build processes.

## Installation

1. Install [NodeJs](https://nodejs.org/en/download/package-manager)
2. Install dependencies `npm install`
3. To run the development UI server `npm run dev`
4. To build for production `npm run build`
   - The production build will be output to the `dist` directory.

## Directory structure

```
swarm-cd-ui/
├── public/                # Static assets
├── src/                   # Source files
│   ├── assets/            # Images, fonts, etc.
│   ├── components/        # React components
│   ├── App.tsx            # Main App component
│   ├── index.tsx          # Entry point
├── package.json           # Project metadata and scripts
└── vite.config.ts         # Vite configuration
```
