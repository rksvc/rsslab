import { Colors, FocusStyleManager, OverlaysProvider } from '@blueprintjs/core'
import { enableMapSet } from 'immer'
import ReactDOM from 'react-dom/client'
import App from './App.tsx'
import ContextProvider from './Context.tsx'
import '@blueprintjs/core/lib/css/blueprint.css'
import './styles.css'

const style = document.querySelector<HTMLHtmlElement>(':root')!.style
style.setProperty('--gray1', Colors.GRAY1)
style.setProperty('--gray4', Colors.GRAY4)
style.setProperty('--light-gray5', Colors.LIGHT_GRAY5)
style.setProperty('--dark-gray2', Colors.DARK_GRAY2)

enableMapSet()
FocusStyleManager.onlyShowFocusOnTabs()

ReactDOM.createRoot(document.getElementById('root')!).render(
  <OverlaysProvider>
    <ContextProvider>
      <App />
    </ContextProvider>
  </OverlaysProvider>,
)
