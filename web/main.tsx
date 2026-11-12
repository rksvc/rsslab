import { FocusStyleManager, OverlaysProvider } from '@blueprintjs/core'
import { StrictMode } from 'react'
import ReactDOM from 'react-dom/client'

import App from './App.tsx'
import ContextProvider from './Context.tsx'

import '@blueprintjs/core/lib/css/blueprint.css'
import './styles.css'

FocusStyleManager.onlyShowFocusOnTabs()

ReactDOM.createRoot(document.getElementById('root')!).render(
  <StrictMode>
    <OverlaysProvider>
      <ContextProvider>
        <App />
      </ContextProvider>
    </OverlaysProvider>
  </StrictMode>,
)
