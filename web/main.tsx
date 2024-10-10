/// <reference types="vite/client" />

import { OverlaysProvider } from '@blueprintjs/core'
import '@blueprintjs/core/lib/css/blueprint.css'
import ReactDOM from 'react-dom/client'
import App from './App.tsx'
import './index.scss'
import './styles.css'

ReactDOM.createRoot(document.getElementById('root')!).render(
  <OverlaysProvider>
    <App />
  </OverlaysProvider>,
)
