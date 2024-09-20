/// <reference types="vite/client" />

import { OverlaysProvider } from '@blueprintjs/core'
import '@blueprintjs/core/lib/css/blueprint.css'
import React from 'react'
import ReactDOM from 'react-dom/client'
import App from './App.tsx'
import './index.scss'

ReactDOM.createRoot(document.getElementById('root')!).render(
  <React.StrictMode>
    <OverlaysProvider>
      <App />
    </OverlaysProvider>
  </React.StrictMode>,
)
